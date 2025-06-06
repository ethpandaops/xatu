package clmimicry

// EventConfig represents configuration for all event types.
type EventConfig struct {
	RecvRPCEnabled                 bool `yaml:"recvRpcEnabled" default:"false"`
	SendRPCEnabled                 bool `yaml:"sendRpcEnabled" default:"false"`
	DropRPCEnabled                 bool `yaml:"dropRpcEnabled" default:"false"`
	RpcMetaControlIHaveEnabled     bool `yaml:"rpcMetaControlIHaveEnabled" default:"false"`
	RpcMetaControlIWantEnabled     bool `yaml:"rpcMetaControlIWantEnabled" default:"false"`
	RpcMetaControlIDontWantEnabled bool `yaml:"rpcMetaControlIDontWantEnabled" default:"false"`
	RpcMetaControlGraftEnabled     bool `yaml:"rpcMetaControlGraftEnabled" default:"false"`
	RpcMetaControlPruneEnabled     bool `yaml:"rpcMetaControlPruneEnabled" default:"false"`
	RpcMetaSubscriptionEnabled     bool `yaml:"rpcMetaSubscriptionEnabled" default:"false"`
	RpcMetaMessageEnabled          bool `yaml:"rpcMetaMessageEnabled" default:"false"`
	AddPeerEnabled                 bool `yaml:"addPeerEnabled" default:"true"`
	RemovePeerEnabled              bool `yaml:"removePeerEnabled" default:"true"`
	ConnectedEnabled               bool `yaml:"connectedEnabled" default:"true"`
	DisconnectedEnabled            bool `yaml:"disconnectedEnabled" default:"true"`
	JoinEnabled                    bool `yaml:"joinEnabled" default:"true"`
	LeaveEnabled                   bool `yaml:"leaveEnabled" default:"false"`
	GraftEnabled                   bool `yaml:"graftEnabled" default:"false"`
	PruneEnabled                   bool `yaml:"pruneEnabled" default:"false"`
	PublishMessageEnabled          bool `yaml:"publishMessageEnabled" default:"false"`
	RejectMessageEnabled           bool `yaml:"rejectMessageEnabled" default:"false"`
	DuplicateMessageEnabled        bool `yaml:"duplicateMessageEnabled" default:"false"`
	DeliverMessageEnabled          bool `yaml:"deliverMessageEnabled" default:"false"`
	HandleMetadataEnabled          bool `yaml:"handleMetadataEnabled" default:"true"`
	HandleStatusEnabled            bool `yaml:"handleStatusEnabled" default:"true"`
	GossipSubBeaconBlockEnabled    bool `yaml:"gossipSubBeaconBlockEnabled" default:"true"`
	GossipSubAttestationEnabled    bool `yaml:"gossipSubAttestationEnabled" default:"true"`
	GossipSubBlobSidecarEnabled    bool `yaml:"gossipSubBlobSidecarEnabled" default:"true"`
}

// Validate validates the event config.
func (e *EventConfig) Validate() error {
	return nil
}
