package mempool

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	TransactionV2Type = "MEMPOOL_TRANSACTION_V2"
)

type TransactionV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTransactionV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TransactionV2 {
	return &TransactionV2{
		log:   log.WithField("event", TransactionV2Type),
		event: event,
	}
}

func (b *TransactionV2) Type() string {
	return TransactionV2Type
}

func (b *TransactionV2) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_MempoolTransactionV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *TransactionV2) Filter(ctx context.Context) bool {
	return false
}
