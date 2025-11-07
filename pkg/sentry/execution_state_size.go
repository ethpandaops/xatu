package sentry

import (
	"context"
	"fmt"
	"time"

	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/sirupsen/logrus"
)

// stateSizeWatcher holds the state size watcher instance.
var stateSizeWatcher *execution.StateSizeWatcher

// startExecutionStateSizeWatcher initializes and starts the execution state size watcher.
func (s *Sentry) startExecutionStateSizeWatcher(ctx context.Context) error {
	// Check if execution is enabled and state size monitoring is configured.
	if s.Config.Execution == nil || !s.Config.Execution.Enabled {
		s.log.Info("Execution debug state size watcher disabled (execution not enabled)")

		return nil
	}

	if s.Config.Execution.StateSize == nil || !s.Config.Execution.StateSize.Enabled {
		s.log.Info("Execution debug state size watcher disabled")

		return nil
	}

	// Ensure execution client is initialized.
	// If mempool watcher was started, the client should already exist.
	if s.execution == nil {
		// Initialize the execution client.
		client, err := execution.NewClient(
			ctx,
			s.log,
			s.Config.Execution,
		)
		if err != nil {
			return fmt.Errorf("failed to create execution client: %w", err)
		}

		// Start the execution client.
		if err := client.Start(ctx); err != nil {
			return fmt.Errorf("failed to start execution client: %w", err)
		}

		s.execution = client

		// Add shutdown function for the client if it wasn't added already.
		s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
			return client.Stop(ctx)
		})
	}

	// Create state size callback.
	callback := func(ctx context.Context, data *execution.DebugStateSizeResponse) error {
		return s.processStateSizeData(ctx, data)
	}

	// Create and start the state size watcher.
	stateSizeWatcher = execution.NewStateSizeWatcher(
		s.execution,
		s.log,
		s.Config.Execution.StateSize,
		callback,
	)

	if err := stateSizeWatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start state size watcher: %w", err)
	}

	// Add shutdown function for the watcher.
	s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
		return stateSizeWatcher.Stop()
	})

	s.log.WithField("trigger_mode", s.Config.Execution.StateSize.TriggerMode).Info("Execution debug state size watcher started")

	return nil
}

// processStateSizeData processes state size data from the execution client.
func (s *Sentry) processStateSizeData(ctx context.Context, data *execution.DebugStateSizeResponse) error {
	now := time.Now().Add(s.clockDrift)

	// Get execution client metadata.
	execMetadata := s.execution.GetClientMetadata()
	if !execMetadata.Initialized {
		s.log.Debug("Skipping state size event - execution client metadata not yet initialized")

		return nil
	}

	// Create client metadata.
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).Error("Failed to create client meta for state size event")

		return err
	}

	// Add execution client metadata.
	if meta.Ethereum != nil && meta.Ethereum.Execution != nil {
		meta.Ethereum.Execution.Implementation = execMetadata.Implementation
		meta.Ethereum.Execution.Version = execMetadata.Version
		meta.Ethereum.Execution.VersionMajor = execMetadata.VersionMajor
		meta.Ethereum.Execution.VersionMinor = execMetadata.VersionMinor
		meta.Ethereum.Execution.VersionPatch = execMetadata.VersionPatch
	}

	// Create the execution debug state size event.
	event := execEvent.NewExecutionStateSize(
		s.log,
		data,
		now,
		s.duplicateCache.ExecutionStateSize,
		meta,
	)

	// Check if we should ignore this event (duplicate detection).
	ignore, err := event.ShouldIgnore(ctx)
	if err != nil {
		s.log.WithError(err).Error("Failed to check if state size event should be ignored")

		return err
	}

	if ignore {
		return nil
	}

	// Decorate the event.
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("state_root", data.StateRoot).Error("Failed to decorate state size event")

		return err
	}

	// Handle the decorated event.
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"state_root":   data.StateRoot,
			"block_number": data.BlockNumber,
		}).Error("Failed to handle decorated state size event")

		return err
	}

	s.summary.AddEventStreamEvents("execution_state_size", 1)

	s.log.WithFields(logrus.Fields{
		"state_root":   data.StateRoot,
		"block_number": data.BlockNumber,
	}).Debug("Processed execution debug state size event")

	return nil
}

// onHeadEventForStateSize should be called when a new head event is received.
// This triggers state size polling if the watcher is in "head" mode.
func (s *Sentry) onHeadEventForStateSize(ctx context.Context) error {
	if stateSizeWatcher == nil {
		return nil
	}

	return stateSizeWatcher.OnHeadEvent(ctx)
}
