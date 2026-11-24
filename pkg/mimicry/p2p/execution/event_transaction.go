package execution

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/internal/frametx"
	"github.com/ethpandaops/xatu/pkg/observability"
)

// mempoolTransaction is a transaction from a devp2p transaction list: tx, or frame with its
// typed envelope raw for an EIP-8141 frame transaction.
type mempoolTransaction struct {
	tx    *types.Transaction
	frame *frametx.Transaction
	raw   []byte
}

func (m mempoolTransaction) hash() common.Hash {
	if m.frame != nil {
		return m.frame.Hash
	}

	return m.tx.Hash()
}

// decodeTransactions decodes each transaction of a devp2p transaction list on its own, so a
// transaction that neither go-ethereum nor frametx decodes drops only itself, not the whole list.
func (p *Peer) decodeTransactions(ctx context.Context, list *rlp.RawList[*types.Transaction]) []mempoolTransaction {
	transactions := make([]mempoolTransaction, 0, list.Len())

	it := list.ContentIterator()
	for it.Next() {
		tx := new(types.Transaction)

		err := rlp.DecodeBytes(it.Value(), tx)
		if err == nil {
			transactions = append(transactions, mempoolTransaction{tx: tx})

			continue
		}

		// A typed transaction is an RLP string holding its EIP-2718 envelope.
		if _, envelope, _, splitErr := rlp.Split(it.Value()); splitErr == nil {
			frame, frameErr := frametx.Decode(envelope)
			if frameErr == nil {
				transactions = append(transactions, mempoolTransaction{frame: frame, raw: envelope})

				continue
			}
		}

		p.log.WithError(err).WithContext(ctx).Warn("Failed to decode transaction")
	}

	if err := it.Err(); err != nil {
		p.log.WithError(err).WithContext(ctx).Warn("Failed to decode transaction list")
	}

	return transactions
}

func (p *Peer) handleTransaction(ctx context.Context, eventTime time.Time, event mempoolTransaction) (*xatu.DecoratedEvent, error) {
	p.log.WithContext(ctx).Debug("Transaction received")

	meta, err := p.createNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if meta != nil {
		now = now.Add(time.Duration(meta.ClockDrift) * time.Millisecond)
	}

	tx := event.raw
	if event.frame == nil {
		tx, err = event.tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_V2,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: meta,
		},
		Data: &xatu.DecoratedEvent_MempoolTransactionV2{
			MempoolTransactionV2: fmt.Sprintf("0x%x", tx),
		},
	}

	var additionalData *xatu.ClientMeta_AdditionalMempoolTransactionV2Data
	if event.frame != nil {
		additionalData = frameTransactionData(event.frame, event.raw)
	} else {
		additionalData, err = p.getTransactionData(ctx, event.tx, meta, now)
	}

	if err != nil {
		p.log.WithError(err).WithContext(ctx).Error("Failed to get extra transaction data")
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
		p.log.WithError(err).WithContext(ctx).Error("failed to get sender")

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

		for i := range hashes {
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
				sidecar := sidecars.Blobs[i][:]
				sidecarsSize += len(sidecar)
				sidecarsEmptySize += ethereum.CountConsecutiveEmptyBytes(sidecar, 4)
			}
		} else {
			return nil, errors.Errorf("no sidecars found for a type 3 transaction %s", event.Hash().String())
		}

		extra.BlobSidecarsSize = fmt.Sprint(sidecarsSize)
		extra.BlobSidecarsEmptySize = fmt.Sprint(sidecarsEmptySize)
	}

	return extra, nil
}

