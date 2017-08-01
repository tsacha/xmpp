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
	"strings"
)

const (
	nsStream       = "http://etherx.jabber.org/streams"
	nsClient       = "jabber:client"
	nsStartTLS     = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL         = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsCaps         = "http://jabber.org/protocol/caps"
	nsBind         = "urn:ietf:params:xml:ns:xmpp-bind"
	nsStreamMgmtv2 = "urn:xmpp:sm:2"
	nsStreamMgmtv3 = "urn:xmpp:sm:3"
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
	conf     XMPPConf
}

type XMPPConf struct {
	stream          int
	stream_optional bool
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
				stream.Id = attr.Value
			case "version":
				stream.Version = attr.Value
			case "xmlns":
				stream.Xmlns = attr.Value
			}
		}
		logrus.WithFields(logrus.Fields{
			"stream":  stream.Stream,
			"lang":    stream.Lang,
			"id":      stream.Id,
			"version": stream.Version,
			"xmlns":   stream.Xmlns,
		}).Info("Received stream for server")
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
	default:
		fmt.Println(se.Name.Space)
		return (incomingResult{xml.Name{}, nil, errors.New("unexpected XMPP message " +
			se.Name.Space + " " + se.Name.Local)})
	}

	// Unmarshal into that storage.
	err = xmppconn.reader.DecodeElement(nv, &se)
	if err != nil {
		return incomingResult{xml.Name{}, nil, err}
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
	starttls_request := "<starttls xmlns='" + nsStartTLS + "'/>"
	xmppconn.outgoing <- starttls_request

	// <proceed>
	xmppconn.NextElement()

	f, err := os.Create("/home/sacha/lol.txt")
	defer f.Close()

	w := bufio.NewWriter(f)

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

	auth_request := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>%s</auth>",
		nsSASL, hash)
	logrus.WithFields(logrus.Fields{
		"account": account,
	}).Info("Authentication")

	xmppconn.outgoing <- auth_request

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
				if attr.XMLName.Space == nsStreamMgmtv3 {
					xmppconn.conf.stream = int(math.Max(float64(xmppconn.conf.stream), 3.0))
				} else if attr.XMLName.Space == nsStreamMgmtv2 {
					xmppconn.conf.stream = int(math.Max(float64(xmppconn.conf.stream), 2.0))
				}
				if attr.Optional == "" {
					xmppconn.conf.stream_optional = true
				}
			}
		}

	case *saslFailure:
		logrus.Error("Authentication failure : " + t.Text)
	default:
		logrus.Error("Authentication failure : XML error")
	}
}
