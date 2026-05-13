package event

import (
	"context"
	"fmt"
	"time"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// FastConfirmationData is the parsed payload of a fast_confirmation SSE event.
type FastConfirmationData struct {
	Slot  uint64
	Block string
}

type EventsFastConfirmation struct {
	log logrus.FieldLogger

	now time.Time

	event          *FastConfirmationData
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsFastConfirmation(log logrus.FieldLogger, event *FastConfirmationData, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsFastConfirmation {
	return &EventsFastConfirmation{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsFastConfirmation) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FAST_CONFIRMATION,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsFastConfirmation{
			EthV1EventsFastConfirmation: &xatuethv1.EventFastConfirmation{
				Slot:  &wrapperspb.UInt64Value{Value: e.event.Slot},
				Block: e.event.Block,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra fast confirmation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsFastConfirmation{
			EthV1EventsFastConfirmation: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsFastConfirmation) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,                     //nolint:goconst // matches existing event handler style
			"time_since_first_item": time.Since(item.Value()), //nolint:goconst // matches existing event handler style
			"slot":                  e.event.Slot,             //nolint:goconst // matches existing event handler style
		}).Debug("Duplicate fast confirmation event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsFastConfirmation) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsFastConfirmationData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsFastConfirmationData{}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(e.event.Slot)
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(e.event.Slot)

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: e.event.Slot},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: &wrapperspb.UInt64Value{
			//nolint:gosec // not concerned in reality
			Value: uint64(e.now.Sub(slot.TimeWindow().Start()).Milliseconds()),
		},
	}

	return extra, nil
}
