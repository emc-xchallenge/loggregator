package dopplerforwarder

import (
	"encoding/binary"
	"unicode"
	"unicode/utf8"

	"metron/clientpool"

	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/dropsonde/signature"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

var metricNames map[events.Envelope_EventType]string

func init() {
	metricNames = make(map[events.Envelope_EventType]string)
	for eventType, eventName := range events.Envelope_EventType_name {
		r, n := utf8.DecodeRuneInString(eventName)
		modifiedName := string(unicode.ToLower(r)) + eventName[n:]
		metricName := "dropsondeMarshaller." + modifiedName + "Marshalled"
		metricNames[events.Envelope_EventType(eventType)] = metricName
	}
}

//go:generate counterfeiter -o fakes/fake_clientpool.go . ClientPool
type ClientPool interface {
	RandomClient() (clientpool.Client, error)
}

type DopplerForwarder struct {
	clientPool   ClientPool
	sharedSecret []byte
	logger       *gosteno.Logger
}

func New(clientPool ClientPool, sharedSecret []byte, logger *gosteno.Logger) *DopplerForwarder {
	return &DopplerForwarder{
		clientPool:   clientPool,
		sharedSecret: sharedSecret,
		logger:       logger,
	}
}

func (d *DopplerForwarder) Write(message *events.Envelope) {
	client, err := d.clientPool.RandomClient()
	if err != nil {
		d.logger.Errord(map[string]interface{}{
			"error": err.Error(),
		}, "DopplerForwarder: can't forward message")
		return
	}

	messageBytes, err := proto.Marshal(message)
	if err != nil {
		d.logger.Errorf("DopplerForwarder: marshal error %v", err)
		metrics.BatchIncrementCounter("dropsondeMarshaller.marshalErrors")
		return
	}

	switch client.Scheme() {
	case "udp":
		signedMessage := signature.SignMessage(messageBytes, d.sharedSecret)

		bytesWritten, err := client.Write(signedMessage)
		if err != nil {
			metrics.BatchIncrementCounter("udp.sendErrorCount")
			d.logger.Debugd(map[string]interface{}{
				"scheme":  client.Scheme(),
				"address": client.Address(),
			}, "Error writing legacy message")
			return
		}
		metrics.BatchIncrementCounter("udp.sentMessageCount")
		metrics.BatchAddCounter("udp.sentByteCount", uint64(bytesWritten))
	case "tls":
		var bytesWritten int
		err = binary.Write(client, binary.LittleEndian, uint32(len(messageBytes)))
		if err == nil {
			bytesWritten, err = client.Write(messageBytes)
		}
		if err != nil {
			metrics.BatchIncrementCounter("tls.sendErrorCount")
			client.Close()

			d.logger.Errord(map[string]interface{}{
				"scheme":  client.Scheme(),
				"address": client.Address(),
				"error":   err.Error(),
			}, "DopplerForwarder: streaming error")
			return
		}
		metrics.BatchIncrementCounter("tls.sentMessageCount")
		metrics.BatchAddCounter("tls.sentByteCount", uint64(bytesWritten+4))
	default:
		d.logger.Errorf("DopplerForwarder: unknown protocol, %s for %s", client.Scheme(), client.Address())
		return
	}

	d.incrementMessageCount(message.GetEventType())
	metrics.BatchIncrementCounter("DopplerForwarder.sentMessages")
}

func (d *DopplerForwarder) incrementMessageCount(eventType events.Envelope_EventType) {
	metricName := metricNames[eventType]
	metrics.BatchIncrementCounter(metricName)
}
