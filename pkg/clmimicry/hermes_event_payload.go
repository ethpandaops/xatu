// Credit: github.com/probe-lab/hermes
// Source: hermes/eth/events/output_full.go
// These types were extracted from the Hermes P2P network monitoring tool.

package clmimicry

import (
	"time"

	ethtypes "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Trace event payload types extracted from github.com/probe-lab/hermes/eth/events.

// TraceEventPhase0Block represents a Phase0 beacon block event.
type TraceEventPhase0Block struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlock
}

// TraceEventAltairBlock represents an Altair beacon block event.
type TraceEventAltairBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockAltair
}

// TraceEventBellatrixBlock represents a Bellatrix beacon block event.
type TraceEventBellatrixBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockBellatrix
}

// TraceEventCapellaBlock represents a Capella beacon block event.
type TraceEventCapellaBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockCapella
}

// TraceEventDenebBlock represents a Deneb beacon block event.
type TraceEventDenebBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockDeneb
}

// TraceEventElectraBlock represents an Electra beacon block event.
type TraceEventElectraBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockElectra
}

// TraceEventFuluBlock represents a Fulu beacon block event.
type TraceEventFuluBlock struct {
	TraceEventPayloadMetaData
	Block *ethtypes.SignedBeaconBlockFulu
}

// TraceEventAttestation represents an attestation event.
type TraceEventAttestation struct {
	TraceEventPayloadMetaData
	Attestation *ethtypes.Attestation
}

// TraceEventAttestationElectra represents an Electra attestation event.
type TraceEventAttestationElectra struct {
	TraceEventPayloadMetaData
	AttestationElectra *ethtypes.AttestationElectra
}

// TraceEventSingleAttestation represents a single attestation event.
type TraceEventSingleAttestation struct {
	TraceEventPayloadMetaData
	SingleAttestation *ethtypes.SingleAttestation
}

// TraceEventSignedAggregateAttestationAndProof represents a signed aggregate attestation and proof event.
type TraceEventSignedAggregateAttestationAndProof struct {
	TraceEventPayloadMetaData
	SignedAggregateAttestationAndProof *ethtypes.SignedAggregateAttestationAndProof
}

// TraceEventSignedAggregateAttestationAndProofElectra represents an Electra signed aggregate attestation and proof event.
type TraceEventSignedAggregateAttestationAndProofElectra struct {
	TraceEventPayloadMetaData
	SignedAggregateAttestationAndProofElectra *ethtypes.SignedAggregateAttestationAndProofElectra
}

// TraceEventSignedContributionAndProof represents a signed contribution and proof event.
type TraceEventSignedContributionAndProof struct {
	TraceEventPayloadMetaData
	SignedContributionAndProof *ethtypes.SignedContributionAndProof
}

// TraceEventVoluntaryExit represents a voluntary exit event.
type TraceEventVoluntaryExit struct {
	TraceEventPayloadMetaData
	VoluntaryExit *ethtypes.VoluntaryExit
}

// TraceEventSyncCommitteeMessage represents a sync committee message event.
type TraceEventSyncCommitteeMessage struct {
	TraceEventPayloadMetaData
	SyncCommitteeMessage *ethtypes.SyncCommitteeMessage //nolint:staticcheck // gRPC API deprecated but still supported until v8 (2026)
}

// TraceEventBLSToExecutionChange represents a BLS to execution change event.
type TraceEventBLSToExecutionChange struct {
	TraceEventPayloadMetaData
	BLSToExecutionChange *ethtypes.BLSToExecutionChange
}

// TraceEventBlobSidecar represents a blob sidecar event.
type TraceEventBlobSidecar struct {
	TraceEventPayloadMetaData
	BlobSidecar *ethtypes.BlobSidecar
}

// TraceEventProposerSlashing represents a proposer slashing event.
type TraceEventProposerSlashing struct {
	TraceEventPayloadMetaData
	ProposerSlashing *ethtypes.ProposerSlashing
}

// TraceEventAttesterSlashing represents an attester slashing event.
type TraceEventAttesterSlashing struct {
	TraceEventPayloadMetaData
	AttesterSlashing *ethtypes.AttesterSlashing
}

// TraceEventDataColumnSidecar represents a data column sidecar event.
type TraceEventDataColumnSidecar struct {
	TraceEventPayloadMetaData
	DataColumnSidecar *ethtypes.DataColumnSidecar
}

// TraceEventCustodyProbe represents a data column custody probe event.
//
//nolint:tagliatelle // JSON tags match Hermes format for compatibility
type TraceEventCustodyProbe struct {
	TraceEventPayloadMetaData
	PeerID     *peer.ID      `json:"peer_id,omitempty"`
	Epoch      uint64        `json:"epoch"`
	Slot       uint64        `json:"slot"`
	BlockHash  string        `json:"block_hash"`
	Column     uint64        `json:"column_id"`
	Result     string        `json:"result,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
	ColumnSize int           `json:"column_size,omitempty"`
	Error      string        `json:"error,omitempty"`
}
