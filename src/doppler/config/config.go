package config

import (
	"doppler/iprange"
	"errors"
	"time"

	"encoding/json"
	"os"
)

const HeartbeatInterval = 10 * time.Second

type TLSListenerConfig struct {
	Port     uint32
	CertFile string
	KeyFile  string
	CAFile   string
}

type Config struct {
	Syslog                        string
	EtcdUrls                      []string
	EtcdMaxConcurrentRequests     int
	Index                         uint
	DropsondeIncomingMessagesPort uint32
	OutgoingPort                  uint32
	LogFilePath                   string
	MaxRetainedLogMessages        uint32
	MessageDrainBufferSize        uint
	SharedSecret                  string
	SinkSkipCertVerify            bool
	SinkTlsSkipCertVerify         bool
	BlackListIps                  []iprange.IPRange
	JobName                       string
	Zone                          string
	ContainerMetricTTLSeconds     int
	SinkInactivityTimeoutSeconds  int
	SinkIOTimeoutSeconds          int
	UnmarshallerCount             int
	MetronAddress                 string
	MonitorIntervalSeconds        uint
	SinkDialTimeoutSeconds        int
	WebsocketWriteTimeoutSeconds  int
	EnableTLSTransport            bool
	TLSListenerConfig             TLSListenerConfig
}

func (c *Config) validate() (err error) {
	if c.MaxRetainedLogMessages == 0 {
		return errors.New("Need max number of log messages to retain per application")
	}

	if c.BlackListIps != nil {
		err = iprange.ValidateIpAddresses(c.BlackListIps)
		if err != nil {
			return err
		}
	}

	if c.EnableTLSTransport {
		if c.TLSListenerConfig.CertFile == "" || c.TLSListenerConfig.KeyFile == "" || c.TLSListenerConfig.Port == 0 {
			return errors.New("invalid TLS listener configuration")
		}
	}

	return err
}

func ParseConfig(configFile string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(config)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	if config.MonitorIntervalSeconds == 0 {
		config.MonitorIntervalSeconds = 60
	}

	if config.SinkDialTimeoutSeconds == 0 {
		config.SinkDialTimeoutSeconds = 1
	}

	if config.WebsocketWriteTimeoutSeconds == 0 {
		config.WebsocketWriteTimeoutSeconds = 30
	}

	if config.UnmarshallerCount == 0 {
		config.UnmarshallerCount = 1
	}

	if config.EtcdMaxConcurrentRequests < 1 {
		config.EtcdMaxConcurrentRequests = 1
	}

	return config, nil
}
