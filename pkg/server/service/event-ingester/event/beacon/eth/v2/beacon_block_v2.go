package v2

import (
	"context"
	"errors"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockV2Type = "BEACON_API_ETH_V2_BEACON_BLOCK_V2"
)

type BeaconBlockV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent

	// cache to store non-finalized block hashes to only forward them on once.
	// Typically non finalized blocks are ingested multiple times from the sentry.
	// Finalized blocks are only ingested once from the cannon.
	nonFinalizedCache store.Cache
}

func NewBeaconBlockV2(log logrus.FieldLogger, event *xatu.DecoratedEvent, cache store.Cache) *BeaconBlockV2 {
	return &BeaconBlockV2{
		log:               log.WithField("event", BeaconBlockV2Type),
		event:             event,
		nonFinalizedCache: cache,
	}
}

func (b *BeaconBlockV2) Type() string {
	return BeaconBlockV2Type
}

func (b *BeaconBlockV2) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockV2) Filter(ctx context.Context) bool {
	data, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockV2)
	if !ok {
		b.log.Error("failed to cast event data")

		return true
	}

	additionalData, ok := b.event.Meta.Client.AdditionalData.(*xatu.ClientMeta_EthV2BeaconBlockV2)
	if !ok {
		b.log.Error("failed to cast client additional data")

		return true
	}

	version := additionalData.EthV2BeaconBlockV2.GetVersion()
	if version == "" {
		b.log.Error("failed to get version")

		return true
	}

	var hash string

	switch version {
	case spec.DataVersionPhase0.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_Phase0Block).Phase0Block.StateRoot
	case spec.DataVersionAltair.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_AltairBlock).AltairBlock.StateRoot
	case spec.DataVersionBellatrix.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_BellatrixBlock).BellatrixBlock.StateRoot
	case spec.DataVersionCapella.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_CapellaBlock).CapellaBlock.StateRoot
	case spec.DataVersionDeneb.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_DenebBlock).DenebBlock.StateRoot
	case spec.DataVersionElectra.String():
		hash = data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_ElectraBlock).ElectraBlock.StateRoot
	case spec.DataVersionFulu.String():
		fuluBlock, ok := data.EthV2BeaconBlockV2.Message.(*v2.EventBlockV2_FuluBlock)
		if !ok {
			b.log.Error("failed to cast message to FuluBlock")

			return true
		}

		hash = fuluBlock.FuluBlock.StateRoot
	default:
		b.log.Error(fmt.Errorf("unknown version: %s", version))

		return true
	}

	if hash == "" {
		b.log.Error("failed to get hash")

		return true
	}

	return false
}

func (b *BeaconBlockV2) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
