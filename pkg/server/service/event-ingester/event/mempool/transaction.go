package mempool

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	TransactionType = "MEMPOOL_TRANSACTION"
)

type Transaction struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTransaction(log logrus.FieldLogger, event *xatu.DecoratedEvent) *Transaction {
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
