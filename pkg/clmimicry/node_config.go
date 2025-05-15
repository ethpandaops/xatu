package clmimicry

import (
	"time"

	hermes "github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
)

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
	PrysmUseTLS   bool   `yaml:"prysmUseTls" default:"false"`

	// The maximum number of peers our libp2p host can be connected to.
	MaxPeers int `yaml:"maxPeers" default:"30"`

	// Limits the number of concurrent connection establishment routines. When
	// we discover peers over discv5 and are not at our MaxPeers limit we try
	// to establish a connection to a peer. However, we limit the concurrency to
	// this DialConcurrency value.
	DialConcurrency int `yaml:"dialConcurrency" default:"16"`

	// DataStreamType is the type of data stream to use for the node (e.g. kinesis, callback, etc).
	DataStreamType string `yaml:"dataStreamType" default:"callback"`

	// Subnets is the configuration for gossipsub subnets.
	Subnets map[string]*hermes.SubnetConfig `yaml:"subnets"`
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
		PrysmUseTLS:     h.PrysmUseTLS,
		MaxPeers:        h.MaxPeers,
		DialConcurrency: h.DialConcurrency,
		DataStreamType:  host.DataStreamtypeFromStr(h.DataStreamType),
		SubnetConfigs:   h.Subnets,
	}
}
