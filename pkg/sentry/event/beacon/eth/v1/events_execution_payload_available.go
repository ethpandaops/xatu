package event

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/xatu/pkg/observability"
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

// EventsExecutionPayloadAvailable handles the EIP-7732
// `execution_payload_available` SSE event — a lightweight signal that the
// beacon node has confirmed the payload + blobs are locally available, ready
// for the PTC to vote payload_present = true. Carries only block_root + slot.
type EventsExecutionPayloadAvailable struct {
	log observability.ContextualLogger

	now time.Time

	event          *apiv1.ExecutionPayloadAvailableEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsExecutionPayloadAvailable(log observability.ContextualLogger, event *apiv1.ExecutionPayloadAvailableEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsExecutionPayloadAvailable {
	return &EventsExecutionPayloadAvailable{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_AVAILABLE"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsExecutionPayloadAvailable) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_AVAILABLE,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadAvailable{
			EthV1EventsExecutionPayloadAvailable: xatuethv1.NewExecutionPayloadAvailableFromAPIV1(e.event),
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).WithContext(ctx).Error("Failed to get extra execution payload available data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsExecutionPayloadAvailable{
			EthV1EventsExecutionPayloadAvailable: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsExecutionPayloadAvailable) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	if e.event == nil {
		return true, nil
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			hashLogField:               hash,
			timeSinceFirstItemLogField: time.Since(item.Value()),
			slotLogField:               e.event.Slot,
		}).WithContext(ctx).Debug("Duplicate execution payload available event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsExecutionPayloadAvailable) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadAvailableData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadAvailableData{}

	if e.event == nil {
		return extra, fmt.Errorf("execution payload available event is nil")
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
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
