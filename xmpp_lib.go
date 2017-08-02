package xmpp

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	mathrand "math/rand"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	nsStream       = "http://etherx.jabber.org/streams"
	nsClient       = "jabber:client"
	nsStartTLS     = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL         = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsCaps         = "http://jabber.org/protocol/caps"
	nsBind         = "urn:ietf:params:xml:ns:xmpp-bind"
	nsSession      = "urn:ietf:params:xml:ns:xmpp-session"
	nsStreamMgmtv2 = "urn:xmpp:sm:2"
	nsStreamMgmtv3 = "urn:xmpp:sm:3"
	nsDiscoInfo    = "http://jabber.org/protocol/disco#info"
)

type incomingResult struct {
	XMLName   xml.Name
	Interface interface{}
	Error     error
}

type XMPPConnection struct {
	incoming chan incomingResult
	outgoing chan string
	reader   *xml.Decoder
	writer   *bufio.Writer
	conn     net.Conn
	state    XMPPState
}

type XMPPState struct {
	jid      string
	resource string
	sm       *StreamManagementConfig
	session  *SessionConfig
	ping     *PingConfig
}

// Cookie is a unique XMPP session identifier
type Cookie uint64

func getCookie() Cookie {
	var buf [8]byte
	if _, err := rand.Reader.Read(buf[:]); err != nil {
		panic("Failed to read random bytes: " + err.Error())
	}
	return Cookie(binary.LittleEndian.Uint64(buf[:]))
}

var err error

func (xmppconn *XMPPConnection) Read() {
	for {
		if xmppconn.reader != nil {
			t, _ := xmppconn.reader.Token()
			switch t := t.(type) {
			case xml.StartElement:
				xmppconn.incoming <- xmppconn.next(t)
			}
		} else {
			return
		}
	}
}

func (xmppconn *XMPPConnection) Write() {
	for {
		xmppconn.writer.WriteString(<-xmppconn.outgoing)
		xmppconn.writer.Flush()
		if xmppconn.state.sm != nil && xmppconn.state.sm.state {
			xmppconn.state.sm.seq += 1
			xmppconn.state.sm.output <- 1
		}
	}
}

func (xmppconn *XMPPConnection) Listen() {
	go xmppconn.Write()
	go xmppconn.Read()
}

func resolvServer(account string) (string, string, string) {
	domain := strings.Split(account, "@")[1]
	_, addrs, _ := net.LookupSRV("xmpp-client", "tcp", domain)

	// Random choice between SRV records
	server_choice := mathrand.Intn(len(addrs))

	logrus.WithFields(logrus.Fields{
		"nb_entries": len(addrs),
		"domain":     domain,
		"addr":       addrs[server_choice].Target,
		"port":       addrs[server_choice].Port,
	}).Info("Resolve XMPP server")
	return addrs[server_choice].Target, fmt.Sprint(addrs[server_choice].Port), domain
}

func ConnectServer(addr string, port string) net.Conn {
	logrus.WithFields(logrus.Fields{
		"addr": addr,
		"port": port,
	}).Info("TCP Connection")
	conn, err := net.Dial("tcp", addr+":"+port)
	LogError(err, "Error while initializing TCP connection")

	return conn
}

