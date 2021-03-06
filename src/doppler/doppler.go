package main

import (
	"fmt"
	"sync"
	"time"

	"doppler/config"
	"doppler/sinkserver"
	"doppler/sinkserver/blacklist"
	"doppler/sinkserver/sinkmanager"
	"doppler/sinkserver/websocketserver"

	"common/monitor"

	"doppler/listeners"

	"github.com/cloudfoundry/dropsonde/dropsonde_unmarshaller"
	"github.com/cloudfoundry/dropsonde/signature"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/appservice"
	"github.com/cloudfoundry/loggregatorlib/store"
	"github.com/cloudfoundry/loggregatorlib/store/cache"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/cloudfoundry/storeadapter"
)

type Doppler struct {
	*gosteno.Logger
	appStoreWatcher *store.AppServiceStoreWatcher

	errChan         chan error
	udpListener     listeners.Listener
	tlsListener     listeners.Listener
	sinkManager     *sinkmanager.SinkManager
	messageRouter   *sinkserver.MessageRouter
	websocketServer *websocketserver.WebsocketServer

	dropsondeUnmarshallerCollection dropsonde_unmarshaller.DropsondeUnmarshallerCollection
	dropsondeBytesChan              <-chan []byte
	dropsondeVerifiedBytesChan      chan []byte
	envelopeChan                    chan *events.Envelope
	signatureVerifier               *signature.Verifier

	storeAdapter storeadapter.StoreAdapter

	uptimeMonitor monitor.Monitor

	newAppServiceChan, deletedAppServiceChan <-chan appservice.AppService
	wg                                       sync.WaitGroup
}

func New(logger *gosteno.Logger,
	host string,
	config *config.Config,
	storeAdapter storeadapter.StoreAdapter,
	messageDrainBufferSize uint,
	dropsondeOrigin string,
	websocketWriteTimeout time.Duration,
	dialTimeout time.Duration) (*Doppler, error) {

	keepAliveInterval := 30 * time.Second

	appStoreCache := cache.NewAppServiceCache()
	appStoreWatcher, newAppServiceChan, deletedAppServiceChan := store.NewAppServiceStoreWatcher(storeAdapter, appStoreCache, logger)

	var udpListener listeners.Listener
	var tlsListener listeners.Listener
	var dropsondeBytesChan <-chan []byte
	var err error
	listenerEnvelopeChan := make(chan *events.Envelope)

	if config.EnableTLSTransport {
		tlsListener, err = listeners.NewTLSListener("tlsListener", fmt.Sprintf("%s:%d", host, config.TLSListenerConfig.Port), config.TLSListenerConfig, listenerEnvelopeChan, logger)
		if err != nil {
			return nil, err
		}
	}

	udpListener, dropsondeBytesChan = listeners.NewUDPListener(fmt.Sprintf("%s:%d", host, config.DropsondeIncomingMessagesPort), logger, "dropsondeListener")

	signatureVerifier := signature.NewVerifier(logger, config.SharedSecret)

	unmarshallerCollection := dropsonde_unmarshaller.NewDropsondeUnmarshallerCollection(logger, config.UnmarshallerCount)

	blacklist := blacklist.New(config.BlackListIps)
	metricTTL := time.Duration(config.ContainerMetricTTLSeconds) * time.Second
	sinkTimeout := time.Duration(config.SinkInactivityTimeoutSeconds) * time.Second
	sinkIOTimeout := time.Duration(config.SinkIOTimeoutSeconds) * time.Second
	sinkManager := sinkmanager.New(config.MaxRetainedLogMessages, config.SinkSkipCertVerify, blacklist, logger, messageDrainBufferSize, dropsondeOrigin, sinkTimeout, sinkIOTimeout, metricTTL, dialTimeout)

	websocketServer, err := websocketserver.New(fmt.Sprintf("%s:%d", host, config.OutgoingPort), sinkManager, websocketWriteTimeout, keepAliveInterval, config.MessageDrainBufferSize, dropsondeOrigin, logger)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the websocket server: %s", err.Error())
	}

	return &Doppler{
		Logger:                          logger,
		udpListener:                     udpListener,
		tlsListener:                     tlsListener,
		sinkManager:                     sinkManager,
		messageRouter:                   sinkserver.NewMessageRouter(sinkManager, logger),
		websocketServer:                 websocketServer,
		newAppServiceChan:               newAppServiceChan,
		deletedAppServiceChan:           deletedAppServiceChan,
		appStoreWatcher:                 appStoreWatcher,
		storeAdapter:                    storeAdapter,
		dropsondeBytesChan:              dropsondeBytesChan,
		dropsondeUnmarshallerCollection: unmarshallerCollection,
		envelopeChan:                    listenerEnvelopeChan,
		signatureVerifier:               signatureVerifier,
		dropsondeVerifiedBytesChan:      make(chan []byte),
		uptimeMonitor:                   monitor.NewUptimeMonitor(time.Duration(config.MonitorIntervalSeconds) * time.Second),
	}, nil
}

func (doppler *Doppler) Start() {
	doppler.errChan = make(chan error)

	doppler.wg.Add(6 + doppler.dropsondeUnmarshallerCollection.Size())

	go func() {
		defer doppler.wg.Done()
		doppler.appStoreWatcher.Run()
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.udpListener.Start()
	}()

	if doppler.tlsListener != nil {
		doppler.wg.Add(1)
		go func() {
			defer doppler.wg.Done()
			doppler.tlsListener.Start()
		}()
	}

	doppler.dropsondeUnmarshallerCollection.Run(doppler.dropsondeVerifiedBytesChan, doppler.envelopeChan, &doppler.wg)

	go func() {
		defer func() {
			doppler.wg.Done()
			close(doppler.dropsondeVerifiedBytesChan)
		}()
		doppler.signatureVerifier.Run(doppler.dropsondeBytesChan, doppler.dropsondeVerifiedBytesChan)
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.sinkManager.Start(doppler.newAppServiceChan, doppler.deletedAppServiceChan)
	}()

	go func() {
		defer func() {
			doppler.wg.Done()
			close(doppler.envelopeChan)
		}()
		doppler.messageRouter.Start(doppler.envelopeChan)
	}()

	go func() {
		defer doppler.wg.Done()
		doppler.websocketServer.Start()
	}()

	go doppler.uptimeMonitor.Start()

	// The following runs forever. Put all startup functions above here.
	for err := range doppler.errChan {
		doppler.Errorf("Got error %s", err)
	}
}

func (doppler *Doppler) Stop() {
	go doppler.udpListener.Stop()
	go doppler.tlsListener.Stop()
	go doppler.sinkManager.Stop()
	go doppler.messageRouter.Stop()
	go doppler.websocketServer.Stop()
	doppler.appStoreWatcher.Stop()
	doppler.wg.Wait()

	doppler.storeAdapter.Disconnect()
	close(doppler.errChan)
	doppler.uptimeMonitor.Stop()
}
