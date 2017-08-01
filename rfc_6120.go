package xmpp

import (
	"encoding/xml"
)

// RFC 6120 # 4.1 — Stream Fundamentals
type clientIQ struct {
	XMLName xml.Name `xml:"jabber:client iq"`
	From    string   `xml:"from,attr"`
	ID      string   `xml:"id,attr"`
	To      string   `xml:"to,attr"`
	Type    string   `xml:"type,attr"`
	Query   []byte   `xml:",innerxml"`
	Error   clientError
	Bind    bindBind
}

// RFC 6120 # 4.3.2 — Stream features
type streamFeatures struct {
	XMLName    xml.Name `xml:"http://etherx.jabber.org/streams features"`
	StartTLS   *tlsStartTLS
	Mechanisms saslMechanisms
	Bind       bindBind
	Session    bool
	Sms        []sm `xml:"sm"` // XEP 0198
	Caps       *caps
}

// RFC 6120  # 4.7 — Stream Attributes
type streamStream struct {
	Stream  string
	Lang    string
	From    string
	To      string
	Id      string
	Version string
	Xmlns   string
}

// RFC 6120 # 4.9 — Stream Errors
type clientError struct {
	XMLName xml.Name `xml:"jabber:client error"`
	Code    string   `xml:",attr"`
	Type    string   `xml:",attr"`
	Any     xml.Name
	Text    string
}

// RFC 6120  # 5.4.3 — TLS Negociation
type tlsStartTLS struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
	Required *string  `xml:"required"`
}

type tlsProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

// RFC 6120  # 6 — SASL Negociation
type saslMechanisms struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
	Mechanism []string `xml:"mechanism"`
}

// RFC 6120 # 6.4.4 — Abort
type saslAbort struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl abort"`
}

// RFC 6120 # 6.4.5 — Failure
type saslFailure struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl failure"`
	Any     xml.Name `xml:",any"`
	Text    string   `xml:"text"`
}

// RFC 6120 # 6.4.6 — Success
type saslSuccess struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl success"`
}

// RFC 6120  # 9.1.3 — Resource Binding
type bindBind struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	Resource string
	Jid      string `xml:"jid"`
}
