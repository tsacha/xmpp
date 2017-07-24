package xmpp

import (
	"fmt"
)

var err error

func Connect(account string, password string) {
	LogInit()
	addr, port, domain := resolvServer(account)

	// TCP Connection
	conn := ConnectServer(addr, port)
	defer conn.Close()

	// Stream request
	stream_request := fmt.Sprintf("<?xml version='1.0'?>"+
		"<stream:stream to='%s' xmlns='%s'"+
		" xmlns:stream='%s' version='1.0'>",
		domain, nsClient, nsStream)
	QueryServer("Stream request", stream_request, conn)

	// StartTLS request
	starttls_request := "<starttls xmlns='" + nsStartTLS + "'/>"
	QueryServer("StartTLS request", starttls_request, conn)

	// TLS encryption
	conn = EncryptConnection(domain, conn)

	// TLS Stream request
	QueryServer("TLS Stream request", stream_request, conn)

	// Authentication query
	AuthenticateUser(account, password, conn)
}
