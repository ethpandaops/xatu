package clmimicry

import (
	"time"

	ethtypes "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Block payload builders

// NewPhase0BlockPayload creates a Phase0 block payload.
func NewPhase0BlockPayload(block *ethtypes.SignedBeaconBlock, meta *TraceEventPayloadMetaData) *TraceEventPhase0Block {
	return &TraceEventPhase0Block{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewAltairBlockPayload creates an Altair block payload.
func NewAltairBlockPayload(block *ethtypes.SignedBeaconBlockAltair, meta *TraceEventPayloadMetaData) *TraceEventAltairBlock {
	return &TraceEventAltairBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewBellatrixBlockPayload creates a Bellatrix block payload.
func NewBellatrixBlockPayload(block *ethtypes.SignedBeaconBlockBellatrix, meta *TraceEventPayloadMetaData) *TraceEventBellatrixBlock {
	return &TraceEventBellatrixBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewCapellaBlockPayload creates a Capella block payload.
func NewCapellaBlockPayload(block *ethtypes.SignedBeaconBlockCapella, meta *TraceEventPayloadMetaData) *TraceEventCapellaBlock {
	return &TraceEventCapellaBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewDenebBlockPayload creates a Deneb block payload.
func NewDenebBlockPayload(block *ethtypes.SignedBeaconBlockDeneb, meta *TraceEventPayloadMetaData) *TraceEventDenebBlock {
	return &TraceEventDenebBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewElectraBlockPayload creates an Electra block payload.
func NewElectraBlockPayload(block *ethtypes.SignedBeaconBlockElectra, meta *TraceEventPayloadMetaData) *TraceEventElectraBlock {
	return &TraceEventElectraBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// NewFuluBlockPayload creates a Fulu block payload.
func NewFuluBlockPayload(block *ethtypes.SignedBeaconBlockFulu, meta *TraceEventPayloadMetaData) *TraceEventFuluBlock {
	return &TraceEventFuluBlock{
		TraceEventPayloadMetaData: *meta,
		Block:                     block,
	}
}

// Attestation payload builders

// NewAttestationPayload creates a pre-Electra attestation payload.
func NewAttestationPayload(att *ethtypes.Attestation, meta *TraceEventPayloadMetaData) *TraceEventAttestation {
	return &TraceEventAttestation{
		TraceEventPayloadMetaData: *meta,
		Attestation:               att,
	}
}

// NewAttestationElectraPayload creates an Electra attestation payload.
func NewAttestationElectraPayload(att *ethtypes.AttestationElectra, meta *TraceEventPayloadMetaData) *TraceEventAttestationElectra {
	return &TraceEventAttestationElectra{
		TraceEventPayloadMetaData: *meta,
		AttestationElectra:        att,
	}
}

// NewSingleAttestationPayload creates a single attestation payload.
func NewSingleAttestationPayload(att *ethtypes.SingleAttestation, meta *TraceEventPayloadMetaData) *TraceEventSingleAttestation {
	return &TraceEventSingleAttestation{
		TraceEventPayloadMetaData: *meta,
		SingleAttestation:         att,
	}
}

// Aggregate attestation payload builders

// NewSignedAggregateAttestationAndProofPayload creates a pre-Electra aggregate attestation payload.
func NewSignedAggregateAttestationAndProofPayload(agg *ethtypes.SignedAggregateAttestationAndProof, meta *TraceEventPayloadMetaData) *TraceEventSignedAggregateAttestationAndProof {
	return &TraceEventSignedAggregateAttestationAndProof{
		TraceEventPayloadMetaData:          *meta,
		SignedAggregateAttestationAndProof: agg,
	}
}

// NewSignedAggregateAttestationAndProofElectraPayload creates an Electra aggregate attestation payload.
func NewSignedAggregateAttestationAndProofElectraPayload(agg *ethtypes.SignedAggregateAttestationAndProofElectra, meta *TraceEventPayloadMetaData) *TraceEventSignedAggregateAttestationAndProofElectra {
	return &TraceEventSignedAggregateAttestationAndProofElectra{
		TraceEventPayloadMetaData:                 *meta,
		SignedAggregateAttestationAndProofElectra: agg,
	}
}

// Sidecar payload builders

// NewBlobSidecarPayload creates a blob sidecar payload.
func NewBlobSidecarPayload(blob *ethtypes.BlobSidecar, meta *TraceEventPayloadMetaData) *TraceEventBlobSidecar {
	return &TraceEventBlobSidecar{
		TraceEventPayloadMetaData: *meta,
		BlobSidecar:               blob,
	}
}

// NewDataColumnSidecarPayload creates a data column sidecar payload.
func NewDataColumnSidecarPayload(dataColumn *ethtypes.DataColumnSidecar, meta *TraceEventPayloadMetaData) *TraceEventDataColumnSidecar {
	return &TraceEventDataColumnSidecar{
		TraceEventPayloadMetaData: *meta,
		DataColumnSidecar:         dataColumn,
	}
}

// Custody probe payload builder

// NewCustodyProbePayload creates a custody probe payload.
func NewCustodyProbePayload(
	peerID *peer.ID,
	epoch, slot, column uint64,
	blockHash, result, errorStr string,
	duration time.Duration,
	columnSize int,
) *TraceEventCustodyProbe {
	return &TraceEventCustodyProbe{
		PeerID:     peerID,
		Epoch:      epoch,
		Slot:       slot,
		BlockHash:  blockHash,
		Column:     column,
		Result:     result,
		Duration:   duration,
		ColumnSize: columnSize,
		Error:      errorStr,
	}
}

// Consensus engine API payload builder

// NewConsensusEngineAPINewPayloadPayload creates a consensus engine API new payload event.
func NewConsensusEngineAPINewPayloadPayload(
	requestedAt time.Time,
	duration time.Duration,
	slot, proposerIndex uint64,
	blockRoot, parentBlockRoot string,
	blockNumber uint64,
	blockHash, parentHash string,
	gasUsed, gasLimit uint64,
	txCount, blobCount uint32,
	status, latestValidHash, validationError string,
	methodVersion string,
) *TraceEventConsensusEngineAPINewPayload {
	return &TraceEventConsensusEngineAPINewPayload{
		RequestedAt:     requestedAt,
		Duration:        duration,
		Slot:            slot,
		BlockRoot:       blockRoot,
		ParentBlockRoot: parentBlockRoot,
		ProposerIndex:   proposerIndex,
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		ParentHash:      parentHash,
		GasUsed:         gasUsed,
		GasLimit:        gasLimit,
		TxCount:         txCount,
		BlobCount:       blobCount,
		Status:          status,
		LatestValidHash: latestValidHash,
		ValidationError: validationError,
		MethodVersion:   methodVersion,
	}
}
