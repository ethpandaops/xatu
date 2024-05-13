package clmimicry

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clmimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	hermes "github.com/probe-lab/hermes/eth"
	"github.com/sirupsen/logrus"
)

type Config struct {
	LoggingLevel string  `yaml:"logging" default:"info"`
	MetricsAddr  string  `yaml:"metricsAddr" default:":9090"`
	PProfAddr    *string `yaml:"pprofAddr"`
	ProbeAddr    *string `yaml:"probeAddr"`

	// The name of the mimicry
	Name string `yaml:"name"`

	// Ethereum configuration
	Ethereum ethereum.Config `yaml:"ethereum"`

	// Outputs configuration
	Outputs []output.Config `yaml:"outputs"`

	// Labels configures the mimicry with labels
	Labels map[string]string `yaml:"labels"`

	// NTP Server to use for clock drift correction
	NTPServer string `yaml:"ntpServer" default:"time.google.com"`

	// Node is the configuration for the node
	Node NodeConfig `yaml:"node"`

	// Events is the configuration for the events
	Events EventConfig `yaml:"events"`
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return fmt.Errorf("invalid ethereum config: %w", err)
	}

	for _, output := range c.Outputs {
		if err := output.Validate(); err != nil {
			return fmt.Errorf("invalid output config %s: %w", output.Name, err)
		}
	}

	if err := c.Events.Validate(); err != nil {
		return fmt.Errorf("invalid events config: %w", err)
	}

	return nil
}

func (c *Config) CreateSinks(log logrus.FieldLogger) ([]output.Sink, error) {
	sinks := make([]output.Sink, len(c.Outputs))

	for i, out := range c.Outputs {
		sink, err := output.NewSink(
			out.Name,
			out.SinkType,
			out.Config,
			log,
			out.FilterConfig,
			processor.ShippingMethodAsync,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

type NodeConfig struct {
	// The private key for the libp2p host and local enode in hex format
	PrivateKeyStr string `yaml:"privateKeyStr" default:""`

	// General timeout when communicating with other network participants
	DialTimeout time.Duration `yaml:"dialTimeout" default:"5s"`

	// The address information of the local ethereuem [enode.Node].
	Devp2pHost string `yaml:"devp2pHost" default:"0.0.0.0"`
	Devp2pPort int    `yaml:"devp2pPort" default:"0"`

	// The address information of the local libp2p host
	Libp2pHost string `yaml:"libp2pHost" default:"0.0.0.0"`
	Libp2pPort int    `yaml:"libp2pPort" default:"0"`

	// The address information where the Beacon API or Prysm's custom API is accessible at
	PrysmHost     string `yaml:"prysmHost" default:"127.0.0.1"`
	PrysmPortHTTP int    `yaml:"prysmPortHttp" default:"3500"`
	PrysmPortGRPC int    `yaml:"prysmPortGrpc" default:"4000"`

	// The maximum number of peers our libp2p host can be connected to.
	MaxPeers int `yaml:"maxPeers" default:"30"`

	// Limits the number of concurrent connection establishment routines. When
	// we discover peers over discv5 and are not at our MaxPeers limit we try
	// to establish a connection to a peer. However, we limit the concurrency to
	// this DialConcurrency value.
	DialConcurrency int `yaml:"dialConcurrency" default:"16"`
}

func (h *NodeConfig) AsHermesConfig() *hermes.NodeConfig {
	return &hermes.NodeConfig{
		PrivateKeyStr:   h.PrivateKeyStr,
		DialTimeout:     h.DialTimeout,
		Devp2pHost:      h.Devp2pHost,
		Devp2pPort:      h.Devp2pPort,
		Libp2pHost:      h.Libp2pHost,
		Libp2pPort:      h.Libp2pPort,
		PrysmHost:       h.PrysmHost,
		PrysmPortHTTP:   h.PrysmPortHTTP,
		PrysmPortGRPC:   h.PrysmPortGRPC,
		MaxPeers:        h.MaxPeers,
		DialConcurrency: h.DialConcurrency,
	}
}

type EventConfig struct {
	RecvRPCEnabled                    bool `yaml:"recvRpcEnabled" default:"false"`
	SendRPCEnabled                    bool `yaml:"sendRpcEnabled" default:"false"`
	AddPeerEnabled                    bool `yaml:"addPeerEnabled" default:"true"`
	RemovePeerEnabled                 bool `yaml:"removePeerEnabled" default:"true"`
	ConnectedEnabled                  bool `yaml:"connectedEnabled" default:"true"`
	DisconnectedEnabled               bool `yaml:"disconnectedEnabled" default:"true"`
	JoinEnabled                       bool `yaml:"joinEnabled" default:"true"`
	HandleMetadataEnabled             bool `yaml:"handleMetadataEnabled" default:"true"`
	HandleStatusEnabled               bool `yaml:"handleStatusEnabled" default:"true"`
	GossipSubBeaconBlockEnabled       bool `yaml:"gossipSubBeaconBlockEnabled" default:"true"`
	GossipSubAttestationEnabled       bool `yaml:"gossipSubAttestationEnabled" default:"true"`
	GossipSubDataColumnSidecarEnabled bool `yaml:"gossipSubDataColumnSidecarEnabled" default:"true"`
}

func (e *EventConfig) Validate() error {
	return nil
}
