// XEP 0199 — XMPP Ping
package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// XEP 0199 # 3 — Protocol
type ping struct {
	XMLName xml.Name `xml:"urn:xmpp:ping ping"`
}

type PingConfig struct {
	incoming chan string
}

func (xmppconn *XMPPConnection) Ping() {
	xmppconn.State.Ping = &PingConfig{
		incoming: make(chan string),
	}

	id_ping := strconv.FormatUint(uint64(get_cookie()), 10)
	iq_ping := &clientIQ{
		Type: "get",
		ID:   id_ping,
		From: xmppconn.State.Jid,
		Ping: &ping{},
	}
	output, _ := xml.Marshal(iq_ping)
	xmppconn.outgoing <- string(output)
	logrus.WithFields(logrus.Fields{
		"id": id_ping,
	}).Info("[XEP 0199] Ping")
	for {
		ping_id := <-xmppconn.State.Ping.incoming
		if ping_id == id_ping {
			logrus.WithFields(logrus.Fields{
				"id": ping_id,
			}).Info("[XEP 0199] Pong")
			return
		}
	}
}

func (xmppconn *XMPPConnection) InfinitePing() {
	for {
		xmppconn.Ping()
		time.Sleep(2 * time.Second)
	}
}
