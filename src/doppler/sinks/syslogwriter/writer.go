package syslogwriter

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	rfc5424 = "2006-01-02T15:04:05.999999Z07:00"
)

var badBytes = []byte("\000")
var emptyBytes = []byte{}
var newLine = []byte("\n")

type Writer interface {
	Connect() error
	Write(p int, b []byte, source, sourceId string, timestamp int64) (int, error)
	Close() error
}

func NewWriter(outputUrl *url.URL, appId string, skipCertVerify bool, dialTimeout time.Duration, ioTimeout time.Duration) (Writer, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	switch outputUrl.Scheme {
	case "https":
		return NewHttpsWriter(outputUrl, appId, skipCertVerify, dialer, ioTimeout)
	case "syslog":
		return NewSyslogWriter(outputUrl, appId, dialer, ioTimeout)
	case "syslog-tls":
		return NewTlsWriter(outputUrl, appId, skipCertVerify, dialer, ioTimeout)
	default:
		return nil, errors.New(fmt.Sprintf("Invalid scheme type %s, must be https, syslog-tls or syslog", outputUrl.Scheme))
	}
}

func clean(in []byte) []byte {
	return bytes.Replace(in, badBytes, emptyBytes, -1)
}

func createMessage(p int, appId string, source string, sourceId string, msg []byte, timestamp int64) string {
	// ensure it ends in a \n
	nl := ""
	if !bytes.HasSuffix(msg, newLine) {
		nl = "\n"
	}

	msg = clean(msg)
	timeString := time.Unix(0, timestamp).Format(rfc5424)
	timeString = strings.Replace(timeString, "Z", "+00:00", 1)

	var formattedSource string
	if strings.ToUpper(source) == "APP" {
		formattedSource = fmt.Sprintf("[%s/%s]", source, sourceId)
	} else {
		formattedSource = fmt.Sprintf("[%s]", source)
	}

	// syslog format https://tools.ietf.org/html/rfc5424#section-6
	return fmt.Sprintf("<%d>1 %s %s %s %s - - %s%s", p, timeString, "loggregator", appId, formattedSource, msg, nl)
}
