package clmimicry

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p/gossipsub"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func (p *Processor) handleGossipInclusionList(
	ctx context.Context,
	clientMeta *xatu.ClientMeta,
	event *TraceEvent,
	payload *TraceEventInclusionList,
) error {
	var totalBytes uint64

	hashes := make([]string, 0, len(payload.Transactions))

	for _, tx := range payload.Transactions {
		totalBytes += uint64(len(tx))
		hashes = append(hashes, crypto.Keccak256Hash(tx).Hex())
	}

	data := &gossipsub.InclusionList{
		Slot:                   wrapperspb.UInt64(payload.Slot),
		ValidatorIndex:         wrapperspb.UInt64(payload.ValidatorIndex),
		DependentRoot:          wrapperspb.String(fmt.Sprintf("0x%x", payload.DependentRoot)),
		TransactionsCount:      wrapperspb.UInt32(uint32(len(payload.Transactions))), //nolint:gosec // bounded by MAX_BYTES_PER_INCLUSION_LIST.
		TransactionsTotalBytes: wrapperspb.UInt64(totalBytes),
		TransactionHashes:      hashes,
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := p.createAdditionalGossipSubInclusionListData(payload, event.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create additional data: %w", err)
	}

	metadata.AdditionalData = &xatu.ClientMeta_Libp2PTraceGossipsubInclusionList{
		Libp2PTraceGossipsubInclusionList: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_GOSSIPSUB_INCLUSION_LIST,
			DateTime: timestamppb.New(event.Timestamp.Add(p.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceGossipsubInclusionList{
			Libp2PTraceGossipsubInclusionList: data,
		},
	}

	return p.output.HandleDecoratedEvent(ctx, decoratedEvent)
}

//nolint:gosec // int -> uint32 common conversion pattern in xatu.
func (p *Processor) createAdditionalGossipSubInclusionListData(
	payload *TraceEventInclusionList,
	timestamp time.Time,
) (*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubInclusionListData, error) {
	wallclockSlot, wallclockEpoch, err := p.wallclock.FromTime(timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallclock time: %w", err)
	}

	slot := p.wallclock.Slots().FromNumber(payload.Slot)
	epoch := p.wallclock.Epochs().FromSlot(payload.Slot)
	timestampAdjusted := timestamp.Add(p.clockDrift)

	return &xatu.ClientMeta_AdditionalLibP2PTraceGossipSubInclusionListData{
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
			Number:        wrapperspb.UInt64(payload.Slot),
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