func (xmppconn *XMPPConnection) next(se xml.StartElement) incomingResult {
	var nv interface{}

	switch se.Name.Space + " " + se.Name.Local {
	case nsStream + " stream":
		var stream streamStream
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "stream":
				stream.Stream = attr.Value
			case "lang":
				stream.Lang = attr.Value
			case "id":
				stream.ID = attr.Value
			case "version":
				stream.Version = attr.Value
			case "xmlns":
				stream.Xmlns = attr.Value
			}
		}
		logrus.WithFields(logrus.Fields{
			"stream":  stream.Stream,
			"lang":    stream.Lang,
			"id":      stream.ID,
			"version": stream.Version,
			"xmlns":   stream.Xmlns,
		}).Info("Received stream from server")
		return (incomingResult{se.Name, stream, nil})
	case nsStream + " features":
		nv = &streamFeatures{}
	case nsStartTLS + " proceed":
		nv = &tlsProceed{}
	case nsSASL + " success":
		nv = &saslSuccess{}
	case nsSASL + " failure":
		nv = &saslFailure{}
	case nsClient + " iq":
		nv = &clientIQ{}
	case nsStreamMgmtv2 + " enabled":
		nv = &enabled{}
	case nsStreamMgmtv3 + " enabled":
		nv = &enabled{}
	case nsStreamMgmtv3 + " a":
		nv = &answer{}
	case nsStreamMgmtv3 + " r":
		nv = &request{}
	default:
		return (incomingResult{xml.Name{}, nil, errors.New("unexpected XMPP message " +
			se.Name.Space + " " + se.Name.Local)})
	}

	// Unmarshal into that storage.
	err = xmppconn.reader.DecodeElement(nv, &se)
	if err != nil {
		return incomingResult{xml.Name{}, nil, err}
	}

	// If stream management is active
	if xmppconn.state.sm != nil && xmppconn.state.sm.state {
		// Do not count namespace stream management
		if se.Name.Space != nsStreamMgmtv3 {
			xmppconn.state.sm.handled += 1
		}
	}
	return incomingResult{se.Name, nv, err}
}

func (xmppconn *XMPPConnection) NextElement() incomingResult {
	var nv incomingResult
	t, _ := xmppconn.reader.Token()

	switch t := t.(type) {
	case xml.ProcInst:
		logrus.Info("Received XML from server")
		return (incomingResult{xml.Name{}, nil, nil})
	case xml.StartElement:
		nv = xmppconn.next(t)
	}
	return nv
}

func (xmppconn *XMPPConnection) StartStream(domain string) {
	// Stream request
	stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
		"<stream:stream to='%s' xmlns='%s'"+
		" xmlns:stream='%s' version='1.0'>",
		domain, nsClient, nsStream)

	logrus.Info("Send stream request")
	xmppconn.outgoing <- stream_request

	// XML ProcInst
	xmppconn.NextElement()

	// <stream>
	xmppconn.NextElement()

	// <features>
	xmppconn.NextElement()
}

func (xmppconn *XMPPConnection) EncryptConnection(domain string, conn net.Conn) {
	starttls := &tlsStartTLS{}
	output, _ := xml.Marshal(starttls)
	xmppconn.outgoing <- string(output)

	f, err := os.Create("/home/sacha/lol.txt")
	defer f.Close()

	w := bufio.NewWriter(f)

	// <proceed>
	xmppconn.NextElement()

	conf := &tls.Config{
		ServerName:   domain,
		KeyLogWriter: w,
	}

	// TLS Handshake
	logrus.Info("TLS Handshake")
	t := tls.Client(conn, conf)
	err = t.Handshake()
	LogError(err, "TLS Handshake")

	xmppconn.reader = xml.NewDecoder(teeIn{t})
	xmppconn.writer = bufio.NewWriter(teeOut{t})

	xmppconn.StartStream(domain)

	w.Flush()
}

func create_user_hash(account string, password string) []byte {
	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	return enc
}

func (xmppconn *XMPPConnection) AuthenticateUser(account string, password string, domain string) {
	hash := create_user_hash(account, password)
	auth := &saslAuth{Mechanism: "PLAIN", Auth: string(hash)}
	output, _ := xml.Marshal(auth)

	logrus.WithFields(logrus.Fields{
		"account": account,
	}).Info("Authentication")

	xmppconn.outgoing <- string(output)
	auth_result := <-xmppconn.incoming

	switch t := auth_result.Interface.(type) {
	case *saslSuccess:
		logrus.Info("Authenticated, request new stream")

		stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
			"<stream:stream to='%s' xmlns='%s'"+
			" xmlns:stream='%s' version='1.0'>",
			domain, nsClient, nsStream)

		xmppconn.outgoing <- stream_request

		// <stream>
		<-xmppconn.incoming

		// <features>
		features := <-xmppconn.incoming
		switch t := features.Interface.(type) {
		case *streamFeatures:
			for _, attr := range t.Sms {
				if attr.XMLName.Space == nsStreamMgmtv3 || attr.XMLName.Space == nsStreamMgmtv2 {
					xmppconn.state.sm = &StreamManagementConfig{}
				}
				if attr.XMLName.Space == nsStreamMgmtv3 {
					xmppconn.state.sm.version = int(math.Max(float64(xmppconn.state.sm.version), 3.0))
				} else if attr.XMLName.Space == nsStreamMgmtv2 {
					xmppconn.state.sm.version = int(math.Max(float64(xmppconn.state.sm.version), 2.0))
				}
				if attr.Optional == "" {
					xmppconn.state.sm.optional = true
				}
			}
		}

	case *saslFailure:
		logrus.Error("Authentication failure : " + t.Text)
	default:
		logrus.Error("Authentication failure : XML error")
	}
}

