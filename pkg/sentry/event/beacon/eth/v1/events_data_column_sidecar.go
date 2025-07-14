package event

import (
	"context"
	"fmt"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
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

type EventsDataColumnSidecar struct {
	log logrus.FieldLogger

	now time.Time

	event          *apiv1.DataColumnSidecarEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsDataColumnSidecar(log logrus.FieldLogger, event *apiv1.DataColumnSidecarEvent, now time.Time, beaconNode *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsDataColumnSidecar {
	return &EventsDataColumnSidecar{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR"),
		now:            now,
		event:          event,
		beacon:         beaconNode,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsDataColumnSidecar) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsDataColumnSidecar{
			EthV1EventsDataColumnSidecar: &xatuethv1.EventDataColumnSidecar{
				Slot:           &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
				Index:          &wrapperspb.UInt64Value{Value: e.event.Index},
				BlockRoot:      e.event.BlockRoot.String(),
				KzgCommitments: e.convertKZGCommitments(),
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra data column sidecar data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsDataColumnSidecar{
			EthV1EventsDataColumnSidecar: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsDataColumnSidecar) ShouldIgnore(ctx context.Context) (bool, error) {
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
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"slot":                  e.event.Slot,
			"index":                 e.event.Index,
		}).Debug("Duplicate data column sidecar event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsDataColumnSidecar) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsDataColumnSidecarData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsDataColumnSidecarData{}

	dataColumnSlot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.event.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.event.Slot))

	extra.Slot = &xatu.SlotV2{
		Number:        &wrapperspb.UInt64Value{Value: dataColumnSlot.Number()},
		StartDateTime: timestamppb.New(dataColumnSlot.TimeWindow().Start()),
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Propagation = &xatu.PropagationV2{
		SlotStartDiff: &wrapperspb.UInt64Value{
			//nolint:gosec // not concerned in reality
			Value: uint64(e.now.Sub(dataColumnSlot.TimeWindow().Start()).Milliseconds()),
		},
	}

	return extra, nil
}

func (e *EventsDataColumnSidecar) convertKZGCommitments() []string {
	commitments := make([]string, len(e.event.KZGCommitments))
	for i, commitment := range e.event.KZGCommitments {
		commitments[i] = fmt.Sprintf("%#x", commitment)
	}

	return commitments
}
