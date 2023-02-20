package execution

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (p *Peer) handleTransaction(ctx context.Context, eventTime time.Time, event *types.Transaction) (*xatu.DecoratedEvent, error) {
	p.log.Debug("Transaction received")

	hash, err := hashstructure.Hash(event.Hash().String(), hashstructure.FormatV2, nil)
	if err != nil {
		return nil, err
	}

	item, retrieved := p.duplicateCache.Transaction.GetOrSet(fmt.Sprint(hash), time.Now(), ttlcache.DefaultTTL)
	if retrieved {
		p.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"transaction_hash":      event.Hash().String(),
		}).Debug("Duplicate transaction event received")
		// TODO(savid): add metrics
		return nil, errors.New("duplicate transaction event received")
	}

	meta, err := p.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if meta != nil {
		now = now.Add(time.Duration(meta.ClockDrift) * time.Millisecond)
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION,
			DateTime: timestamppb.New(now),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_MempoolTransaction{
			MempoolTransaction: string(event.Data()),
		},
	}

	additionalData, err := p.getTransactionData(ctx, event, meta, now)
	if err != nil {
		p.log.WithError(err).Error("Failed to get extra transaction data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_MempoolTransaction{
			MempoolTransaction: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (p *Peer) getTransactionData(ctx context.Context, event *types.Transaction, meta *xatu.ClientMeta, eventTime time.Time) (*xatu.ClientMeta_AdditionalMempoolTransactionData, error) {
	var to string
	if event.To() != nil {
		to = event.To().String()
	}

	from, err := p.signer.Sender(event)
	if err != nil {
		p.log.WithError(err).Error("failed to get sender")

		return nil, err
	}

	extra := &xatu.ClientMeta_AdditionalMempoolTransactionData{
		Nonce:        event.Nonce(),
		GasPrice:     event.GasPrice().String(),
		From:         from.String(),
		To:           to,
		Gas:          event.Gas(),
		Value:        event.Value().String(),
		Hash:         event.Hash().String(),
		Size:         strconv.FormatFloat(float64(event.Size()), 'f', 0, 64),
		CallDataSize: fmt.Sprintf("%d", len(event.Data())),
	}

	return extra, nil
}

type TransactionHashItem struct {
	Hash common.Hash
	Seen time.Time
}

type TransactionExporter struct {
	log logrus.FieldLogger

	handler func(ctx context.Context, items []*TransactionHashItem) error
}

func NewTransactionExporter(log logrus.FieldLogger, handler func(ctx context.Context, items []*TransactionHashItem) error) (TransactionExporter, error) {
	return TransactionExporter{
		log:     log,
		handler: handler,
	}, nil
}

func (t TransactionExporter) ExportItems(ctx context.Context, items []*TransactionHashItem) error {
	return t.handler(ctx, items)
}

func (t TransactionExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (p *Peer) ExportTransactions(ctx context.Context, items []*TransactionHashItem) error {
	go func() {
		hashes := make([]common.Hash, len(items))
		seenMap := map[common.Hash]time.Time{}

		for i, item := range items {
			exists := p.sharedCache.Transaction.Get(item.Hash.String())
			if exists == nil {
				hashes[i] = item.Hash
				seenMap[item.Hash] = item.Seen
			}
		}

		txs, err := p.client.GetPooledTransactions(ctx, hashes)
		if err != nil {
			p.log.WithError(err).Error("Failed to get pooled transactions")

			return
		}

		if txs != nil {
			for _, tx := range txs.PooledTransactionsPacket {
				exists := p.sharedCache.Transaction.Get(tx.Hash().String())
				if exists == nil {
					p.sharedCache.Transaction.Set(tx.Hash().String(), tx, ttlcache.DefaultTTL)

					seen := seenMap[tx.Hash()]
					if seen.IsZero() {
						p.log.WithField("hash", tx.Hash().String()).Error("Failed to find seen time for transaction")

						seen = time.Now()
					}

					event, err := p.handleTransaction(ctx, seen, tx)

					if err != nil {
						p.log.WithError(err).Error("Failed to handle transaction")
					}

					if event != nil {
						if err := p.handlers.DecoratedEvent(ctx, event); err != nil {
							p.log.WithError(err).Error("Failed to handle transaction")
						}
					}
				}
			}
		}
	}()

	return nil
}
