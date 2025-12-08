package clmimicry

import (
	"context"
	"encoding/hex"
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

func (p *Processor) handleGossipBlobSidecar(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventBlobSidecar,
) error {
	if payload.BlobSidecar == nil {
		return fmt.Errorf("handleGossipBlobSidecar() called with nil blob sidecar")
	}

	header := payload.BlobSidecar.GetSignedBlockHeader().GetHeader()

	blockRoot, err := header.HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to calculate block header hash tree root: %w", err)
	}

	data := &gossipsub.BlobSidecar{
		Index:         wrapperspb.UInt64(payload.BlobSidecar.GetIndex()),
		Slot:          wrapperspb.UInt64(uint64(header.GetSlot())),
		ProposerIndex: wrapperspb.UInt64(uint64(header.GetProposerIndex())),
		StateRoot:     wrapperspb.String(hex.EncodeToString(header.GetStateRoot())),
		ParentRoot:    wrapperspb.String(hex.EncodeToString(header.GetParentRoot())),
		BlockRoot:     wrapperspb.String(hex.EncodeToString(blockRoot[:])),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubBlobSidecarData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBlobSidecar{
		Libp2PTraceGossipsubBlobSidecar: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar{
			Libp2PTraceGossipsubBlobSidecar: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubBlobSidecarData(
	payload *TraceEventBlobSidecar,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slotNumber := payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetSlot()
	slot := p.wallclock.Slots().FromNumber(uint64(slotNumber))
	epoch := p.wallclock.Epochs().FromSlot(uint64(slotNumber))
	timestampAdjusted := timestamp.Add(p.clockDrift)

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData{
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
			Number:        wrapperspb.UInt64(uint64(slotNumber)),
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
