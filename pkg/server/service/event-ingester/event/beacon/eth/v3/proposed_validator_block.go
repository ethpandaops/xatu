package v3

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorBlockType = "BEACON_API_ETH_V3_VALIDATOR_BLOCK"
)

type ValidatorBlock struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewValidatorBlock(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ValidatorBlock {
	return &ValidatorBlock{
		log:   log.WithField("event", ValidatorBlockType),
		event: event,
	}
}

func (b *ValidatorBlock) Type() string {
	return ValidatorBlockType
}

func (b *ValidatorBlock) Validate(_ context.Context) error {
	if _, ok := b.event.Data.(*xatu.DecoratedEvent_EthV3ValidatorBlock); !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *ValidatorBlock) Filter(_ context.Context) bool {
	data, ok := b.event.Data.(*xatu.DecoratedEvent_EthV3ValidatorBlock)
	if !ok {
		b.log.Error("failed to cast event data")

		return true
	}

	additionalData, ok := b.event.Meta.Client.AdditionalData.(*xatu.ClientMeta_EthV3ValidatorBlock)
	if !ok {
		b.log.Error("failed to cast client additional data")

		return true
	}

	version := additionalData.EthV3ValidatorBlock.GetVersion()
	if version == "" {
		b.log.Error("failed to get version")

		return true
	}

	var hash string

	switch version {
	case "phase0":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_Phase0Block).Phase0Block.GetStateRoot()
	case "altair":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_AltairBlock).AltairBlock.GetStateRoot()
	case "bellatrix":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_BellatrixBlock).BellatrixBlock.GetStateRoot()
	case "capella":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_CapellaBlock).CapellaBlock.GetStateRoot()
	case "deneb":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_DenebBlock).DenebBlock.GetStateRoot()
	case "electra":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_ElectraBlock).ElectraBlock.GetStateRoot()
	case "fulu":
		hash = data.EthV3ValidatorBlock.GetMessage().(*v2.EventBlockV2_FuluBlock).FuluBlock.GetStateRoot()
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

func (b *ValidatorBlock) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
