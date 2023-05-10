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
	log       logrus.FieldLogger
	event     *xatu.DecoratedEvent
	networkID uint64
}

func NewTransaction(log logrus.FieldLogger, event *xatu.DecoratedEvent, networkID uint64) *Transaction {
	return &Transaction{
		log:       log.WithField("event", TransactionType),
		event:     event,
		networkID: networkID,
	}
}

func (b *Transaction) Type() string {
	return TransactionType
}

func (b *Transaction) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_MempoolTransaction)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Transaction) Filter(ctx context.Context) bool {
	networkID := b.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()

	return networkID != b.networkID
}
