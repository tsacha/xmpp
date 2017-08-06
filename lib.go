package xmpp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/sirupsen/logrus"
	mathrand "math/rand"
	"net"
	"strings"
)

const (
	nsStream        = "http://etherx.jabber.org/streams"
	nsClient        = "jabber:client"
	nsStartTLS      = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL          = "urn:ietf:params:xml:ns:xmpp-sasl"
	nsCaps          = "http://jabber.org/protocol/caps"
	nsBind          = "urn:ietf:params:xml:ns:xmpp-bind"
	nsPing          = "urn:xmpp:ping"
	nsBlocking      = "urn:xmpp:blocking"
	nsStreamMgmt    = "urn:xmpp:sm:3"
	nsMam           = "urn:xmpp:mam:2"
	nsUniqueStanza  = "urn:xmpp:sid:0"
	nsPush          = "urn:xmpp:push:0"
	nsTime          = "urn:xmpp:time"
	nsCarbons       = "urn:xmpp:carbons:2"
	nsLastActivity  = "jabber:iq:last"
	nsVersion       = "jabber:iq:version"
	nsRoster        = "jabber:iq:roster"
	nsRosterVer     = "urn:xmpp:features:rosterver"
	nsPrivate       = "jabber:iq:private"
	nsRegister      = "jabber:iq:register"
	nsOffline       = "msgoffline"
	nsVcard         = "vcard-temp"
	nsCommands      = "http://jabber.org/protocol/commands"
	nsDiscoInfo     = "http://jabber.org/protocol/disco#info"
	nsDiscoItems    = "http://jabber.org/protocol/disco#items"
	nsPubSubPublish = "http://jabber.org/protocol/pubsub#publish"
)

func resolv_server(account string) (string, string, string) {
	domain := strings.Split(account, "@")[1]
	_, addrs, _ := net.LookupSRV("xmpp-client", "tcp", domain)

	// Random choice between SRV records
	server_choice := mathrand.Intn(len(addrs))

	logrus.WithFields(logrus.Fields{
		"nb_entries": len(addrs),
		"domain":     domain,
		"addr":       addrs[server_choice].Target,
		"port":       addrs[server_choice].Port,
	}).Info("Resolve XMPP server")
	return addrs[server_choice].Target, fmt.Sprint(addrs[server_choice].Port), domain
}

func connect_server(addr string, port string) net.Conn {
	logrus.WithFields(logrus.Fields{
		"addr": addr,
		"port": port,
	}).Info("TCP Connection")
	conn, err := net.Dial("tcp", addr+":"+port)
	LogError(err, "Error while initializing TCP connection")

	return conn
}

// Cookie is a unique XMPP session identifier
type Cookie uint64

func get_cookie() Cookie {
	var buf [8]byte
	if _, err := rand.Reader.Read(buf[:]); err != nil {
		panic("Failed to read random bytes: " + err.Error())
	}
	return Cookie(binary.LittleEndian.Uint64(buf[:]))
}

func create_user_hash(account string, password string) []byte {
	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	return enc
}
