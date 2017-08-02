package xmpp

import (
	"encoding/xml"
)

// RFC 6120 # 4.1 — Stream Fundamentals
type clientIQ struct {
	XMLName xml.Name `xml:"jabber:client iq"`
	From    string   `xml:"from,attr,omitempty"`
	ID      string   `xml:"id,attr"`
	To      string   `xml:"to,attr,omitempty"`
	Type    string   `xml:"type,attr"`
	Query   []byte   `xml:",innerxml"`
	Bind    *bind    `xml:"bind,omitempty"`
	Session *session `xml:"session,omitempty"`
}

// RFC 6120 # 4.3.2 — Stream features
// List of features: https://xmpp.org/registrar/stream-features.html
type streamFeatures struct {
	XMLName    xml.Name `xml:"http://etherx.jabber.org/streams features"`
	StartTLS   *tlsStartTLS
	Mechanisms saslMechanisms
	Bind       bind
	Session    bool
	Sms        []sm `xml:"sm"` // XEP 0198
	Caps       caps // XEP 0115
}

// RFC 6120  # 4.7 — Stream Attributes
type streamStream struct {
	Stream  string
	Lang    string
	From    string
	To      string
	ID      string
	Version string
	Xmlns   string
}

// RFC 6120 # 5.4.3 — TLS Negociation
type tlsStartTLS struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls starttls"`
	Required *string  `xml:"required,omitempty"`
}

type tlsProceed struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-tls proceed"`
}

// RFC 6120  # 6.4.1 — Exchange of Stream Headers and Stream Features
type saslMechanisms struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl mechanisms"`
	Mechanism []string `xml:"mechanism"`
}

// RFC 6120  # 6.4.2 — Initiation
type saslAuth struct {
	XMLName   xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-sasl auth"`
	Auth      string   `xml:",chardata"`
	Mechanism string   `xml:"mechanism,attr"`
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
type bind struct {
	XMLName  xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
	Resource string   `xml:"resource,omitempty"`
	Jid      string   `xml:"jid,omitempty"`
}
