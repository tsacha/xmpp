package xmpp

import (
	"bufio"
	"encoding/xml"
	"github.com/sirupsen/logrus"
	"net"
)

type incomingResult struct {
	XMLName   xml.Name
	Interface interface{}
	Error     error
}

type XMPPConnection struct {
	incoming chan incomingResult
	outgoing chan string
	reader   *xml.Decoder
	writer   *bufio.Writer
	conn     net.Conn
	State    XMPPState
}

type XMPPState struct {
	Jid       string
	Resource  string
	Roster    *RosterConfig
	Discovery *DiscoveryConfig
	Sm        *StreamManagementConfig
	Ping      *PingConfig
}

var err error

func (xmpp *XMPPConnection) Read() {
	for {
		if xmpp.reader != nil {
			t, _ := xmpp.reader.Token()
			switch t := t.(type) {
			case xml.StartElement:
				xmpp.incoming <- xmpp.ProcessElement(t)
			}
		} else {
			return
		}
	}
}

func (xmpp *XMPPConnection) Write() {
	for {
		xmpp.writer.WriteString(<-xmpp.outgoing)
		xmpp.writer.Flush()
		if xmpp.State.Sm != nil && xmpp.State.Sm.state {
			xmpp.State.Sm.seq += 1
			xmpp.State.Sm.output <- 1
		}
	}
}

func (xmpp *XMPPConnection) Process() {
	for {
		t := <-xmpp.incoming
		switch t := (t.Interface).(type) {
		case *streamMgmtRequest:
			if xmpp.State.Sm != nil && xmpp.State.Sm.state {
				// Stream Management: answer to server request
				xmpp.State.Sm.input <- 1
			}
		case *streamMgmtAnswer:
			if xmpp.State.Sm != nil && xmpp.State.Sm.state {
				// Stream Management: verify answer from server
				xmpp.State.Sm.verify <- t.Handled
			}
		case *clientIQ:
			if t.Type == "result" {
				if t.Query != nil {
					switch t.Query.XMLName.Space {
					case nsDiscoInfo:
						// Discovery results
						xmpp.State.Discovery.incoming <- t
					case nsRoster:
						// Roster results
						xmpp.State.Roster.incoming <- t
					}
				} else if xmpp.State.Ping != nil {
					// Pong
					xmpp.State.Ping.incoming <- t.ID
				}
			}
		}
	}
}

func (xmpp *XMPPConnection) Close() {
	xmpp.incoming = nil
	xmpp.outgoing = nil
	xmpp.reader = nil
	xmpp.writer = nil
	xmpp = nil
	logrus.Info("Disconnected")
}

func Connect(account string, password string, resource string) *XMPPConnection {
	LogInit()
	addr, port, domain := resolv_server(account)

	// TCP Connection
	conn := connect_server(addr, port)
	//	defer conn.Close()

	xmpp := &XMPPConnection{
		incoming: make(chan incomingResult),
		outgoing: make(chan string),
		reader:   xml.NewDecoder(teeIn{conn}),
		writer:   bufio.NewWriter(teeOut{conn}),
		State:    XMPPState{},
	}
	go xmpp.Write()

	xmpp.StartStream(domain)
	xmpp.EncryptConnection(domain, conn)

	go xmpp.Read()

	xmpp.AuthenticateUser(account, password, domain)
	xmpp.Bind(resource)

	if xmpp.State.Sm.version == 3 {
		xmpp.StartStreamManagement(true)
	}
	go xmpp.Process()

	return xmpp
}
