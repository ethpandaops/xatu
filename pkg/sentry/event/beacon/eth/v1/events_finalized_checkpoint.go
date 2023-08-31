package event

import (
	"context"
	"fmt"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
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

type EventsFinalizedCheckpoint struct {
	log logrus.FieldLogger

	now time.Time

	event          *eth2v1.FinalizedCheckpointEvent
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
}

func NewEventsFinalizedCheckpoint(log logrus.FieldLogger, event *eth2v1.FinalizedCheckpointEvent, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *EventsFinalizedCheckpoint {
	return &EventsFinalizedCheckpoint{
		log:            log.WithField("event", "BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *EventsFinalizedCheckpoint) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1EventsFinalizedCheckpointV2{
			EthV1EventsFinalizedCheckpointV2: &xatuethv1.EventFinalizedCheckpointV2{
				Epoch: &wrapperspb.UInt64Value{Value: uint64(e.event.Epoch)},
				State: xatuethv1.RootAsString(e.event.State),
				Block: xatuethv1.RootAsString(e.event.Block),
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra finalized checkpoint data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1EventsFinalizedCheckpointV2{
			EthV1EventsFinalizedCheckpointV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *EventsFinalizedCheckpoint) ShouldIgnore(ctx context.Context) (bool, error) {
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
			"epoch":                 e.event.Epoch,
		}).Debug("Duplicate finalized checkpoint event received")

		return true, nil
	}

	return false, nil
}

func (e *EventsFinalizedCheckpoint) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1EventsFinalizedCheckpointV2Data, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1EventsFinalizedCheckpointV2Data{}

	epoch := e.beacon.Metadata().Wallclock().Epochs().FromNumber(uint64(e.event.Epoch))

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
