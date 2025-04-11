package sentry

import (
	"context"
	"fmt"
	"time"

	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/sirupsen/logrus"
)

// startMempoolTransactionWatcher initializes and starts the mempool transaction watcher if enabled.
func (s *Sentry) startMempoolTransactionWatcher(ctx context.Context) error {
	if s.Config.Execution == nil || !s.Config.Execution.Enabled {
		s.log.Info("Mempool transaction watcher disabled")

		return nil
	}

	// Validate execution config.
	if s.Config.Execution.Address == "" {
		return fmt.Errorf("execution.address is required when execution is enabled")
	}

	// Init the execution client with subscription configuration.
	execClient, err := execution.NewClient(
		ctx,
		s.log,
		&execution.Config{
			Address:         s.Config.Execution.Address,
			Headers:         s.Config.Execution.Headers,
			PollingInterval: time.Duration(s.Config.Execution.PollingInterval) * time.Second,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create execution client: %w", err)
	}

	s.execution = execClient

	// Register transaction handler.
	s.execution.OnNewPendingTransactions(s.createNewMempoolTransactionEvent)

	// Start the execution client.
	if err := s.execution.Start(ctx); err != nil {
		return fmt.Errorf("failed to start execution client: %w", err)
	}

	// Add shutdown function.
	s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
		return s.execution.Stop(ctx)
	})

	return nil
}

// createNewMempoolTransactionEvent creates a new mempool transaction event.
func (s *Sentry) createNewMempoolTransactionEvent(ctx context.Context, txHash string) error {
	// Get the full transaction.
	tx, err := s.execution.GetTransactionByHash(ctx, txHash)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to get transaction by hash")

		return err
	}

	// Build out the client meta for the event.
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to create client meta")

		return err
	}

	// Create the mempool transaction event.
	event := execEvent.NewMempoolTransaction(
		s.log,
		tx,
		time.Now().Add(s.clockDrift),
		s.duplicateCache.MempoolTransaction,
		meta,
		s.execution.GetSigner(),
	)

	// Check if we should ignore this transaction.
	ignore, err := event.ShouldIgnore(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to check if transaction should be ignored")

		return err
	}

	if ignore {
		s.log.WithField("tx_hash", txHash).Debug("Transaction ignored as duplicate")

		return nil
	}

	// Decorate the event.
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to decorate event")

		return err
	}

	// Handle the decorated event.
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"tx_hash": txHash,
		}).Error("Failed to handle decorated event")

		return err
	}

	s.summary.AddEventStreamEvents("mempool_transaction", 1)

	return nil
}
