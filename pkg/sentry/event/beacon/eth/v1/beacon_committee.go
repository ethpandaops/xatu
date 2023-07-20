package event

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type BeaconCommittee struct {
	log logrus.FieldLogger

	now time.Time

	event *v1.BeaconCommittee
	epoch phase0.Epoch

	beacon         *ethereum.BeaconNode
	clientMeta     *xatu.ClientMeta
	duplicateCache *ttlcache.Cache[string, time.Time]
	id             uuid.UUID
}

func NewBeaconCommittee(log logrus.FieldLogger, event *v1.BeaconCommittee, epoch phase0.Epoch, now time.Time, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta, duplicateCache *ttlcache.Cache[string, time.Time]) *BeaconCommittee {
	return &BeaconCommittee{
		log:            log.WithField("event", "BEACON_API_ETH_V1_BEACON_COMMITTEE"),
		now:            now,
		event:          event,
		epoch:          epoch,
		duplicateCache: duplicateCache,
		beacon:         beacon,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *BeaconCommittee) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	validators := make([]*wrapperspb.UInt64Value, len(e.event.Validators))
	for i := range e.event.Validators {
		validators[i] = &wrapperspb.UInt64Value{Value: uint64(e.event.Validators[i])}
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &xatuethv1.Committee{
				Slot:       &wrapperspb.UInt64Value{Value: uint64(e.event.Slot)},
				Index:      &wrapperspb.UInt64Value{Value: uint64(e.event.Index)},
				Validators: validators,
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon committee data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconCommitee{
			EthV1BeaconCommitee: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *BeaconCommittee) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	key := fmt.Sprintf("%d-%d-%d", e.epoch, e.event.Index, e.event.Slot)

	item, retrieved := e.duplicateCache.GetOrSet(key, e.now, ttlcache.DefaultTTL)
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"epoch":                 e.epoch,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate beacon committee event received")

		return true, nil
	}

	return false, nil
}

func (e *BeaconCommittee) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
		Epoch: uint64(e.epoch),
	}

	return extra, nil
}
