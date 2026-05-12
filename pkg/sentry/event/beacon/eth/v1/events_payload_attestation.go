package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/gloas"
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

// EventsPayloadAttestation handles the EIP-7732 `payload_attestation_message`
// SSE event — an individual PTC validator's payload attestation. ~512 messages
// per slot, high volume.
type EventsPayloadAttestation struct {
	log logrus.FieldLogger

	now time.Time

	event          *gloas.PayloadAttestationMessage
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsPayloadAttestation(log logrus.FieldLogger, event *gloas.PayloadAttestationMessage, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsPayloadAttestation {
	return &EventsPayloadAttestation{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_PAYLOAD_ATTESTATION"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsPayloadAttestation) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_PAYLOAD_ATTESTATION,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsPayloadAttestation{
			EthV1EventsPayloadAttestation: xatuethv1.NewPayloadAttestationMessageFromGloas(e.event),
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra payload attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsPayloadAttestation{
			EthV1EventsPayloadAttestation: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsPayloadAttestation) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	if e.event == nil || e.event.Data == nil {
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
			"validator_index":          e.event.ValidatorIndex,
			slotLogField:               e.event.Data.Slot,
		}).Debug("Duplicate payload attestation message received")

		return true, nil
	}

	return false, nil
}

func (e *EventsPayloadAttestation) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsPayloadAttestationData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsPayloadAttestationData{}

	if e.event == nil || e.event.Data == nil {
		return extra, fmt.Errorf("payload attestation message missing data")
	}

	slotNumber := uint64(e.event.Data.Slot)

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
