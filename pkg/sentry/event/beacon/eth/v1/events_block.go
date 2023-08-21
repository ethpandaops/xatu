package event

import (
	"context"
	"fmt"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type EventsBlock struct {
	log logrus.FieldLogger

	now time.Time

	event          *eth2v1.BlockEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsBlock(log logrus.FieldLogger, event *eth2v1.BlockEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsBlock {
	return &EventsBlock{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_BLOCK"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsBlock) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsBlock{
			EthV1EventsBlock: &xatuethv1.EventBlock{
				Slot:                uint64(e.event.Slot),
				SlotV2:              &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
				Block:               xatuethv1.RootAsString(e.event.Block),
				ExecutionOptimistic: e.event.ExecutionOptimistic,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsBlock{
			EthV1EventsBlock: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsBlock) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.DefaultTTL)
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"slot":                  e.event.Slot,
		}).Debug("Duplicate block event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsBlock) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsBlockData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsBlockData{}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.Slot{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        uint64(e.event.Slot),
		NumberV2:      &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		NumberV2:      &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.Propagation{
		SlotStartDiff: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
		SlotStartDiffV2: &wrapperspb.UInt64Value{
			Value: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
		},
	}

	return extra, nil
}
