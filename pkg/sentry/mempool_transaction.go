package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

// startMempoolTransactionWatcher starts the mempool transaction watcher on the shared execution client.
func (s *Sentry) startMempoolTransactionWatcher(ctx context.Context, client *execution.Client, metrics *execution.Metrics) (*runningWatcher, error) {
	processTxCallback := func(ctx context.Context, record *execution.PendingTxRecord, txData json.RawMessage) error {
		return s.processMempoolTransaction(ctx, client.GetSigner(), record, txData)
	}

	watcher := execution.NewMempoolWatcher(
		client,
		s.log,
		s.Config.Execution,
		processTxCallback,
		metrics,
	)

	switch {
	case s.Config.Execution.RemovedTxsEnabled && s.Config.Execution.TxPoolContentEnabled:
		watcher.OnRemovedTransactions(s.processRemovedMempoolTransactions)
	case s.Config.Execution.RemovedTxsEnabled:
		s.log.WithContext(ctx).Warn("removedTxsEnabled needs txPoolContentEnabled, not reporting removed transactions")
	}

	if err := watcher.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start mempool watcher: %w", err)
	}

	return &runningWatcher{
		healthy: watcher.Healthy,
		stop: func() error {
			watcher.Stop()

			return nil
		},
	}, nil
}

// processMempoolTransaction is called when a full transaction is found in txpool_content.
func (s *Sentry) processMempoolTransaction(ctx context.Context, signer types.Signer, record *execution.PendingTxRecord, txData json.RawMessage) error {
	// Get the hash from the record.
	txHash := common.HexToHash(record.Hash)

	// First unmarshal into a map to access the hash field and essential metadata.
	var txMap map[string]any
	if err := json.Unmarshal(txData, &txMap); err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to unmarshal transaction data")

		return err
	}

	// Extract the from address if available.
	var fromAddress string
	if from, ok := txMap["from"].(string); ok {
		fromAddress = from
	}

	// Build out the client meta for the event.
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to create client meta")

		return err
	}

	// Parse the transaction.
	tx, err := parseRawTransactionFromTxPool(txData, txHash)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to parse transaction data")

		return err
	}

	// Pre-populate additional data with the from address from txpool_content.
	// This will be used if signature recovery fails in the MempoolTransaction.Decorate method.
	additionalData := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Hash: record.Hash,
		From: fromAddress,
	}

	meta.AdditionalData = &xatu.ClientMeta_MempoolTransactionV2{
		MempoolTransactionV2: additionalData,
	}

	// Create the mempool transaction event using the ORIGINAL timestamp when the tx was first seen.
	event := execEvent.NewMempoolTransaction(
		s.log,
		tx,
		record.FirstSeen.Add(s.clockDrift),
		s.duplicateCache.MempoolTransaction,
		meta,
		signer,
	)

	// Decorate the event.
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to decorate event")

		return err
	}

	// Handle the decorated event.
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"tx_hash": record.Hash,
		}).WithContext(ctx).Error("Failed to handle decorated event")

		return err
	}

	// Note: No need to add metrics here anymore as the MempoolWatcher keeps track of metrics directly.
	s.summary.AddEventStreamEvents("mempool_transaction", 1)

	return nil
}

// processRemovedMempoolTransactions emits a MEMPOOL_TRANSACTION_REMOVED event for each transaction that left the
// txpool without being included.
func (s *Sentry) processRemovedMempoolTransactions(ctx context.Context, removed []*execution.RemovedTx) {
	for _, tx := range removed {
		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).WithContext(ctx).Error("Failed to create client meta")

			return
		}

		if err := s.handleNewDecoratedEvent(ctx, newMempoolTransactionRemovedEvent(tx, meta, s.clockDrift)); err != nil {
			s.log.WithError(err).WithField("tx_hash", tx.Hash).WithContext(ctx).Error("Failed to handle decorated event")

			continue
		}

		s.summary.AddEventStreamEvents("mempool_transaction_removed", 1)
	}
}

func newMempoolTransactionRemovedEvent(tx *execution.RemovedTx, meta *xatu.ClientMeta, clockDrift time.Duration) *xatu.DecoratedEvent {
	data := &xatu.MempoolTransactionRemoved{
		Hash:      tx.Hash,
		From:      tx.From.Hex(),
		Nonce:     wrapperspb.UInt64(tx.Nonce),
		Type:      wrapperspb.UInt32(uint32(tx.Type)),
		Gas:       wrapperspb.UInt64(tx.Gas),
		GasTipCap: tx.GasTipCap.String(),
		GasFeeCap: tx.GasFeeCap.String(),
		Reason:    tx.Reason,
		FirstSeen: timestamppb.New(tx.FirstSeen.Add(clockDrift)),
		LastSeen:  timestamppb.New(tx.LastSeen.Add(clockDrift)),
		Pool:      tx.Pool,
	}

	if tx.To != nil {
		data.To = strings.ToLower(tx.To.Hex())
	}

	if tx.BlobGasFeeCap != nil {
		data.BlobGasFeeCap = tx.BlobGasFeeCap.String()
	}

	if tx.Replacement != nil {
		data.ReplacementHash = tx.Replacement.Hash
		data.ReplacementGasTipCap = tx.Replacement.GasTipCap.String()
		data.ReplacementGasFeeCap = tx.Replacement.GasFeeCap.String()
	}

	if tx.AccountNonce != nil {
		data.AccountNonce = wrapperspb.UInt64(*tx.AccountNonce)
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_REMOVED,
			DateTime: timestamppb.New(tx.DetectedAt.Add(clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: meta},
		Data: &xatu.DecoratedEvent_MempoolTransactionRemoved{MempoolTransactionRemoved: data},
	}
}

// parseRawTransactionFromTxPool decodes the signed transaction from txpool_content or eth_getTransactionByHash JSON.
// It rejects a transaction whose decoded hash differs from the one the client reported, rather than record wrong
// fields under the real hash.
func parseRawTransactionFromTxPool(txData json.RawMessage, expectedHash common.Hash) (*types.Transaction, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalJSON(txData); err != nil {
		return nil, fmt.Errorf("decoding transaction: %w", err)
	}

	if tx.Hash() != expectedHash {
		return nil, fmt.Errorf("decoded transaction hash %s differs from the reported %s", tx.Hash(), expectedHash)
	}

	return tx, nil
}
