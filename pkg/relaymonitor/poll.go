package relaymonitor

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func (r *RelayMonitor) scheduleBidTraceFetchingAtSlotTime(ctx context.Context, at time.Duration, client *relay.Client) {
	offset := at

	logCtx := r.log.
		WithField("proccer", "at_slot_time").
		WithField("slot_time", offset.String()).
		WithField("relay", client.Name())

	logCtx.Debug("Scheduling bid trace fetching at slot time")

	r.ethereum.Wallclock().OnSlotChanged(func(slot ethwallclock.Slot) {
		time.Sleep(offset)

		err := r.fetchBidTraces(ctx, client, phase0.Slot(slot.Number()))
		if err != nil {
			logCtx.
				WithField("slot_time", offset.String()).
				WithField("slot", slot.Number()).
				WithError(err).Error("Failed to fetch bid traces")
		}
	})
}

func (r *RelayMonitor) fetchBidTraces(ctx context.Context, client *relay.Client, slot phase0.Slot) error {
	requestedAt := time.Now()

	bids, err := client.GetBids(ctx, url.Values{
		"slot":  {strconv.FormatUint(uint64(slot), 10)},
		"limit": {strconv.FormatUint(uint64(500), 10)},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get bids")
	}

	responseAt := time.Now()

	for _, payload := range bids {
		if r.bidCache.Has(client.Name(), slot, payload.BlockHash.GetValue()) {
			continue
		}

		r.bidCache.Set(client.Name(), slot, payload.BlockHash.GetValue())

		event, err := r.createNewDecoratedEvent(ctx, client, slot, payload, requestedAt, responseAt)
		if err != nil {
			return errors.Wrap(err, "failed to create new decorated event")
		}

		err = r.handleNewDecoratedEvent(ctx, event)
		if err != nil {
			r.log.WithError(err).Error("Failed to handle new decorated event")
		}
	}

	return nil

}

func (r *RelayMonitor) createNewDecoratedEvent(
	ctx context.Context,
	re *relay.Client,
	slot phase0.Slot,
	payload *mevrelay.BidTrace,
	requestedAt time.Time,
	responseAt time.Time,
) (*xatu.DecoratedEvent, error) {
	clientMeta, err := r.createNewClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get client meta")
	}

	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	additionalData, err := r.createAdditionalMevRelayBidTraceData(re, slot, requestedAt, responseAt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create additional mev relay bid trace data")
	}

	metadata.AdditionalData = &xatu.ClientMeta_MevRelayBidTraceBuilderBlockSubmission{
		MevRelayBidTraceBuilderBlockSubmission: additionalData,
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
			DateTime: timestamppb.New(requestedAt.Add(r.clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_MevRelayBidTraceBuilderBlockSubmission{
			MevRelayBidTraceBuilderBlockSubmission: payload,
		},
	}

	return decoratedEvent, nil
}

func (r *RelayMonitor) createAdditionalMevRelayBidTraceData(
	re *relay.Client,
	slot phase0.Slot,
	requestedAt time.Time,
	responseAt time.Time,
) (*xatu.ClientMeta_AdditionalMevRelayBidTraceBuilderBlockSubmissionData, error) {
	slotInBid := r.ethereum.Wallclock().Slots().FromNumber(uint64(slot))
	epochFromSlot := r.ethereum.Wallclock().Epochs().FromSlot(uint64(slot))

	wallclockSlot := r.ethereum.Wallclock().Slots().FromTime(requestedAt)
	wallclockEpoch := r.ethereum.Wallclock().Epochs().FromTime(requestedAt)

	requestedAtDiff := uint64(requestedAt.Sub(wallclockSlot.TimeWindow().Start()).Milliseconds())
	responseAtDiff := uint64(responseAt.Sub(wallclockSlot.TimeWindow().Start()).Milliseconds())

	return &xatu.ClientMeta_AdditionalMevRelayBidTraceBuilderBlockSubmissionData{
		Relay: &mevrelay.Relay{
			Url:  wrapperspb.String(re.URL()),
			Name: wrapperspb.String(re.Name()),
		},
		Slot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(uint64(slot)),
			StartDateTime: timestamppb.New(slotInBid.TimeWindow().Start()),
		},
		Epoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(epochFromSlot.Number()),
			StartDateTime: timestamppb.New(epochFromSlot.TimeWindow().Start()),
		},
		WallclockSlot: &xatu.SlotV2{
			Number:        wrapperspb.UInt64(wallclockSlot.Number()),
			StartDateTime: timestamppb.New(wallclockSlot.TimeWindow().Start()),
		},
		WallclockEpoch: &xatu.EpochV2{
			Number:        wrapperspb.UInt64(wallclockEpoch.Number()),
			StartDateTime: timestamppb.New(wallclockEpoch.TimeWindow().Start()),
		},
		RequestedAtSlotTime: wrapperspb.UInt64(requestedAtDiff),
		ResponseAtSlotTime:  wrapperspb.UInt64(responseAtDiff),
	}, nil
}
