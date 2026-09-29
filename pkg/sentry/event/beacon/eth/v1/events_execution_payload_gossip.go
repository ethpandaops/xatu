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

// EventsExecutionPayloadGossip handles the EIP-7732 `execution_payload_gossip`
// SSE event — analog of `block_gossip`. It is a flat summary of a revealed
// execution payload, fired when the beacon node first sees it on the gossip
// mesh and it passes gossip validation, before fork-choice import.
type EventsExecutionPayloadGossip struct {
	log observability.ContextualLogger

	now time.Time

	event          *apiv1.ExecutionPayloadEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsExecutionPayloadGossip(log observability.ContextualLogger, event *apiv1.ExecutionPayloadEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsExecutionPayloadGossip {
	return &EventsExecutionPayloadGossip{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_GOSSIP"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsExecutionPayloadGossip) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_GOSSIP,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadGossip{
			EthV1EventsExecutionPayloadGossip: xatuethv1.NewExecutionPayloadEventFromAPIV1(e.event),
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).WithContext(ctx).Error("Failed to get extra execution payload gossip data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsExecutionPayloadGossip{
			EthV1EventsExecutionPayloadGossip: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsExecutionPayloadGossip) ShouldIgnore(ctx context.Context) (bool, error) {
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
			"beacon_block_root":        e.event.BlockRoot.String(),
		}).WithContext(ctx).Debug("Duplicate execution payload gossip event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsExecutionPayloadGossip) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadGossipData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadGossipData{}

	if e.event == nil {
		return extra, fmt.Errorf("execution payload gossip event is nil")
	}

	slotNumber := uint64(e.event.Slot)

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(slotNumber)
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(slotNumber)

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: slotNumber},
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
