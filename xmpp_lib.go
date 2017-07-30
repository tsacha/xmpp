package xmpp

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"strings"
)

const (
	nsStream   = "http://etherx.jabber.org/streams"
	nsClient   = "jabber:client"
	nsStartTLS = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL     = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsCaps     = "http://jabber.org/protocol/caps"
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
}

// RFC 3920  C.1  Streams name space
type streamStream struct {
	Stream  string
	Lang    string
	From    string
	Id      string
	Version string
	Xmlns   string
}

type streamFeatures struct {
	XMLName    xml.Name `xml:"http://etherx.jabber.org/streams features"`
	StartTLS   *tlsStartTLS
	Mechanisms saslMechanisms
	Bind       bindBind
	Session    bool
	Caps       *caps
}

// RFC 3920  C.3  TLS name space
type tlsStartTLS struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
	Required *string  `xml:"required"`
}

type tlsProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

// RFC 3920  C.4  SASL name space
type saslMechanisms struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
	Mechanism []string `xml:"mechanism"`
}

type saslChallenge string

type saslRspAuth string

type saslResponse string

type saslAbort struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl abort"`
}

type saslSuccess struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl success"`
}

type saslFailure struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl failure"`
	Any     xml.Name `xml:",any"`
	Text    string   `xml:"text"`
}

// RFC 3920  C.5  Resource binding name space
type bindBind struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	Resource string
	Jid      string `xml:"jid"`
}

// XEP 0115 Entity
type caps struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/caps c"`
	Ext     string   `xml:"ext,attr"`  // DEPRECATED
	Hash    string   `xml:"hash,attr"` // REQUIRED
	Node    string   `xml:"node,attr"` // REQUIRED
	Ver     string   `xml:"ver,attr"`  // REQUIRED
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
	server_choice := rand.Intn(len(addrs))

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
	default:
		return (incomingResult{xml.Name{}, nil, errors.New("unexpected XMPP message " +
			se.Name.Space + " <" + se.Name.Local + "/>")})
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

	conf := &tls.Config{
		ServerName: domain,
	}

	// TLS Handshake
	logrus.Info("TLS Handshake")
	t := tls.Client(conn, conf)
	err = t.Handshake()
	LogError(err, "TLS Handshake")

	xmppconn.reader = xml.NewDecoder(t)
	xmppconn.writer = bufio.NewWriter(t)

	xmppconn.StartStream(domain)
}

func create_user_hash(account string, password string) []byte {
	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	return enc
}

func (xmppconn *XMPPConnection) AuthenticateUser(account string, password string) {
	hash := create_user_hash(account, password)

	auth_request := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>%s</auth>",
		nsSASL, hash)
	auth_request_anonymous := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>********</auth>",
		nsSASL)

	logrus.WithFields(logrus.Fields{
		"account": account,
	}).Info("Authentication")
	LogInOut("out", auth_request_anonymous)

	xmppconn.outgoing <- auth_request

	auth_result := <-xmppconn.incoming

	switch t := auth_result.Interface.(type) {
	case *saslSuccess:
		logrus.Info("Authenticated")
	case *saslFailure:
		logrus.Error("Authentication failure : " + t.Text)
	default:
		logrus.Error("Authentication failure : XML error")
	}
}
