package execution

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
)

func (p *Peer) handleTransaction(ctx context.Context, eventTime time.Time, event *types.Transaction) (*xatu.DecoratedEvent, error) {
	p.log.Debug("Transaction received")

	meta, err := p.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if meta != nil {
		now = now.Add(time.Duration(meta.ClockDrift) * time.Millisecond)
	}

	tx, err := event.MarshalBinary()
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_V2,
			DateTime: timestamppb.New(now),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_MempoolTransactionV2{
			MempoolTransactionV2: fmt.Sprintf("0x%x", tx),
		},
	}

	additionalData, err := p.getTransactionData(ctx, event, meta, now)
	if err != nil {
		p.log.WithError(err).Error("Failed to get extra transaction data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_MempoolTransactionV2{
			MempoolTransactionV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (p *Peer) getTransactionData(ctx context.Context, event *types.Transaction, meta *xatu.ClientMeta, eventTime time.Time) (*xatu.ClientMeta_AdditionalMempoolTransactionV2Data, error) {
	var to string
	if event.To() != nil {
		to = event.To().String()
	}

	from, err := p.signer.Sender(event)
	if err != nil {
		p.log.WithError(err).Error("failed to get sender")

		return nil, err
	}

	extra := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Nonce:        wrapperspb.UInt64(event.Nonce()),
		From:         from.String(),
		To:           to,
		Gas:          wrapperspb.UInt64(event.Gas()),
		GasPrice:     event.GasPrice().String(),
		GasTipCap:    event.GasTipCap().String(),
		GasFeeCap:    event.GasFeeCap().String(),
		Value:        event.Value().String(),
		Hash:         event.Hash().String(),
		Size:         strconv.FormatFloat(float64(event.Size()), 'f', 0, 64),
		CallDataSize: fmt.Sprintf("%d", len(event.Data())),
		Type:         wrapperspb.UInt32(uint32(event.Type())),
	}

	if event.Type() == 3 {
		hashes := event.BlobHashes()
		blobHashes := make([]string, len(hashes))

		for i := 0; i < len(hashes); i++ {
			hash := hashes[i]
			blobHashes[i] = hash.String()
		}

		extra.BlobGas = wrapperspb.UInt64(event.BlobGas())
		extra.BlobGasFeeCap = event.BlobGasFeeCap().String()
		extra.BlobHashes = blobHashes
		sidecarsEmptySize := 0
		sidecarsSize := 0

		sidecars := event.BlobTxSidecar()

		if sidecars != nil {
			for i := 0; i < len(sidecars.Blobs); i++ {
				sidecars := sidecars.Blobs[i][:]
				sidecarsSize += len(sidecars)
				sidecarsEmptySize += ethereum.CountConsecutiveEmptyBytes(sidecars, 4)
			}
		} else {
			p.log.WithField("versioned hash", event.Hash().String()).WithField("transaction", event.Hash().String()).Warn("no sidecars found for a type 3 transaction")
		}

		extra.BlobSidecarsSize = fmt.Sprint(sidecarsSize)
		extra.BlobSidecarsEmptySize = fmt.Sprint(sidecarsEmptySize)
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
			p.log.WithError(err).Warn("Failed to get pooled transactions")

			return
		}

		if txs != nil {
			for _, tx := range txs.PooledTransactionsResponse {
				_, retrieved := p.sharedCache.Transaction.GetOrSet(tx.Hash().String(), true, ttlcache.WithTTL[string, bool](1*time.Hour))
				// transaction was just set in shared cache, so we need to handle it
				if !retrieved {
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
