package clmimicry

// EventConfig represents configuration for all event types.
type EventConfig struct {
	RecvRPCEnabled                            bool `yaml:"recvRpcEnabled" default:"false"`
	SendRPCEnabled                            bool `yaml:"sendRpcEnabled" default:"false"`
	DropRPCEnabled                            bool `yaml:"dropRpcEnabled" default:"false"`
	RpcMetaControlIHaveEnabled                bool `yaml:"rpcMetaControlIHaveEnabled" default:"false"`
	RpcMetaControlIWantEnabled                bool `yaml:"rpcMetaControlIWantEnabled" default:"false"`
	RpcMetaControlIDontWantEnabled            bool `yaml:"rpcMetaControlIDontWantEnabled" default:"false"`
	RpcMetaControlGraftEnabled                bool `yaml:"rpcMetaControlGraftEnabled" default:"false"`
	RpcMetaControlPruneEnabled                bool `yaml:"rpcMetaControlPruneEnabled" default:"false"`
	RpcMetaSubscriptionEnabled                bool `yaml:"rpcMetaSubscriptionEnabled" default:"false"`
	RpcMetaMessageEnabled                     bool `yaml:"rpcMetaMessageEnabled" default:"false"`
	AddPeerEnabled                            bool `yaml:"addPeerEnabled" default:"true"`
	RemovePeerEnabled                         bool `yaml:"removePeerEnabled" default:"true"`
	ConnectedEnabled                          bool `yaml:"connectedEnabled" default:"true"`
	DisconnectedEnabled                       bool `yaml:"disconnectedEnabled" default:"true"`
	IdentifyEnabled                           bool `yaml:"identifyEnabled" default:"true"`
	SyntheticHeartbeatEnabled                 bool `yaml:"syntheticHeartbeatEnabled" default:"true"`
	JoinEnabled                               bool `yaml:"joinEnabled" default:"true"`
	LeaveEnabled                              bool `yaml:"leaveEnabled" default:"false"`
	GraftEnabled                              bool `yaml:"graftEnabled" default:"false"`
	PruneEnabled                              bool `yaml:"pruneEnabled" default:"false"`
	PublishMessageEnabled                     bool `yaml:"publishMessageEnabled" default:"false"`
	RejectMessageEnabled                      bool `yaml:"rejectMessageEnabled" default:"false"`
	DuplicateMessageEnabled                   bool `yaml:"duplicateMessageEnabled" default:"false"`
	DeliverMessageEnabled                     bool `yaml:"deliverMessageEnabled" default:"false"`
	HandleMetadataEnabled                     bool `yaml:"handleMetadataEnabled" default:"true"`
	HandleStatusEnabled                       bool `yaml:"handleStatusEnabled" default:"true"`
	CustodyProbeEnabled                       bool `yaml:"custodyProbeEnabled" default:"true"`
	GossipSubBeaconBlockEnabled               bool `yaml:"gossipSubBeaconBlockEnabled" default:"true"`
	GossipSubAttestationEnabled               bool `yaml:"gossipSubAttestationEnabled" default:"true"`
	GossipSubAggregateAndProofEnabled         bool `yaml:"gossipSubAggregateAndProofEnabled" default:"true"`
	GossipSubBlobSidecarEnabled               bool `yaml:"gossipSubBlobSidecarEnabled" default:"true"`
	GossipSubDataColumnSidecarEnabled         bool `yaml:"gossipSubDataColumnSidecarEnabled" default:"true"`
	GossipSubExecutionPayloadEnvelopeEnabled  bool `yaml:"gossipSubExecutionPayloadEnvelopeEnabled" default:"true"`
	GossipSubExecutionPayloadBidEnabled       bool `yaml:"gossipSubExecutionPayloadBidEnabled" default:"true"`
	GossipSubPayloadAttestationMessageEnabled bool `yaml:"gossipSubPayloadAttestationMessageEnabled" default:"true"`
	GossipSubProposerPreferencesEnabled       bool `yaml:"gossipSubProposerPreferencesEnabled" default:"true"`
	EngineAPINewPayloadEnabled                bool `yaml:"engineApiNewPayloadEnabled" default:"false"`
	EngineAPIGetBlobsEnabled                  bool `yaml:"engineApiGetBlobsEnabled" default:"false"`

	// Beacon synthetic events (TYSM-instrumented internals, EIP-7732 ePBS).
	BeaconSyntheticPayloadStatusResolvedEnabled           bool `yaml:"beaconSyntheticPayloadStatusResolvedEnabled" default:"true"`
	BeaconSyntheticBuilderPendingPaymentSettlementEnabled bool `yaml:"beaconSyntheticBuilderPendingPaymentSettlementEnabled" default:"true"`
	BeaconSyntheticPayloadAttestationProcessedEnabled     bool `yaml:"beaconSyntheticPayloadAttestationProcessedEnabled" default:"true"`
}

// Validate validates the event config.
func (e *EventConfig) Validate() error {
	return nil
}
