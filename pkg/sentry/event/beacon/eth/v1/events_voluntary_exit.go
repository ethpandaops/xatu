package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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

type EventsVoluntaryExit struct {
	log logrus.FieldLogger

	now time.Time

	event          *phase0.SignedVoluntaryExit
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsVoluntaryExit(log logrus.FieldLogger, event *phase0.SignedVoluntaryExit, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsVoluntaryExit {
	return &EventsVoluntaryExit{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsVoluntaryExit) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsVoluntaryExitV2{
			EthV1EventsVoluntaryExitV2: &xatuethv1.EventVoluntaryExitV2{
				Signature: e.event.Signature.String(),
				Message: &xatuethv1.EventVoluntaryExitMessageV2{
					Epoch:          &wrapperspb.UInt64Value{Value: uint64(e.event.Message.Epoch)},
					ValidatorIndex: &wrapperspb.UInt64Value{Value: uint64(e.event.Message.ValidatorIndex)},
				},
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra voluntary exit data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsVoluntaryExitV2{
			EthV1EventsVoluntaryExitV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsVoluntaryExit) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		//nolint:nilerr // Returning nil is intentional.
		return true, nil
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
			"epoch":                 e.event.Message.Epoch,
		}).Debug("Duplicate voluntary exit event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsVoluntaryExit) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsVoluntaryExitV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsVoluntaryExitV2Data{}

	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Message.Epoch))

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
