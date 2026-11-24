package mempool

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TransactionRemovedType = "MEMPOOL_TRANSACTION_REMOVED"
)

type TransactionRemoved struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTransactionRemoved(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TransactionRemoved {
	return &TransactionRemoved{
		log:   log.WithField("event", TransactionRemovedType),
		event: event,
	}
}

func (b *TransactionRemoved) Type() string {
	return TransactionRemovedType
}

func (b *TransactionRemoved) Validate(ctx context.Context) error {
	if b.event.GetMempoolTransactionRemoved() == nil {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *TransactionRemoved) Filter(ctx context.Context) bool {
	return false
}

func (b *TransactionRemoved) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
