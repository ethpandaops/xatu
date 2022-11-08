package sentry

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleChainReOrg(ctx context.Context, event *v1.ChainReorgEvent) error {
	s.log.Debug("ChainReorg received")

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG)
	if err != nil {
		return err
	}

	decoratedEvent := xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Event: &xatu.DecoratedEvent_EthV1ChainReorg{
			EthV1ChainReorg: &xatuethv1.EventChainReorg{
				Slot:         uint64(event.Slot),
				Epoch:        uint64(event.Epoch),
				OldHeadBlock: xatuethv1.RootAsString(event.OldHeadBlock),
				OldHeadState: xatuethv1.RootAsString(event.OldHeadState),
				NewHeadBlock: xatuethv1.RootAsString(event.NewHeadBlock),
				NewHeadState: xatuethv1.RootAsString(event.NewHeadState),
				Depth:        uint64(event.Depth),
			},
		},
	}

	additionalData, err := s.getChainReorgData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_ChainReorg{
			ChainReorg: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

func (s *Sentry) getChainReorgData(ctx context.Context, event *v1.ChainReorgEvent, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalChainReorgData, error) {
	extra := &xatu.ClientMeta_AdditionalChainReorgData{}
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
