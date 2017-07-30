package xmpp

import (
	"bufio"
	"encoding/xml"
)

func Connect(account string, password string) {
	LogInit()
	addr, port, domain := resolvServer(account)

	// TCP Connection
	conn := ConnectServer(addr, port)
	defer conn.Close()

	xmppconn := &XMPPConnection{
		incoming: make(chan incomingResult),
		outgoing: make(chan string),
		reader:   xml.NewDecoder(conn),
		writer:   bufio.NewWriter(conn),
	}
	go xmppconn.Write()

	xmppconn.StartStream(domain)
	xmppconn.EncryptConnection(domain, conn)

	go xmppconn.Read()

	xmppconn.AuthenticateUser(account, password)
}
