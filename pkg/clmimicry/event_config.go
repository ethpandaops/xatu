package clmimicry

// EventConfig represents configuration for all event types.
type EventConfig struct {
	RecvRPCEnabled              bool `yaml:"recvRpcEnabled" default:"false"`
	SendRPCEnabled              bool `yaml:"sendRpcEnabled" default:"false"`
	AddPeerEnabled              bool `yaml:"addPeerEnabled" default:"true"`
	RemovePeerEnabled           bool `yaml:"removePeerEnabled" default:"true"`
	ConnectedEnabled            bool `yaml:"connectedEnabled" default:"true"`
	DisconnectedEnabled         bool `yaml:"disconnectedEnabled" default:"true"`
	JoinEnabled                 bool `yaml:"joinEnabled" default:"true"`
	HandleMetadataEnabled       bool `yaml:"handleMetadataEnabled" default:"true"`
	HandleStatusEnabled         bool `yaml:"handleStatusEnabled" default:"true"`
	GossipSubBeaconBlockEnabled bool `yaml:"gossipSubBeaconBlockEnabled" default:"true"`
	GossipSubAttestationEnabled bool `yaml:"gossipSubAttestationEnabled" default:"true"`
	GossipSubBlobSidecarEnabled bool `yaml:"gossipSubBlobSidecarEnabled" default:"true"`
}

// Validate validates the event config.
func (e *EventConfig) Validate() error {
	return nil
}
