package event

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ProposerDuty struct {
	log logrus.FieldLogger

	now time.Time

	duty  *v1.ProposerDuty
	epoch phase0.Epoch

	beacon         *ethereum.BeaconNode
	clientMeta     *xatu.ClientMeta
	duplicateCache *ttlcache.Cache[string, time.Time]
	id             uuid.UUID
}

func NewProposerDuty(log logrus.FieldLogger, duty *v1.ProposerDuty, epoch phase0.Epoch, now time.Time, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta, duplicateCache *ttlcache.Cache[string, time.Time]) *ProposerDuty {
	return &ProposerDuty{
		log:            log.WithField("event", "BEACON_API_ETH_V1_PROPOSER_DUTY"),
		now:            now,
		duty:           duty,
		epoch:          epoch,
		duplicateCache: duplicateCache,
		beacon:         beacon,
		clientMeta:     clientMeta,
		id:             uuid.New(),
	}
}

func (e *ProposerDuty) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	// Make a clone of the metadata
	metadata, ok := proto.Clone(e.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, fmt.Errorf("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
			EthV1ProposerDuty: &xatuethv1.ProposerDuty{
				Slot:           wrapperspb.UInt64(uint64(e.duty.Slot)),
				Pubkey:         fmt.Sprintf("0x%s", hex.EncodeToString(e.duty.PubKey[:])),
				ValidatorIndex: wrapperspb.UInt64(uint64(e.duty.ValidatorIndex)),
			},
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra proposer duty data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1ProposerDuty{
			EthV1ProposerDuty: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *ProposerDuty) ShouldIgnore(ctx context.Context) (bool, error) {
	if err := e.beacon.Synced(ctx); err != nil {
		return true, err
	}

	key := fmt.Sprintf("%d-%d", e.epoch, e.duty.Slot)

	item, retrieved := e.duplicateCache.GetOrSet(key, e.now, ttlcache.WithTTL[string, time.Time](ttlcache.DefaultTTL))
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"epoch":                 e.epoch,
			"time_since_first_item": time.Since(item.Value()),
		}).Debug("Duplicate proposer duty event received")

		return true, nil
	}

	return false, nil
}

func (e *ProposerDuty) getAdditionalData(_ context.Context) (*xatu.ClientMeta_AdditionalEthV1ProposerDutyData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
		StateId: xatuethv1.StateIDHead,
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(e.duty.Slot))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(e.duty.Slot))

	extra.Slot = &xatu.SlotV2{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        &wrapperspb.UInt64Value{Value: uint64(e.duty.Slot)},
	}

	extra.Epoch = &xatu.EpochV2{
		Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	return extra, nil
}
