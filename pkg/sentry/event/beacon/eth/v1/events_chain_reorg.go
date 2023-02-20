package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventsChainReorg struct {
	log logrus.FieldLogger

	now time.Time

	event          *eth2v1.ChainReorgEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
}

func NewEventsChainReorg(log logrus.FieldLogger, event *eth2v1.ChainReorgEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsChainReorg {
	return &EventsChainReorg{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_CHAIN_REORG"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
	}
}

func (e *EventsChainReorg) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	ignore, err := e.shouldIgnore(ctx)
	if err != nil {
		return nil, err
	}

	if ignore {
		return nil, errors.New("duplicate event")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
			DateTime: timestamppb.New(e.now),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsChainReorg{
			EthV1EventsChainReorg: &xatuethv1.EventChainReorg{
				Slot:         uint64(e.event.Slot),
				Epoch:        uint64(e.event.Epoch),
				OldHeadBlock: xatuethv1.RootAsString(e.event.OldHeadBlock),
				OldHeadState: xatuethv1.RootAsString(e.event.OldHeadState),
				NewHeadBlock: xatuethv1.RootAsString(e.event.NewHeadBlock),
				NewHeadState: xatuethv1.RootAsString(e.event.NewHeadState),
				Depth:        e.event.Depth,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra chain reorg data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsChainReorg{
			EthV1EventsChainReorg: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsChainReorg) shouldIgnore(ctx context.Context) (bool, error) {
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
		}).Debug("Duplicate chain reorg event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsChainReorg) getAdditionalData(ctx context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsChainReorgData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsChainReorgData{}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.Slot{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.Propagation{
		SlotStartDiff: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
	}

	return extra, nil
}
