package v2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockType = "BEACON_API_ETH_V2_BEACON_BLOCK"
)

type BeaconBlock struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent

	cache store.Cache
}

func NewBeaconBlock(log logrus.FieldLogger, event *xatu.DecoratedEvent, cache store.Cache) *BeaconBlock {
	return &BeaconBlock{
		log:   log.WithField("event", BeaconBlockType),
		event: event,
		cache: cache,
	}
}

func (b *BeaconBlock) Type() string {
	return BeaconBlockType
}

func (b *BeaconBlock) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlock)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlock) Filter(ctx context.Context) bool {
	data, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlock)
	if !ok {
		b.log.Error("failed to cast event data")

		return true
	}

	additionalData, ok := b.event.Meta.Client.AdditionalData.(*xatu.ClientMeta_EthV2BeaconBlock)
	if !ok {
		b.log.Error("failed to cast client additional data")

		return true
	}

	version := additionalData.EthV2BeaconBlock.GetVersion()
	if version == "" {
		b.log.Error("failed to get version")

		return true
	}

	var hash string

	switch version {
	case spec.DataVersionPhase0.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_Phase0Block).Phase0Block.StateRoot
	case spec.DataVersionAltair.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_AltairBlock).AltairBlock.StateRoot
	case spec.DataVersionBellatrix.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_BellatrixBlock).BellatrixBlock.StateRoot
	case spec.DataVersionCapella.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_CapellaBlock).CapellaBlock.StateRoot
	case spec.DataVersionDeneb.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_DenebBlock).DenebBlock.StateRoot
	case spec.DataVersionElectra.String():
		//nolint:staticcheck // Handled by v2
		hash = data.EthV2BeaconBlock.Message.(*v2.EventBlock_ElectraBlock).ElectraBlock.StateRoot
	default:
		b.log.Error(fmt.Errorf("unknown version: %s", version))

		return true
	}

	if hash == "" {
		b.log.Error("failed to get hash")

		return true
	}

	key := "beacon_block" + ":" + hash

	_, retrieved, err := b.cache.GetOrSet(ctx, key, version, time.Minute*30)
	if err != nil {
		b.log.WithError(err).Error("failed to retrieve from cache")

		return true
	}

	// If the block is already in the cache, filter it out
	return retrieved
}

func (b *BeaconBlock) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
