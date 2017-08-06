// XEP 0115 — Entity
package xmpp

import (
	"encoding/xml"
)

// XEP 0115 # 1.2 — How it works
type Caps struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/caps c"`
	Ext     string   `xml:"ext,attr"`  // DEPRECATED
	Hash    string   `xml:"hash,attr"` // REQUIRED
	Node    string   `xml:"node,attr"` // REQUIRED
	Ver     string   `xml:"ver,attr"`  // REQUIRED
}
