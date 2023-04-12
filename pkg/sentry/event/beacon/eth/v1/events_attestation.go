package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventsAttestation struct {
	log logrus.FieldLogger

	now time.Time

	event          *phase0.Attestation
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
}

func NewEventsAttestation(log logrus.FieldLogger, event *phase0.Attestation, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsAttestation {
	return &EventsAttestation{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_ATTESTATION"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
	}
}

func (e *EventsAttestation) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			DateTime: timestamppb.New(e.now),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsAttestation{
			EthV1EventsAttestation: &xatuethv1.Attestation{
				AggregationBits: xatuethv1.BytesToString(e.event.AggregationBits),
				Data: &xatuethv1.AttestationData{
					Slot:            uint64(e.event.Data.Slot),
					Index:           uint64(e.event.Data.Index),
					BeaconBlockRoot: xatuethv1.RootAsString(e.event.Data.BeaconBlockRoot),
					Source: &xatuethv1.Checkpoint{
						Epoch: uint64(e.event.Data.Source.Epoch),
						Root:  xatuethv1.RootAsString(e.event.Data.Source.Root),
					},
					Target: &xatuethv1.Checkpoint{
						Epoch: uint64(e.event.Data.Target.Epoch),
						Root:  xatuethv1.RootAsString(e.event.Data.Target.Root),
					},
				},
				Signature: xatuethv1.TrimmedString(fmt.Sprintf("%#x", e.event.Signature)),
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra attestation data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsAttestation{
			EthV1EventsAttestation: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsAttestation) ShouldIgnore(ctx context.Context) (bool, error) {
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
			"slot":                  e.event.Data.Slot,
		}).Debug("Duplicate attestation event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsAttestation) getAdditionalData(ctx context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsAttestationData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsAttestationData{}

	attestionSlot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Data.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Data.Slot))

	extra.Slot = &xatu.Slot{
		Number:        attestionSlot.Number(),
		StartDateTime: timestamppb.New(attestionSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.Propagation{
		SlotStartDiff: uint64(e.now.Sub(attestionSlot.TimeWindow().Start()).Milliseconds()),
	}

	// Build out the target section
	targetEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.event.Data.Target.Epoch))
	extra.Target = &xatu.ClientMeta_AdditionalEthV1AttestationTargetData{
		Epoch: &xatu.Epoch{
			Number:        targetEpoch.Number(),
			StartDateTime: timestamppb.New(targetEpoch.TimeWindow().Start()),
		},
	}

	// Build out the source section
	sourceEpoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.event.Data.Source.Epoch))
	extra.Source = &xatu.ClientMeta_AdditionalEthV1AttestationSourceData{
		Epoch: &xatu.Epoch{
			Number:        sourceEpoch.Number(),
			StartDateTime: timestamppb.New(sourceEpoch.TimeWindow().Start()),
		},
	}

	return extra, nil
}
