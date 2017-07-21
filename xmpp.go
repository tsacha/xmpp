package xmpp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
)

func resolv_xmpp_server(account string) (string, string, string) {
	domain := strings.Split(account, "@")[1]
	_, addrs, _ := net.LookupSRV("xmpp-client", "tcp", domain)

	server_choice := rand.Intn(len(addrs))
	return addrs[server_choice].Target, fmt.Sprint(addrs[server_choice].Port), domain
}

func Connect(account string, password string) {
	addr, port, domain := resolv_xmpp_server(account)

	println("# TCP Connection")
	conn, err := net.Dial("tcp", addr+":"+port)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	println("# Stream request")
	_, err = fmt.Fprintf(conn, "<?xml version='1.0'?>\n"+
		"<stream:stream to='%s' xmlns='%s'\n"+
		" xmlns:stream='%s' version='1.0'>\n",
		domain, "jabber:client", "http://etherx.jabber.org/streams")

	if err != nil {
		log.Println(err)
		return
	}

	reply := make([]byte, 1024)

	_, err = conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
		return
	}

	println("# Stream response")
	println(string(reply))

	println("# StartTLS request")
	_, err = fmt.Fprintf(conn,
		"<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>\n")

	if err != nil {
		log.Println(err)
		return
	}

	reply = make([]byte, 1024)

	_, err = conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
		return
	}

	println("# StartTLS response")
	println(string(reply))

	conf := &tls.Config{
		ServerName: domain,
		//InsecureSkipVerify: true,
	}

	println("# TLS Handshake")
	t := tls.Client(conn, conf)

	if err = t.Handshake(); err != nil {
		println("TLS Handshake error:", err.Error())
		return
	}
	conn = t

	println("# TLS Stream request")
	_, err = fmt.Fprintf(conn, "<?xml version='1.0'?>\n"+
		"<stream:stream to='%s' xmlns='%s'\n"+
		" xmlns:stream='%s' version='1.0'>\n",
		domain, "jabber:client", "http://etherx.jabber.org/streams")

	if err != nil {
		log.Println(err)
		return
	}

	reply = make([]byte, 1024)

	_, err = conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
		return
	}

	println("# TLS Stream response")
	println(string(reply))

	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	println("# Authentication")
	fmt.Fprintf(conn, "<auth xmlns='%s' mechanism='PLAIN'>%s</auth>\n", "urn:ietf:params:xml:ns:xmpp-sasl", enc)

	_, err = conn.Read(reply)
	if err != nil {
		println("Write to server failed:", err.Error())
		return
	}

	println("# Authentication response")
	println(string(reply))

}
