package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"strconv"
)

type ver struct {
	XMLName xml.Name `xml:"urn:xmpp:features:rosterver ver"`
}

type RosterConfig struct {
	version_supported bool
}

func (xmppconn *XMPPConnection) GetRoster() {
	query := &query{XMLName: xml.Name{Local: "query", Space: nsRoster}}
	query_disco_id := strconv.FormatUint(uint64(getCookie()), 10)
	query_disco := &clientIQ{
		Type:  "get",
		ID:    query_disco_id,
		From:  xmppconn.state.jid,
		Query: query,
	}
	output, _ := xml.Marshal(query_disco)

	logrus.Info("[RFC 6121] Retrieving roster…")
	xmppconn.outgoing <- string(output)
}
