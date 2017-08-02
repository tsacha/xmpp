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
	XMLName xml.Name `xml:"enabled"`
	Resume  string   `xml:"resume,attr"`
	ID      string   `xml:"id,attr"`
}

func (xmppconn *XMPPConnection) SMAnswers() {
}

func (xmppconn *XMPPConnection) SMRequests() {
}

func (xmppconn *XMPPConnection) StartStreamManagement(resume bool) {
	logrus.Info("Start stream management v3")
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
		}).Info("Stream management v3 enabled")
		xmppconn.state.sm_state = true

		go xmppconn.SMAnswers()
		go xmppconn.SMRequests()
	}
}
