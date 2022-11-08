package sentry

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleBlock(ctx context.Context, event *v1.BlockEvent) error {
	s.log.Debug("BlockEvent received")

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_BLOCK)
	if err != nil {
		return err
	}

	decoratedEvent := xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Event: &xatu.DecoratedEvent_EthV1Block{
			EthV1Block: &xatuethv1.EventBlock{
				Slot:                uint64(event.Slot),
				Block:               xatuethv1.RootAsString(event.Block),
				ExecutionOptimistic: event.ExecutionOptimistic,
			},
		},
	}

	additionalData, err := s.getBlockData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_Block{
			Block: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getBlockData(ctx context.Context, event *v1.BlockEvent, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalBlockData, error) {
	extra := &xatu.ClientMeta_AdditionalBlockData{}
	eventTime := meta.Event.DateTime.AsTime()

	// Get the wallclock time window for when we saw the event
	slot, epoch, err := s.beacon.Metadata().Wallclock().FromTime(eventTime)
	if err != nil {
		return extra, err
	}

	extra.Slot = &xatu.AdditionalSlotData{
		StartDateTime:   timestamppb.New(slot.TimeWindow().Start()),
		PropagationDiff: uint64(meta.Event.DateTime.AsTime().Sub(slot.TimeWindow().Start()).Milliseconds()),
	}

	extra.Epoch = &xatu.AdditionalEpochData{
		Number:        uint64(epoch.Number()),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
