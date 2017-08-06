package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"strconv"
)

type Ver struct {
	XMLName xml.Name `xml:"urn:xmpp:features:rosterver ver"`
}

type Item struct {
	XMLName      xml.Name `xml:"item"`
	Jid          string   `xml:"jid,attr"`
	Subscription string   `xml:"subscription,attr"`
	Name         string   `xml:"name,attr"`
	Group        string   `xml:"group"`
}

type Contact struct {
	Name         string `json:"name"`
	Jid          string `json:"jid"`
	Group        string `json:"group"`
	Subscription string `json:"subscription"`
}

type RosterConfig struct {
	version_supported bool
	incoming          chan *clientIQ
	Contacts          []*Contact
}

func (xmpp *XMPPConnection) GetRoster() {
	xmpp.State.Roster.incoming = make(chan *clientIQ)

	query := &query{XMLName: xml.Name{Local: "query", Space: nsRoster}}
	query_disco_id := strconv.FormatUint(uint64(get_cookie()), 10)
	query_disco := &clientIQ{
		Type:  "get",
		ID:    query_disco_id,
		From:  xmpp.State.Jid,
		Query: query,
	}
	output, _ := xml.Marshal(query_disco)

	logrus.Info("[RFC 6121] Retrieving roster…")
	xmpp.outgoing <- string(output)

	result := <-xmpp.State.Roster.incoming
	xmpp.State.Roster.Contacts = make([]*Contact, 0)

	for _, item := range result.Query.Items {
		logrus.WithFields(logrus.Fields{
			"name":         item.Name,
			"jid":          item.Jid,
			"group":        item.Group,
			"subscription": item.Subscription,
		}).Info("[RFC 6121] Found roster item : ")
		c := &Contact{
			Name:         item.Name,
			Jid:          item.Jid,
			Group:        item.Group,
			Subscription: item.Subscription,
		}
		xmpp.State.Roster.Contacts = append(xmpp.State.Roster.Contacts, c)
	}
}
