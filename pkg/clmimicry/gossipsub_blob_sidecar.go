package clmimicry

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/probe-lab/hermes/eth"
	"github.com/probe-lab/hermes/host"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (m *Mimicry) handleGossipBlobSidecar(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *host.TraceEvent,
	payload *eth.TraceEventBlobSidecar,
) error {
	if payload.BlobSidecar == nil {
		return fmt.Errorf("handleGossipBlobSidecar() called with nil blob sidecar")
	}

	data := &gossipsub.BlobSidecar{
		Index:         wrapperspb.UInt64(payload.BlobSidecar.GetIndex()),
		Slot:          wrapperspb.UInt64(uint64(payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetSlot())),
		ProposerIndex: wrapperspb.UInt64(uint64(payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetProposerIndex())),
		StateRoot:     wrapperspb.String(hex.EncodeToString(payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetStateRoot())),
		ParentRoot:    wrapperspb.String(hex.EncodeToString(payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetParentRoot())),
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := m.createAdditionalGossipSubBlobSidecarData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubBlobSidecar{
		Libp2PTraceGossipsubBlobSidecar: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
			DateTime: timestamppb.New(event.Timestamp.Add(m.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar{
			Libp2PTraceGossipsubBlobSidecar: data,
		},
	}

	return m.handleNewDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (m *Mimicry) createAdditionalGossipSubBlobSidecarData(
	payload *eth.TraceEventBlobSidecar,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubBlobSidecarData, error) {
	wallclockSlot, wallclockEpoch, err := m.ethereum.Metadata().Wallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slotNumber := payload.BlobSidecar.GetSignedBlockHeader().GetHeader().GetSlot()
	slot := m.ethereum.Metadata().Wallclock().Slots().FromNumber(uint64(slotNumber))
	epoch := m.ethereum.Metadata().Wallclock().Epochs().FromSlot(uint64(slotNumber))
	timestampAdjusted := timestamp.Add(m.clockDrift)

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
