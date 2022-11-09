package sentry

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Sentry) handleHead(ctx context.Context, event *v1.HeadEvent) error {
	s.log.Debug("Head received")

	meta, err := s.createNewClientMeta(ctx, xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_HEAD)
	if err != nil {
		return err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: meta,
		},
		Event: &xatu.DecoratedEvent_EthV1Head{
			EthV1Head: &xatuethv1.EventHead{
				Slot:                      uint64(event.Slot),
				Block:                     xatuethv1.RootAsString(event.Block),
				State:                     xatuethv1.RootAsString(event.State),
				EpochTransition:           event.EpochTransition,
				PreviousDutyDependentRoot: xatuethv1.RootAsString(event.PreviousDutyDependentRoot),
				CurrentDutyDependentRoot:  xatuethv1.RootAsString(event.CurrentDutyDependentRoot),
			},
		},
	}

	additionalData, err := s.getHeadData(ctx, event, meta)
	if err != nil {
		s.log.WithError(err).Error("Failed to get extra head data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_Head{
			Head: additionalData,
		}
	}

	return s.handleNewDecoratedEvent(ctx, decoratedEvent)
}

//nolint:dupl // Not worth refactoring to save a few lines.
func (s *Sentry) getHeadData(ctx context.Context, event *v1.HeadEvent, meta *xatu.ClientMeta) (*xatu.ClientMeta_AdditionalHeadData, error) {
	extra := &xatu.ClientMeta_AdditionalHeadData{}
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

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
