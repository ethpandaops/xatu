package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// MempoolTransaction is an event that represents a transaction in the mempool.
type MempoolTransaction struct {
	log            logrus.FieldLogger
	now            time.Time
	tx             *types.Transaction
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
	id             uuid.UUID
	signer         types.Signer
}

// NewMempoolTransaction creates a new MempoolTransaction.
func NewMempoolTransaction(
	log logrus.FieldLogger,
	tx *types.Transaction,
	now time.Time,
	duplicateCache *ttlcache.Cache[string, time.Time],
	clientMeta *xatu.ClientMeta,
	signer types.Signer,
) *MempoolTransaction {
	return &MempoolTransaction{
		log:            log.WithField("event", "MEMPOOL_TRANSACTION_V2"),
		now:            now,
		tx:             tx,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
		signer:         signer,
		id:             uuid.New(),
	}
}

// Decorate decorates the transaction with additional data.
func (e *MempoolTransaction) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	rawTx, err := e.tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_V2,
			DateTime: timestamppb.New(e.now),
			Id:       e.id.String(),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_MempoolTransactionV2{
			MempoolTransactionV2: hexutil.Encode(rawTx),
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra transaction data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_MempoolTransactionV2{
			MempoolTransactionV2: additionalData,
		}
	}

	return decoratedEvent, nil
}

// ShouldIgnore checks if the transaction should be ignored based on the cache.
func (e *MempoolTransaction) ShouldIgnore(ctx context.Context) (bool, error) {
	hash, err := e.tx.Hash().MarshalText()
	if err != nil {
		return true, err
	}

	// Create a cache key that includes more transaction details to avoid false duplicates.
	// We're using hash + nonce + to_address to create a more specific identity for the transaction.
	var toAddr string
	if e.tx.To() != nil {
		toAddr = e.tx.To().String()
	}

	// Create a composite cache key. Format: hash:nonce:to_address.
	cacheKey := fmt.Sprintf("%s:%d:%s", string(hash), e.tx.Nonce(), toAddr)

	// Check if this specific transaction is already in the cache.
	existing := e.duplicateCache.Get(cacheKey)
	if existing != nil {
		e.log.WithFields(logrus.Fields{
			"hash":                  string(hash),
			"nonce":                 e.tx.Nonce(),
			"to_addr":               toAddr,
			"time_since_first_seen": time.Since(existing.Value()),
		}).Debug("Ignoring duplicate transaction")

		return true, nil
	}

	// Add this transaction to the cache with the default TTL.
	e.duplicateCache.Set(cacheKey, e.now, ttlcache.DefaultTTL)

	return false, nil
}

func (e *MempoolTransaction) getAdditionalData(ctx context.Context) (*xatu.ClientMeta_AdditionalMempoolTransactionV2Data, error) {
	var (
		to   string
		from string
	)

	if e.tx.To() != nil {
		to = e.tx.To().String()
	}

	if e.signer != nil {
		sender, err := e.signer.Sender(e.tx)
		if err == nil {
			from = sender.String()
		} else {
			e.log.WithError(err).Error("Failed to get transaction sender")
		}
	}

	extra := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Nonce:        wrapperspb.UInt64(e.tx.Nonce()),
		From:         from,
		To:           to,
		Gas:          wrapperspb.UInt64(e.tx.Gas()),
		GasPrice:     e.tx.GasPrice().String(),
		GasTipCap:    e.tx.GasTipCap().String(),
		GasFeeCap:    e.tx.GasFeeCap().String(),
		Value:        e.tx.Value().String(),
		Hash:         e.tx.Hash().String(),
		Size:         fmt.Sprintf("%d", e.tx.Size()),
		CallDataSize: fmt.Sprintf("%d", len(e.tx.Data())),
		Type:         wrapperspb.UInt32(uint32(e.tx.Type())),
	}

	if e.tx.Type() == 3 {
		var (
			hashes            = e.tx.BlobHashes()
			blobHashes        = make([]string, len(hashes))
			sidecarsEmptySize = 0
			sidecarsSize      = 0
			sidecars          = e.tx.BlobTxSidecar()
		)

		for i, hash := range hashes {
			blobHashes[i] = hash.String()
		}

		extra.BlobGas = wrapperspb.UInt64(e.tx.BlobGas())
		extra.BlobGasFeeCap = e.tx.BlobGasFeeCap().String()
		extra.BlobHashes = blobHashes

		if sidecars != nil {
			// When iterating, use the index to access the sidecar.
			// These aren't pointers, so we'll be copying a shit ton of data otherwise.
			for i := 0; i < len(sidecars.Blobs); i++ {
				sidecar := sidecars.Blobs[i][:]
				sidecarsSize += len(sidecar)
				sidecarsEmptySize += ethereum.CountConsecutiveEmptyBytes(sidecar, 4)
			}
		} else {
			return nil, errors.Errorf("no sidecars found for a type 3 transaction %s", e.tx.Hash().String())
		}

		extra.BlobSidecarsSize = fmt.Sprint(sidecarsSize)
		extra.BlobSidecarsEmptySize = fmt.Sprint(sidecarsEmptySize)
	}

	return extra, nil
}
