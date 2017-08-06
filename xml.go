package xmpp

import (
	"encoding/xml"
	"errors"
	"github.com/sirupsen/logrus"
)

// Read next XML element and send it to ProcessElement function
func (xmpp *XMPPConnection) NextElement() incomingResult {
	var nv incomingResult
	t, _ := xmpp.reader.Token()

	switch t := t.(type) {
	case xml.ProcInst:
		logrus.Info("Received XML from server")
		return xmpp.NextElement()
	case xml.StartElement:
		nv = xmpp.ProcessElement(t)
	}
	return nv
}

// Decode XML element
func (xmpp *XMPPConnection) ProcessElement(se xml.StartElement) incomingResult {
	var nv interface{}

	switch se.Name.Space + " " + se.Name.Local {
	// <stream> has no end element, parse it manually
	case nsStream + " stream":
		var stream streamStream
		for _, attr := range se.Attr {
			switch attr.Name.Local {
			case "stream":
				stream.Stream = attr.Value
			case "lang":
				stream.Lang = attr.Value
			case "id":
				stream.ID = attr.Value
			case "version":
				stream.Version = attr.Value
			case "xmlns":
				stream.Xmlns = attr.Value
			}
		}
		logrus.WithFields(logrus.Fields{
			"stream":  stream.Stream,
			"lang":    stream.Lang,
			"id":      stream.ID,
			"version": stream.Version,
			"xmlns":   stream.Xmlns,
		}).Info("Received stream from server")
		return (incomingResult{se.Name, stream, nil})
	case nsStream + " features":
		nv = &streamFeatures{}
	case nsStartTLS + " proceed":
		nv = &tlsProceed{}
	case nsSASL + " success":
		nv = &saslSuccess{}
	case nsSASL + " failure":
		nv = &saslFailure{}
	case nsClient + " iq":
		nv = &clientIQ{}
	case nsStreamMgmt + " enabled":
		nv = &streamMgmtEnabled{}
	case nsStreamMgmt + " a":
		nv = &streamMgmtAnswer{}
	case nsStreamMgmt + " r":
		nv = &streamMgmtRequest{}
	default:
		return (incomingResult{xml.Name{}, nil, errors.New("unexpected XMPP message " +
			se.Name.Space + " " + se.Name.Local)})
	}

	if xmpp.reader == nil {
		return incomingResult{xml.Name{}, nil, err}
	}

	// Unmarshal into that storage.
	err = xmpp.reader.DecodeElement(nv, &se)
	if err != nil {
		return incomingResult{xml.Name{}, nil, err}
	}

	// If stream management is active
	if xmpp.State.Sm != nil && xmpp.State.Sm.state {
		// Do not count namespace stream management
		if se.Name.Space != nsStreamMgmt {
			xmpp.State.Sm.handled += 1
		}
	}
	return incomingResult{se.Name, nv, err}
}
