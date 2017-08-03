// XEP 0198 — Stream Management
package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
)

// XEP 0198 # 2 — Stream Feature
type sm struct {
	XMLName  xml.Name `xml:"sm"`
	Optional string   `xml:"optional"`
}

// XEP 0198 # 3 — Enabling Stream Management
type enable struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 enable"`
	Resume  string   `xml:"resume,attr,omitempty"`
}

type enabled struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 enabled"`
	Resume  string   `xml:"resume,attr"`
	ID      string   `xml:"id,attr"`
}

type request struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 r"`
}

type answer struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 a"`
	Handled int      `xml:"h,attr"`
}

type StreamManagementConfig struct {
	version  int
	optional bool
	state    bool
	handled  int
	seq      int
	window   int
	input    chan int
	output   chan int
	verify   chan int
}

func (xmppconn *XMPPConnection) SMAnswers() {
	for {
		<-xmppconn.state.sm.input
		answer := answer{Handled: xmppconn.state.sm.handled}
		output, _ := xml.Marshal(answer)

		logrus.WithFields(logrus.Fields{
			"h": xmppconn.state.sm.handled,
		}).Info("[XEP 0198] Answering to server request")

		xmppconn.writer.WriteString(string(output))
		xmppconn.writer.Flush()
	}
}

func (xmppconn *XMPPConnection) SMRequests() {
	for {
		<-xmppconn.state.sm.output
		if (xmppconn.state.sm.seq % xmppconn.state.sm.window) == 0 {
			request := request{}
			output, _ := xml.Marshal(request)

			xmppconn.writer.WriteString(string(output))
			xmppconn.writer.Flush()
			logrus.WithFields(logrus.Fields{
				"seq": xmppconn.state.sm.seq,
			}).Info("[XEP 0198] Request ACK to server")
		}
	}
}

func (xmppconn *XMPPConnection) SMVerify() {
	for {
		srv_handled := <-xmppconn.state.sm.verify
		logrus.WithFields(logrus.Fields{
			"h": srv_handled,
		}).Info("[XEP 0198] Receiving request from server")
	}
}

func (xmppconn *XMPPConnection) StartStreamManagement(resume bool) {
	logrus.Info("[XEP 0198] Start stream management")

	var resume_str string
	if resume {
		resume_str = "true"
	} else {
		resume_str = "false"
	}

	enable := &enable{Resume: resume_str}
	output, _ := xml.Marshal(enable)

	xmppconn.outgoing <- string(output)
	stream_response := <-xmppconn.incoming
	switch t := stream_response.Interface.(type) {
	case *enabled:
		logrus.WithFields(logrus.Fields{
			"id":     t.ID,
			"resume": resume,
		}).Info("[XEP 0198] Stream management enabled")
		xmppconn.state.sm.state = true
		xmppconn.state.sm.window = 5
		xmppconn.state.sm.input = make(chan int)
		xmppconn.state.sm.output = make(chan int)
		xmppconn.state.sm.verify = make(chan int)

		go xmppconn.SMAnswers()
		go xmppconn.SMRequests()
		go xmppconn.SMVerify()
	}
}
