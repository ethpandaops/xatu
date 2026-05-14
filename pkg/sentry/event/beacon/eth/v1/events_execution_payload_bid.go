package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/gloas"
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

// EventsExecutionPayloadBid handles the EIP-7732 `execution_payload_bid` SSE
// event — the builder's signed bid for the upcoming slot's execution payload.
type EventsExecutionPayloadBid struct {
	log observability.ContextualLogger

	now time.Time

	event          *gloas.SignedExecutionPayloadBid
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsExecutionPayloadBid(log observability.ContextualLogger, event *gloas.SignedExecutionPayloadBid, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsExecutionPayloadBid {
	return &EventsExecutionPayloadBid{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_BID"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsExecutionPayloadBid) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_BID,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadBid{
			EthV1EventsExecutionPayloadBid: xatuethv1.NewSignedExecutionPayloadBidFromGloas(e.event),
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).WithContext(ctx).Error("Failed to get extra execution payload bid data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsExecutionPayloadBid{
			EthV1EventsExecutionPayloadBid: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsExecutionPayloadBid) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	if e.event == nil || e.event.Message == nil {
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
			slotLogField:               e.event.Message.Slot,
			"builder_index":            e.event.Message.BuilderIndex,
			"block_hash":               e.event.Message.BlockHash.String(),
		}).WithContext(ctx).Debug("Duplicate execution payload bid event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsExecutionPayloadBid) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadBidData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadBidData{}

	if e.event == nil || e.event.Message == nil {
		return extra, fmt.Errorf("execution payload bid missing message")
	}

	slotNumber := uint64(e.event.Message.Slot)

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
