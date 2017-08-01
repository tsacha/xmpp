package xmpp

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"strconv"
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
		reader:   xml.NewDecoder(teeIn{conn}),
		writer:   bufio.NewWriter(teeOut{conn}),
		conf:     XMPPConf{},
	}
	go xmppconn.Write()

	xmppconn.StartStream(domain)
	xmppconn.EncryptConnection(domain, conn)

	go xmppconn.Read()

	xmppconn.AuthenticateUser(account, password, domain)

	id_bind := strconv.FormatUint(uint64(getCookie()), 10)
	iq_request := fmt.Sprintf("<iq type='%s' id='%s'>"+
		"<bind xmlns='%s'>"+
		"<resource>%s</resource>"+
		"</bind>"+
		"</iq>",
		"set", id_bind, nsBind, "xmpp-sacha")

	xmppconn.outgoing <- iq_request
	iq_response := <-xmppconn.incoming
	switch t := iq_response.Interface.(type) {
	case *clientIQ:
		fmt.Println(string(t.Query))
	}

	if xmppconn.conf.stream == 3 {
		stream_request := fmt.Sprintf("<enable xmlns='%s' resume='%s' />",
			nsStreamMgmtv3, "true")
		xmppconn.outgoing <- stream_request
		stream_response := <-xmppconn.incoming
		switch t := stream_response.Interface.(type) {
		case *enabled:
			fmt.Println(t.Id)
		}
	} else if xmppconn.conf.stream == 2 {
		stream_request := fmt.Sprintf("<enable xmlns='%s' resume='%s' />",
			nsStreamMgmtv3, "true")
		xmppconn.outgoing <- stream_request
		stream_response := <-xmppconn.incoming
		switch t := stream_response.Interface.(type) {
		case *enabled:
			fmt.Println(t.Id)
		}
	}

}
