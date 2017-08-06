// XEP 0030 — Service Discovery
package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"strconv"
)

type DiscoveryConfig struct {
	incoming chan *clientIQ
}

// XEP 0030 # 3.1 — Basic Protocol
type Identity struct {
	XMLName  xml.Name `xml:"identity"`
	Type     string   `xml:"type,attr"`
	Name     string   `xml:"name,attr"`
	Category string   `xml:"category,attr"`
}

type Feature struct {
	XMLName xml.Name `xml:"feature"`
	Var     string   `xml:"var,attr"`
}

func (xmppconn *XMPPConnection) Disco(to string) {
	xmppconn.State.Discovery = &DiscoveryConfig{
		incoming: make(chan *clientIQ),
	}

	query := &query{XMLName: xml.Name{Local: "query", Space: nsDiscoInfo}}
	query_disco_id := strconv.FormatUint(uint64(get_cookie()), 10)
	query_disco := &clientIQ{
		Type:  "get",
		ID:    query_disco_id,
		From:  xmppconn.State.Jid,
		To:    to,
		Query: query,
	}
	output, _ := xml.Marshal(query_disco)

	logrus.Info("[XEP 0030] Starting discovery on " + to + "…")
	xmppconn.outgoing <- string(output)

	for {
		response := <-xmppconn.State.Discovery.incoming
		logrus.Info("[XEP 0030] Received discovery response for " + to)

		for _, attr := range response.Query.Features {
			switch attr.Var {
			case nsPing:
				logrus.Info("[XEP 0030] ✔ XMPP Ping (XEP-0199)")
			case nsLastActivity:
				logrus.Info("[XEP 0030] ✔ Last Activity (XEP-0012)")
			case nsCommands:
				logrus.Info("[XEP 0030] ✔ Ad-Hoc Commands (XEP-0050)")
			case nsBlocking:
				logrus.Info("[XEP 0030] ✔ Blocking Command (XEP-0191)")
			case nsMam:
				logrus.Info("[XEP 0030] ✔ Message Archive Management (XEP-0313)")
			case nsPush:
				logrus.Info("[XEP 0030] ✔ Push Notifications (XEP-0357)")
			case nsUniqueStanza:
				logrus.Info("[XEP 0030] ✔ Unique and Stable Stanza IDs (XEP-0359)")
			case nsPubSubPublish:
				logrus.Info("[XEP 0030] ✔ Publish-Subscribe (Publishing items) (XEP-0060)")
			case nsOffline:
				logrus.Info("[XEP 0030] ✔ Handling Offline Messages (XEP-0160)")
			case nsVcard:
				logrus.Info("[XEP 0030] ✔ vCard XML (XEP-0054)")
			case nsRoster:
				logrus.Info("[XEP 0030] ✔ Roster (RFC 3921)")
			case nsVersion:
				logrus.Info("[XEP 0030] ✔ Sofware Version (XEP-0092)")
			case nsTime:
				logrus.Info("[XEP 0030] ✔ Entity Time (XEP-0202)")
			case nsPrivate:
				logrus.Info("[XEP 0030] ✔ Private XML Storage (XEP-0049)")
			case nsRegister:
				logrus.Info("[XEP 0030] ✔ In-Band Registration (XEP-0077)")
			case nsDiscoInfo:
				logrus.Info("[XEP 0030] ✔ Service Discovery — Info (XEP-0030)")
			case nsDiscoItems:
				logrus.Info("[XEP 0030] ✔ Service Discovery — Items (XEP-0030)")
			case nsCarbons:
				logrus.Info("[XEP 0030] ✔ Message Carbons (XEP-0280)")
			default:
				logrus.Info("[XEP 0030] ✘ Unknown feature (" + attr.Var + ")")
			}
		}

		for _, attr := range response.Query.Identities {
			logrus.WithFields(logrus.Fields{
				"type":     attr.Type,
				"name":     attr.Name,
				"category": attr.Category,
			}).Info("[XEP 0030] Found identity")
		}
		return
	}
}
