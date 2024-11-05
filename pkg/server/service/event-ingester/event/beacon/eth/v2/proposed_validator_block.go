package v2

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ProposedValidatorBlockType = "BEACON_API_ETH_V3_PROPOSED_VALIDATOR_BLOCK"
)

type ProposedValidatorBlock struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewProposedValidatorBlock(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ProposedValidatorBlock {
	return &ProposedValidatorBlock{
		log:   log.WithField("event", ProposedValidatorBlockType),
		event: event,
	}
}

func (b *ProposedValidatorBlock) Type() string {
	return ProposedValidatorBlockType
}

func (b *ProposedValidatorBlock) Validate(_ context.Context) error {
	if _, ok := b.event.Data.(*xatu.DecoratedEvent_EthV3ProposedValidatorBlock); !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *ProposedValidatorBlock) Filter(_ context.Context) bool {
	data, ok := b.event.Data.(*xatu.DecoratedEvent_EthV3ProposedValidatorBlock)
	if !ok {
		b.log.Error("failed to cast event data")

		return true
	}

	additionalData, ok := b.event.Meta.Client.AdditionalData.(*xatu.ClientMeta_EthV3ProposedValidatorBlock)
	if !ok {
		b.log.Error("failed to cast client additional data")

		return true
	}

	version := additionalData.EthV3ProposedValidatorBlock.GetVersion()
	if version == "" {
		b.log.Error("failed to get version")

		return true
	}

	var hash string

	switch version {
	case "phase0":
		hash = data.EthV3ProposedValidatorBlock.Message.(*v2.EventBlockV2_Phase0Block).Phase0Block.StateRoot
	case "altair":
		hash = data.EthV3ProposedValidatorBlock.Message.(*v2.EventBlockV2_AltairBlock).AltairBlock.StateRoot
	case "bellatrix":
		hash = data.EthV3ProposedValidatorBlock.Message.(*v2.EventBlockV2_BellatrixBlock).BellatrixBlock.StateRoot
	case "capella":
		hash = data.EthV3ProposedValidatorBlock.Message.(*v2.EventBlockV2_CapellaBlock).CapellaBlock.StateRoot
	case "deneb":
		hash = data.EthV3ProposedValidatorBlock.Message.(*v2.EventBlockV2_DenebBlock).DenebBlock.StateRoot
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

func (b *ProposedValidatorBlock) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
