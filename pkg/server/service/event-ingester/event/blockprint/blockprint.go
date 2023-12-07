package blockprint

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BlockClassificationType = "BLOCKPRINT_BLOCK_CLASSIFICATION"
)

type BlockClassification struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBlockClassification(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BlockClassification {
	return &BlockClassification{
		log:   log.WithField("event", BlockClassificationType),
		event: event,
	}
}

func (b *BlockClassification) Type() string {
	return BlockClassificationType
}

func (b *BlockClassification) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_BlockprintBlockClassification)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BlockClassification) Filter(ctx context.Context) bool {
	return false
}

func (b *BlockClassification) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
