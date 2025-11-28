package clmimicry

// Gossipsub event types that wrap specific payload types from hermes_event_payload.go.
// Each event embeds TraceEventBase and implements the TopicEvent interface.

// BeaconBlockEvent represents a beacon block gossipsub event.
// The Payload can be one of the various block types depending on the fork.
type BeaconBlockEvent struct {
	TraceEventBase
	Topic   string
	MsgID   string
	Payload any // Union of TraceEvent*Block types from different forks
}

// GetTopic returns the gossipsub topic for this event.
func (e *BeaconBlockEvent) GetTopic() string {
	return e.Topic
}

// GetMsgID returns the message ID for this event.
func (e *BeaconBlockEvent) GetMsgID() string {
	return e.MsgID
}

var _ TopicEvent = (*BeaconBlockEvent)(nil)
var _ MessageEvent = (*BeaconBlockEvent)(nil)

// AttestationEvent represents an attestation gossipsub event.
// The Payload can be TraceEventAttestation, TraceEventAttestationElectra, or TraceEventSingleAttestation.
type AttestationEvent struct {
	TraceEventBase
	Topic   string
	MsgID   string
	Payload any // Union of attestation types
}

// GetTopic returns the gossipsub topic for this event.
func (e *AttestationEvent) GetTopic() string {
	return e.Topic
}

// GetMsgID returns the message ID for this event.
func (e *AttestationEvent) GetMsgID() string {
	return e.MsgID
}

var _ TopicEvent = (*AttestationEvent)(nil)
var _ MessageEvent = (*AttestationEvent)(nil)

// AggregateAndProofEvent represents an aggregate and proof gossipsub event.
// The Payload can be TraceEventSignedAggregateAttestationAndProof or TraceEventSignedAggregateAttestationAndProofElectra.
type AggregateAndProofEvent struct {
	TraceEventBase
	Topic   string
	MsgID   string
	Payload any // Union of aggregate and proof types
}

// GetTopic returns the gossipsub topic for this event.
func (e *AggregateAndProofEvent) GetTopic() string {
	return e.Topic
}

// GetMsgID returns the message ID for this event.
func (e *AggregateAndProofEvent) GetMsgID() string {
	return e.MsgID
}

var _ TopicEvent = (*AggregateAndProofEvent)(nil)
var _ MessageEvent = (*AggregateAndProofEvent)(nil)

// BlobSidecarEvent represents a blob sidecar gossipsub event.
type BlobSidecarEvent struct {
	TraceEventBase
	Topic       string
	MsgID       string
	BlobSidecar *TraceEventBlobSidecar
}

// GetTopic returns the gossipsub topic for this event.
func (e *BlobSidecarEvent) GetTopic() string {
	return e.Topic
}

// GetMsgID returns the message ID for this event.
func (e *BlobSidecarEvent) GetMsgID() string {
	return e.MsgID
}

var _ TopicEvent = (*BlobSidecarEvent)(nil)
var _ MessageEvent = (*BlobSidecarEvent)(nil)

// DataColumnSidecarEvent represents a data column sidecar gossipsub event.
type DataColumnSidecarEvent struct {
	TraceEventBase
	Topic             string
	MsgID             string
	DataColumnSidecar *TraceEventDataColumnSidecar
}

// GetTopic returns the gossipsub topic for this event.
func (e *DataColumnSidecarEvent) GetTopic() string {
	return e.Topic
}

// GetMsgID returns the message ID for this event.
func (e *DataColumnSidecarEvent) GetMsgID() string {
	return e.MsgID
}

var _ TopicEvent = (*DataColumnSidecarEvent)(nil)
var _ MessageEvent = (*DataColumnSidecarEvent)(nil)
