package xmpp

import (
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"io"
	"os"
)

var out = ansi.ColorFunc("blue+b")
var in = ansi.ColorFunc("green+b")

func LogInit() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}

func LogError(err error, prefix string) {
	if err != nil {
		logrus.Error(prefix+": ", err)
		return
	}
}

func LogInOut(direction string, xml string) {
	if direction == "in" {
		logrus.Debug(in(xml))
	} else if direction == "out" {
		logrus.Debug(out(xml))
	} else {
		logrus.Debug(xml)
	}

}

type teeIn struct {
	r io.Reader
}

func (t teeIn) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		LogInOut("in", string(p[0:n]))
	}
	return
}

type teeOut struct {
	w io.Writer
}

func (t teeOut) Write(p []byte) (n int, err error) {
	n, err = t.w.Write(p)
	if n > 0 {
		LogInOut("out", string(p[0:n]))
	}
	return
}
