// XEP 0198 — Stream Management
package xmpp

import (
	"encoding/xml"
	"github.com/sirupsen/logrus"
)

// XEP 0198 # 2 — Stream Feature
type streamMgmtSm struct {
	XMLName  xml.Name `xml:"sm"`
	Optional string   `xml:"optional"`
}

// XEP 0198 # 3 — Enabling Stream Management
type streamMgmtEnable struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 enable"`
	Resume  string   `xml:"resume,attr,omitempty"`
}

type streamMgmtEnabled struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 enabled"`
	Resume  string   `xml:"resume,attr"`
	ID      string   `xml:"id,attr"`
}

type streamMgmtRequest struct {
	XMLName xml.Name `xml:"urn:xmpp:sm:3 r"`
}

type streamMgmtAnswer struct {
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
		<-xmppconn.State.Sm.input
		answer := streamMgmtAnswer{Handled: xmppconn.State.Sm.handled}
		output, _ := xml.Marshal(answer)

		logrus.WithFields(logrus.Fields{
			"h": xmppconn.State.Sm.handled,
		}).Info("[XEP 0198] Answering to server request")

		xmppconn.writer.WriteString(string(output))
		xmppconn.writer.Flush()
	}
}

func (xmppconn *XMPPConnection) SMRequests() {
	for {
		<-xmppconn.State.Sm.output
		if (xmppconn.State.Sm.seq % xmppconn.State.Sm.window) == 0 {
			request := streamMgmtRequest{}
			output, _ := xml.Marshal(request)

			xmppconn.writer.WriteString(string(output))
			xmppconn.writer.Flush()
			logrus.WithFields(logrus.Fields{
				"seq": xmppconn.State.Sm.seq,
			}).Info("[XEP 0198] Request ACK to server")
		}
	}
}

func (xmppconn *XMPPConnection) SMVerify() {
	for {
		srv_handled := <-xmppconn.State.Sm.verify
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

	enable := &streamMgmtEnable{Resume: resume_str}
	output, _ := xml.Marshal(enable)

	xmppconn.outgoing <- string(output)
	stream_response := <-xmppconn.incoming
	switch t := stream_response.Interface.(type) {
	case *streamMgmtEnabled:
		logrus.WithFields(logrus.Fields{
			"id":     t.ID,
			"resume": resume,
		}).Info("[XEP 0198] Stream management enabled")
		xmppconn.State.Sm.state = true
		xmppconn.State.Sm.window = 5
		xmppconn.State.Sm.input = make(chan int)
		xmppconn.State.Sm.output = make(chan int)
		xmppconn.State.Sm.verify = make(chan int)

		go xmppconn.SMAnswers()
		go xmppconn.SMRequests()
		go xmppconn.SMVerify()
	}
}
