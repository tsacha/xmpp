package xmpp

import (
	"bufio"
	"crypto/tls"
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"net"
	"os"
)

func (xmpp *XMPPConnection) EncryptConnection(domain string, conn net.Conn) {
	starttls := &tlsStartTLS{}
	output, _ := xml.Marshal(starttls)
	xmpp.outgoing <- string(output)

	f, err := os.Create("/home/sacha/lol.txt")
	defer f.Close()

	w := bufio.NewWriter(f)

	// <proceed>
	xmpp.NextElement()

	conf := &tls.Config{
		ServerName:   domain,
		KeyLogWriter: w,
	}

	// TLS Handshake
	logrus.Info("TLS Handshake")
	t := tls.Client(conn, conf)
	err = t.Handshake()
	LogError(err, "TLS Handshake")

	xmpp.reader = xml.NewDecoder(teeIn{t})
	xmpp.writer = bufio.NewWriter(teeOut{t})

	xmpp.StartStream(domain)

	w.Flush()
}