// frameTransactionData describes an EIP-8141 frame transaction. Each frame has its own target, value
// and calldata, so to, value and input stay empty; max_fee is a total budget rather than a per-gas cap,
// so gas price and fee cap stay empty too.
func frameTransactionData(frame *frametx.Transaction, raw []byte) *xatu.ClientMeta_AdditionalMempoolTransactionV2Data {
	extra := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Nonce:     wrapperspb.UInt64(frame.NonceSeq),
		From:      frame.Sender.String(),
		GasTipCap: frame.Fees.MaxPriorityFeePerGas.String(),
		Hash:      frame.Hash.String(),
		// Like go-ethereum's Size for blob transactions, this includes any blob sidecar.
		Size: strconv.Itoa(len(raw)),
		Type: wrapperspb.UInt32(frametx.TxType),
	}

	if gas, err := frame.MaxGas(); err == nil {
		extra.Gas = wrapperspb.UInt64(gas)
	}

	if len(frame.BlobVersionedHashes) == 0 {
		return extra
	}

	extra.BlobGas = wrapperspb.UInt64(frame.BlobGas())

	for _, hash := range frame.BlobVersionedHashes {
		extra.BlobHashes = append(extra.BlobHashes, hash.String())
	}

	if frame.Blobs != nil {
		sidecarsSize, sidecarsEmptySize := 0, 0

		for _, blob := range frame.Blobs {
			sidecarsSize += len(blob)
			sidecarsEmptySize += ethereum.CountConsecutiveEmptyBytes(blob, 4)
		}

		extra.BlobSidecarsSize = strconv.Itoa(sidecarsSize)
		extra.BlobSidecarsEmptySize = strconv.Itoa(sidecarsEmptySize)
	}

	return extra
}

type TransactionHashItem struct {
	Hash common.Hash
	Seen time.Time
	Type uint8
}

type TransactionExporter struct {
	log observability.ContextualLogger

	handler func(ctx context.Context, items []*TransactionHashItem) error
}

func NewTransactionExporter(log observability.ContextualLogger, handler func(ctx context.Context, items []*TransactionHashItem) error) (TransactionExporter, error) {
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

type txHashMeta struct {
	Seen time.Time
	Type uint8
}

func (p *Peer) ExportTransactions(ctx context.Context, items []*TransactionHashItem) error {
	if len(items) == 0 {
		return nil
	}

	var (
		blobHashes  []common.Hash
		otherHashes []common.Hash
	)

	metaMap := make(map[common.Hash]txHashMeta, len(items))

	for _, item := range items {
		if item == nil {
			continue
		}

		exists := p.sharedCache.Transaction.Get(item.Hash.String())
		if exists == nil {
			metaMap[item.Hash] = txHashMeta{Seen: item.Seen, Type: item.Type}

			if item.Type == 3 {
				blobHashes = append(blobHashes, item.Hash)
			} else {
				otherHashes = append(otherHashes, item.Hash)
			}
		}
	}

	if len(blobHashes) == 0 && len(otherHashes) == 0 {
		return nil
	}

	// Process blob transactions in configured batch size (default 1, they're ~128KB+ each)
	for i := 0; i < len(blobHashes); i += p.blobTransactionBatchSize {
		end := min(i+p.blobTransactionBatchSize, len(blobHashes))

		p.fetchAndProcessTransactions(ctx, blobHashes[i:end], metaMap)
	}

	// Process other transactions in configured batch size (default 10)
	for i := 0; i < len(otherHashes); i += p.transactionBatchSize {
		end := min(i+p.transactionBatchSize, len(otherHashes))

		p.fetchAndProcessTransactions(ctx, otherHashes[i:end], metaMap)
	}

	return nil
}

func (p *Peer) fetchAndProcessTransactions(ctx context.Context, hashes []common.Hash, metaMap map[common.Hash]txHashMeta) {
	if len(hashes) == 0 {
		return
	}

	txs, err := p.client.GetPooledTransactions(ctx, hashes)
	if err != nil {
		p.log.WithError(err).WithContext(ctx).Warn("Failed to get pooled transactions")

		return
	}

	if txs == nil {
		return
	}

	for _, tx := range p.decodeTransactions(ctx, &(*eth.PooledTransactionsPacket)(txs).List) {
		_, retrieved := p.sharedCache.Transaction.GetOrSet(tx.hash().String(), true, ttlcache.WithTTL[string, bool](1*time.Hour))
		// transaction was just set in shared cache, so we need to handle it
		if !retrieved {
			meta := metaMap[tx.hash()]
			seen := meta.Seen

			if seen.IsZero() {
				p.log.WithField("hash", tx.hash().String()).WithContext(ctx).Error("Failed to find seen time for transaction")

				seen = time.Now()
			}

			event, err := p.handleTransaction(ctx, seen, tx)
			if err != nil {
				p.log.WithError(err).WithContext(ctx).Error("Failed to handle transaction")
			}

			if event != nil {
				if err := p.handlers.DecoratedEvent(ctx, event); err != nil {
					p.log.WithError(err).WithContext(ctx).Error("Failed to handle transaction")
				}
			}
		}
	}
}
