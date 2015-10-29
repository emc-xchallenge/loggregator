package helpers

import (
	"crypto/tls"
	"doppler/listeners"
	"encoding/binary"
	"net"

	"github.com/cloudfoundry/dropsonde/emitter"
	"github.com/cloudfoundry/dropsonde/factories"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"

	. "github.com/onsi/gomega"
)

func DialTLS(address, cert, key, ca string) (*tls.Conn, error) {
	tlsConfig, err := listeners.NewTLSConfig("../fixtures/client.crt", "../fixtures/client.key", "../fixtures/loggregator-ca.crt")
	Expect(err).NotTo(HaveOccurred())
	tlsConfig.ServerName = "doppler"
	return tls.Dial("tcp", address, tlsConfig)
}

func SendAppLogTLS(appID string, message string, connection net.Conn) error {
	logMessage := factories.NewLogMessage(events.LogMessage_OUT, message, appID, "APP")

	return SendEventTLS(logMessage, connection)
}

func SendEventTLS(event events.Event, conn net.Conn) error {
	envelope, err := emitter.Wrap(event, "origin")
	Expect(err).NotTo(HaveOccurred())

	bytes, err := proto.Marshal(envelope)
	if err != nil {
		return err
	}

	err = binary.Write(conn, binary.LittleEndian, uint32(len(bytes)))
	if err != nil {
		return err
	}

	_, err = conn.Write(bytes)
	return err
}
