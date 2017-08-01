// XEP 0198 — Stream Management
package xmpp

import (
	"encoding/xml"
)

// XEP 0198 # 2 — Stream Feature
type sm struct {
	XMLName  xml.Name `xml:"sm"`
	Optional string   `xml:"optional"`
}

// XEP 0198 # 3 — Enabling Stream Management
type enabled struct {
	XMLName xml.Name `xml:"enabled"`
	Resume  string   `xml:"resume,attr"`
	Id      string   `xml:"id,attr"`
}
