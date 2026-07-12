package event

import (
	"context"
	"fmt"
	"time"

	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
)

// EventsHeadV3 decorates the beacon-API head_v2 SSE topic (gloas). It is V3
// internally because BEACON_API_ETH_V1_EVENTS_HEAD_V2 is the schema rev of
// the v1 head topic. The topic legitimately fires twice for the same head
// when payload_status transitions from empty to full; payload_status is part
// of the event struct and therefore of the dedup hash, so both firings pass.
type EventsHeadV3 struct {
	log observability.ContextualLogger

	now time.Time

	event          *eth2v1.HeadEventV2
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsHeadV3(log observability.ContextualLogger, event *eth2v1.HeadEventV2, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsHeadV3 {
	return &EventsHeadV3{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_HEAD_V3"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsHeadV3) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V3,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsHeadV3{
			EthV1EventsHeadV3: &xatuethv1.EventHeadV3{
				Slot:                      &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
				Block:                     xatuethv1.RootAsString(e.event.Block),
				State:                     xatuethv1.RootAsString(e.event.State),
				PayloadStatus:             e.event.PayloadStatus,
				EpochTransition:           e.event.EpochTransition,
				CurrentEpochDependentRoot: xatuethv1.RootAsString(e.event.CurrentEpochDependentRoot),
				NextEpochDependentRoot:    xatuethv1.RootAsString(e.event.NextEpochDependentRoot),
				ExecutionOptimistic:       e.event.ExecutionOptimistic,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).WithContext(ctx).Error("Failed to get extra head_v2 data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsHeadV3{
			EthV1EventsHeadV3: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsHeadV3) ShouldIgnore(ctx context.Context) (bool, error) {
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
			hashLogField:               hash,
			timeSinceFirstItemLogField: time.Since(item.Value()),
			slotLogField:               e.event.Slot,
		}).WithContext(ctx).Debug("Duplicate head_v2 event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsHeadV3) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsHeadV3Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsHeadV3Data{}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
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
