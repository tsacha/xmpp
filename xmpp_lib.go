package xmpp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"strings"
)

const (
	nsStream   = "http://etherx.jabber.org/streams"
	nsClient   = "jabber:client"
	nsStartTLS = "urn:ietf:params:xml:ns:xmpp-tls"
	nsSASL     = "urn:ietf:params:xml:ns:xmpp-sasl"
)

func resolvServer(account string) (string, string, string) {
	domain := strings.Split(account, "@")[1]
	_, addrs, _ := net.LookupSRV("xmpp-client", "tcp", domain)

	// Random choice between SRV records
	server_choice := rand.Intn(len(addrs))

	logrus.WithFields(logrus.Fields{
		"nb_entries": len(addrs),
		"domain":     domain,
		"addr":       addrs[server_choice].Target,
		"port":       addrs[server_choice].Port,
	}).Info("Resolve XMPP server")
	return addrs[server_choice].Target, fmt.Sprint(addrs[server_choice].Port), domain
}

func ConnectServer(addr string, port string) net.Conn {
	logrus.WithFields(logrus.Fields{
		"addr": addr,
		"port": port,
	}).Info("TCP Connection")
	conn, err := net.Dial("tcp", addr+":"+port)
	LogError(err, "Error while initializing TCP connection")

	return conn
}

func EncryptConnection(domain string, conn net.Conn) net.Conn {
	conf := &tls.Config{
		ServerName: domain,
		//InsecureSkipVerify: true,
	}

	// TLS Handshake
	logrus.Info("TLS Handshake")
	t := tls.Client(conn, conf)
	err = t.Handshake()
	LogError(err, "TLS Handshake")

	return t
}

func QueryServer(request string, content string, conn net.Conn) {
	logrus.Info(request)
	LogInOut("out", content)
	_, err = fmt.Fprintf(conn, content)
	LogError(err, "Stream request")

	// Stream response
	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	LogError(err, request)
	LogInOut("in", string(reply))
}

func create_user_hash(account string, password string) []byte {
	raw := "\x00" + account + "\x00" + password
	enc := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(enc, []byte(raw))

	return enc
}

func AuthenticateUser(account string, password string, conn net.Conn) {
	hash := create_user_hash(account, password)
	auth_request := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>%s</auth>",
		nsSASL, hash)
	auth_request_anonymous := fmt.Sprintf("<auth xmlns='%s' mechanism='PLAIN'>********</auth>",
		nsSASL)

	logrus.WithFields(logrus.Fields{
		"account": account,
	}).Info("Authentication")
	LogInOut("out", auth_request_anonymous)

	fmt.Fprintf(conn, auth_request)

	// Authentication response
	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	LogError(err, "Authentication")
	LogInOut("in", string(reply))
}
