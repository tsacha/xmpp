package xmpp

import (
	"encoding/xml"
)

// https://xmpp.org/registrar/stream-features.html
type session struct {
	XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-session session"`
}
