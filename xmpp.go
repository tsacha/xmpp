package xmpp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/mgutz/ansi"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"math/rand"
	"net"
	"os"
	"strings"
)

const (
	nsStream   = "http://etherx.jabber.org/streams"
	nsClient   = "jabber:client"
	nsStartTLS = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL     = "urn:ietf:params:xml:ns:xmpp-sasl"
)

var out = ansi.ColorFunc("cyan+b")
var in = ansi.ColorFunc("magenta+b")

func resolv_xmpp_server(account string) (string, string, string) {
	domain := strings.Split(account, "@")[1]
	_, addrs, _ := net.LookupSRV("xmpp-client", "tcp", domain)

	// Random choice between SRV records
	server_choice := rand.Intn(len(addrs))

	log.WithFields(log.Fields{
		"nb_entries": len(addrs),
		"domain":     domain,
		"addr":       addrs[server_choice].Target,
		"port":       addrs[server_choice].Port,
	}).Info("Resolve XMPP server")
	return addrs[server_choice].Target, fmt.Sprint(addrs[server_choice].Port), domain
}

func debug_xmpp(direction string, xml string) {
	if direction == "in" {
		log.Debug(in(xml))
	} else if direction == "out" {
		log.Debug(out(xml))
	} else {
		log.Debug(xml)
	}

}

func check_error(err error, prefix string) {
	if err != nil {
		log.Error(prefix+": ", err)
		return
	}
}

func Connect(account string, password string) {
	log.SetFormatter(&prefixed.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	addr, port, domain := resolv_xmpp_server(account)

	// TCP Connection
	log.WithFields(log.Fields{
		"addr": addr,
		"port": port,
	}).Info("TCP Connection")
	conn, err := net.Dial("tcp", addr+":"+port)
	check_error(err, "Error while initializing TCP connection")
	defer conn.Close()

	// Stream request
	stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
		"<stream:stream to='%s' xmlns='%s'"+
		" xmlns:stream='%s' version='1.0'>",
		domain, nsClient, nsStream)
	log.Info("Stream request")
	debug_xmpp("out", stream_request)
	_, err = fmt.Fprintf(conn, stream_request)
	check_error(err, "Stream request")

	// Stream response
	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	check_error(err, "Stream response")
	debug_xmpp("in", string(reply))

	// StartTLS request
	starttls_request := "<starttls xmlns='" + nsStartTLS + "'/>"
	log.Info("StartTLS request")
	debug_xmpp("out", starttls_request)
	_, err = fmt.Fprintf(conn, starttls_request)
	check_error(err, "StartTLS request")

	reply = make([]byte, 1024)
	_, err = conn.Read(reply)
	check_error(err, "StartTLS response")
	debug_xmpp("in", string(reply))

	conf := &tls.Config{
		ServerName: domain,
		//InsecureSkipVerify: true,
	}

	// TLS Handshake
	log.Info("TLS Handshake")
	t := tls.Client(conn, conf)
	err = t.Handshake()
	check_error(err, "TLS Handshake")
	conn = t

	// TLS Stream request
	log.Info("TLS Stream request")
	debug_xmpp("out", stream_request)
	_, err = fmt.Fprintf(conn, stream_request)
	check_error(err, "TLS Stream request")

	// TLS Stream response
	reply = make([]byte, 1024)
	_, err = conn.Read(reply)
	check_error(err, "TLS Stream response")
	debug_xmpp("in", string(reply))

	// Authentication query
	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	log.WithFields(log.Fields{
		"account": account,
	}).Info("Authentication")
	auth_request := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>%s</auth>",
		nsSASL, enc)
	debug_xmpp("out", auth_request)
	fmt.Fprintf(conn, auth_request)

	// Authentication response
	reply = make([]byte, 1024)
	_, err = conn.Read(reply)
	check_error(err, "Authentication")
	debug_xmpp("in", string(reply))
}
