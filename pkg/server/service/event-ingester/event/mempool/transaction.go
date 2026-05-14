package mempool

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TransactionType = "MEMPOOL_TRANSACTION"
)

type Transaction struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTransaction(log observability.ContextualLogger, event *xatu.DecoratedEvent) *Transaction {
	return &Transaction{
		log:   log.WithField("event", TransactionType),
		event: event,
	}
}

func (b *Transaction) Type() string {
	return TransactionType
}

func (b *Transaction) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_MempoolTransaction)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Transaction) Filter(ctx context.Context) bool {
	return false
}

func (b *Transaction) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
