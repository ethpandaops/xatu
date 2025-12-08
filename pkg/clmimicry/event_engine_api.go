package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// EngineAPINewPayloadData contains the data for an engine_newPayload call event.
type EngineAPINewPayloadData struct {
	// Timing
	RequestedAt time.Time
	DurationMs  uint64

	// Beacon context
	Slot            uint64
	BlockRoot       string
	ParentBlockRoot string
	ProposerIndex   uint64

	// Execution payload
	BlockNumber uint64
	BlockHash   string
	ParentHash  string
	GasUsed     uint64
	GasLimit    uint64
	TxCount     uint32
	BlobCount   uint32

	// Response
	Status          string
	LatestValidHash string
	ValidationError string

	// Meta
	MethodVersion string
}

// HandleEngineAPINewPayload processes an engine_newPayload call event and sends it to the output.
// This method is intended for external callers who want to instrument engine API calls.
func (p *Processor) HandleEngineAPINewPayload(ctx context.Context, data *EngineAPINewPayloadData) error {
	if !p.events.EngineAPINewPayloadEnabled {
		return nil
	}

	clientMeta, err := p.metaProvider.GetClientMeta(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client meta: %w", err)
	}

	networkStr := getNetworkID(clientMeta)
	p.metrics.AddEvent(xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD.String(), networkStr)

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	// Derive additional data (slot/epoch info)
	extra := p.deriveAdditionalDataForEngineAPINewPayload(data)
	metadata.AdditionalData = &xatu.ClientMeta_ConsensusEngineApiNewPayload{
		ConsensusEngineApiNewPayload: extra,
	}

	now := time.Now().Add(p.clockDrift)

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_ConsensusEngineApiNewPayload{
			ConsensusEngineApiNewPayload: &xatu.ConsensusEngineAPINewPayload{
				RequestedAt:     timestamppb.New(data.RequestedAt),
				DurationMs:      wrapperspb.UInt64(data.DurationMs),
				Slot:            wrapperspb.UInt64(data.Slot),
				BlockRoot:       data.BlockRoot,
				ParentBlockRoot: data.ParentBlockRoot,
				ProposerIndex:   wrapperspb.UInt64(data.ProposerIndex),
				BlockNumber:     wrapperspb.UInt64(data.BlockNumber),
				BlockHash:       data.BlockHash,
				ParentHash:      data.ParentHash,
				GasUsed:         wrapperspb.UInt64(data.GasUsed),
				GasLimit:        wrapperspb.UInt64(data.GasLimit),
				TxCount:         wrapperspb.UInt32(data.TxCount),
				BlobCount:       wrapperspb.UInt32(data.BlobCount),
				Status:          data.Status,
				LatestValidHash: data.LatestValidHash,
				ValidationError: data.ValidationError,
				MethodVersion:   data.MethodVersion,
			},
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

func (p *Processor) deriveAdditionalDataForEngineAPINewPayload(data *EngineAPINewPayloadData) *xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData {
	extra := &xatu.ClientMeta_AdditionalConsensusEngineAPINewPayloadData{}

	slot := p.wallclock.Slots().FromNumber(data.Slot)
	epoch := p.wallclock.Epochs().FromSlot(data.Slot)

	extra.Slot = &xatu.SlotV2{
		Number:        wrapperspb.UInt64(slot.Number()),
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        wrapperspb.UInt64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra
}
