package xmpp

import (
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"os"
)

var out = ansi.ColorFunc("cyan+b")
var in = ansi.ColorFunc("magenta+b")

func LogInit() {
	logrus.SetFormatter(&prefixed.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
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

func LogError(err error, prefix string) {
	if err != nil {
		logrus.Error(prefix+": ", err)
		return
	}
}