func (xmppconn *XMPPConnection) Bind(resource string) {
	id_bind := strconv.FormatUint(uint64(getCookie()), 10)
	bind := &bind{Resource: resource}
	iq_bind := &clientIQ{
		Type: "set",
		ID:   id_bind,
		Bind: bind,
	}
	output, _ := xml.Marshal(iq_bind)

	logrus.WithFields(logrus.Fields{
		"resource": resource,
		"id":       id_bind,
	}).Info("Binding to resource")

	xmppconn.outgoing <- string(output)
	iq_response := <-xmppconn.incoming

	switch t := iq_response.Interface.(type) {
	case *clientIQ:
		if t.ID == id_bind {
			logrus.WithFields(logrus.Fields{
				"resource": resource,
				"jid":      t.Bind.Jid,
				"id":       t.ID,
			}).Info("Bound")
			xmppconn.state.jid = t.Bind.Jid
			xmppconn.state.resource = resource
		}
	}
}

func (xmppconn *XMPPConnection) StartSession() {
	xmppconn.state.session = &SessionConfig{
		incoming: make(chan string),
	}

	id_session := strconv.FormatUint(uint64(getCookie()), 10)
	iq_session := &clientIQ{
		Type:    "set",
		ID:      id_session,
		From:    xmppconn.state.jid,
		Session: &session{},
	}
	output, _ := xml.Marshal(iq_session)

	logrus.Info("Starting session")
	xmppconn.outgoing <- string(output)

	for {
		session_id := <-xmppconn.state.session.incoming
		if session_id == id_session {
			logrus.WithFields(logrus.Fields{
				"id": session_id,
			}).Info("Session started")
			xmppconn.state.session.state = true
			return
		}
	}
}

func (xmppconn *XMPPConnection) Process() {
	for {
		t := <-xmppconn.incoming
		switch t := (t.Interface).(type) {
		case *request:
			if xmppconn.state.sm != nil && xmppconn.state.sm.state {
				// Stream Management: answer to server request
				xmppconn.state.sm.input <- 1
			}
		case *answer:
			if xmppconn.state.sm != nil && xmppconn.state.sm.state {
				// Stream Management: verify answer from server
				xmppconn.state.sm.verify <- t.Handled
			}
		case *clientIQ:
			if t.Type == "result" {
				// Session initiated
				if xmppconn.state.session != nil && !xmppconn.state.session.state {
					xmppconn.state.session.incoming <- t.ID
				}
				// Pong
				if xmppconn.state.ping != nil {
					xmppconn.state.ping.incoming <- t.ID
				}
			}
		}
	}
}

func (xmppconn *XMPPConnection) Disco() {
	query := &query{XMLName: xml.Name{Local: "query", Space: nsDiscoInfo}}
	query_disco_id := strconv.FormatUint(uint64(getCookie()), 10)
	query_disco := &clientIQ{
		Type:  "get",
		ID:    query_disco_id,
		From:  xmppconn.state.jid,
		Query: query,
	}
	output, _ := xml.Marshal(query_disco)

	logrus.Info("Starting discovery…")
	xmppconn.outgoing <- string(output)
}
