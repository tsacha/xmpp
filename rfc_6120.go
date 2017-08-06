package xmpp

import (
	"encoding/xml"
	"fmt"
	"github.com/sirupsen/logrus"
	"strconv"
)

// RFC 6120 # 4.1 — Stream Fundamentals
type clientIQ struct {
	XMLName xml.Name `xml:"jabber:client iq"`
	From    string   `xml:"from,attr,omitempty"`
	ID      string   `xml:"id,attr"`
	To      string   `xml:"to,attr,omitempty"`
	Type    string   `xml:"type,attr"`
	Query   *query   `xml:"query,omitempty"`
	Bind    *bind    `xml:"bind,omitempty"`
	Ping    *ping    `xml:"ping,omitempty"`
}

type query struct {
	XMLName    xml.Name
	Identities [](*Identity) `xml:"identity,omitempty"`
	Features   [](*Feature)  `xml:"feature,omitempty"`
	Items      [](*Item)     `xml:"item,omitempty"`
}

// RFC 6120 # 4.3.2 — Stream features
// List of features: https://xmpp.org/registrar/stream-features.html
type streamFeatures struct {
	XMLName    xml.Name          `xml:"http://etherx.jabber.org/streams features"`
	StartTLS   *tlsStartTLS      `xml:"starttls"`
	Mechanisms *saslMechanisms   `xml:"mechanisms"`
	Bind       *bind             `xml:"bind"`
	Sms        [](*streamMgmtSm) `xml:"sm"`  // XEP 0198
	Caps       *Caps             `xml:"c"`   // XEP 0115
	Ver        *Ver              `xml:"ver"` // RFC 6121
	Csi        *Csi              `xml:"csi"` // XEP 0352
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

func (xmpp *XMPPConnection) Bind(resource string) {
	id_bind := strconv.FormatUint(uint64(get_cookie()), 10)
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

	xmpp.outgoing <- string(output)
	iq_response := <-xmpp.incoming

	switch t := iq_response.Interface.(type) {
	case *clientIQ:
		if t.ID == id_bind {
			logrus.WithFields(logrus.Fields{
				"resource": resource,
				"jid":      t.Bind.Jid,
				"id":       t.ID,
			}).Info("Bound")
			xmpp.State.Jid = t.Bind.Jid
			xmpp.State.Resource = resource
		}
	}
}

func (xmpp *XMPPConnection) StartStream(domain string) {
	// Stream request
	stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
		"<stream:stream to='%s' xmlns='%s'"+
		" xmlns:stream='%s' version='1.0'>",
		domain, nsClient, nsStream)

	logrus.Info("Send stream request")
	xmpp.outgoing <- stream_request

	// <stream>
	xmpp.NextElement()

	// <features>
	xmpp.NextElement()
}

func (xmpp *XMPPConnection) AuthenticateUser(account string, password string, domain string) {
	hash := create_user_hash(account, password)
	auth := &saslAuth{Mechanism: "PLAIN", Auth: string(hash)}
	output, _ := xml.Marshal(auth)

	logrus.WithFields(logrus.Fields{
		"account": account,
	}).Info("Authentication")

	xmpp.outgoing <- string(output)
	auth_result := <-xmpp.incoming

	switch t := auth_result.Interface.(type) {
	case *saslSuccess:
		logrus.Info("Authenticated, request new stream")

		stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
			"<stream:stream to='%s' xmlns='%s'"+
			" xmlns:stream='%s' version='1.0'>",
			domain, nsClient, nsStream)

		xmpp.outgoing <- stream_request

		// <stream>
		<-xmpp.incoming

		// <features>
		features := <-xmpp.incoming
		switch t := features.Interface.(type) {
		case *streamFeatures:
			xmpp.State.Roster = &RosterConfig{}
			if t.Ver != nil && t.Ver.XMLName.Space == nsRosterVer {
				xmpp.State.Roster.version_supported = true
			}

			for _, attr := range t.Sms {
				if attr.XMLName.Space == nsStreamMgmt {
					xmpp.State.Sm = &StreamManagementConfig{}
				}
				if attr.XMLName.Space == nsStreamMgmt {
					xmpp.State.Sm.version = 3
				}
			}
		}

	case *saslFailure:
		logrus.Error("Authentication failure : " + t.Text)
		xmpp.Close()
	default:
		logrus.Error("Authentication failure : XML error")
		xmpp.Close()
	}
}
