package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleGossipExecutionPayloadBid(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventExecutionPayloadBid,
) error {
	if payload.ExecutionPayloadBid == nil {
		return fmt.Errorf("handleGossipExecutionPayloadBid() called with nil bid")
	}

	bid := payload.ExecutionPayloadBid.GetMessage()
	if bid == nil {
		return fmt.Errorf("handleGossipExecutionPayloadBid() called with nil bid message")
	}

	data := &gossipsub.ExecutionPayloadBid{
		Slot:                   wrapperspb.UInt64(uint64(bid.GetSlot())),
		BuilderIndex:           wrapperspb.UInt64(uint64(bid.GetBuilderIndex())),
		BlockHash:              wrapperspb.String(fmt.Sprintf("0x%x", bid.GetBlockHash())),
		ParentBlockHash:        wrapperspb.String(fmt.Sprintf("0x%x", bid.GetParentBlockHash())),
		Value:                  wrapperspb.UInt64(uint64(bid.GetValue())),
		ExecutionPayment:       wrapperspb.UInt64(uint64(bid.GetExecutionPayment())),
		FeeRecipient:           wrapperspb.String(fmt.Sprintf("0x%x", bid.GetFeeRecipient())),
		GasLimit:               wrapperspb.UInt64(bid.GetGasLimit()),
		BlobKzgCommitmentCount: wrapperspb.UInt32(uint32(len(bid.GetBlobKzgCommitments()))), //nolint:gosec // bounded by MAX_BLOB_COMMITMENTS_PER_BLOCK
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubExecutionPayloadBidData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubExecutionPayloadBid{
		Libp2PTraceGossipsubExecutionPayloadBid: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadBid{
			Libp2PTraceGossipsubExecutionPayloadBid: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubExecutionPayloadBidData(
	payload *TraceEventExecutionPayloadBid,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadBidData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slotNumber := uint64(payload.ExecutionPayloadBid.GetMessage().GetSlot())
	slot := p.wallclock.Slots().FromNumber(slotNumber)
	epoch := p.wallclock.Epochs().FromSlot(slotNumber)
	timestampAdjusted := timestamp.Add(p.clockDrift)

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubExecutionPayloadBidData{
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epoch.Number()),
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(slotNumber),
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Propagation: &xatu.PropagationV2{
			SlotStartDiff: &wrapperspb.UInt64Value{
				Value: uint64(timestampAdjusted.Sub(slot.TimeWindow().Start()).Milliseconds()),
			},
		},
		Metadata:    &libp2p.TraceEventMetadata{PeerId: wrapperspb.String(payload.PeerID)},
		Topic:       wrapperspb.String(payload.Topic),
		MessageSize: wrapperspb.UInt32(uint32(payload.MsgSize)),
		MessageId:   wrapperspb.String(payload.MsgID),
	}, nil
}
