package sentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

// startExecutionStateSizeWatcher starts the execution state size watcher on the shared execution client.
func (s *Sentry) startExecutionStateSizeWatcher(ctx context.Context, client *execution.Client) (*runningWatcher, error) {
	callback := func(ctx context.Context, data *execution.DebugStateSizeResponse) error {
		return s.processStateSizeData(ctx, client, data)
	}

	watcher := execution.NewStateSizeWatcher(
		client,
		s.log,
		s.Config.Execution.StateSize,
		callback,
	)

	if err := watcher.Start(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to start state size watcher: %w", err), watcher.Stop())
	}

	s.log.WithField("trigger_mode", s.Config.Execution.StateSize.TriggerMode).WithContext(ctx).Info("Execution debug state size watcher started")

	return &runningWatcher{
		healthy:    watcher.Healthy,
		stop:       watcher.Stop,
		registered: func() { s.stateSizeWatcher.Store(watcher) },
	}, nil
}

// processStateSizeData processes state size data from the execution client.
func (s *Sentry) processStateSizeData(ctx context.Context, client *execution.Client, data *execution.DebugStateSizeResponse) error {
	now := time.Now().Add(s.clockDrift)

	// Get execution client metadata.
	execMetadata := client.GetClientMetadata()
	if !execMetadata.Initialized {
		s.log.WithContext(ctx).Debug("Skipping state size event - execution client metadata not yet initialized")

		return nil
	}

	// Create client metadata.
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).WithContext(ctx).Error("Failed to create client meta for state size event")

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
		s.log.WithError(err).WithContext(ctx).Error("Failed to check if state size event should be ignored")

		return err
	}

	if ignore {
		return nil
	}

	// Decorate the event.
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("state_root", data.StateRoot).WithContext(ctx).Error("Failed to decorate state size event")

		return err
	}

	// Handle the decorated event.
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"state_root":   data.StateRoot,
			"block_number": data.BlockNumber,
		}).WithContext(ctx).Error("Failed to handle decorated state size event")

		return err
	}

	s.summary.AddEventStreamEvents("execution_state_size", 1)

	s.log.WithFields(logrus.Fields{
		"state_root":   data.StateRoot,
		"block_number": data.BlockNumber,
	}).WithContext(ctx).Debug("Processed execution debug state size event")

	return nil
}

// onHeadEventForStateSize should be called when a new head event is received.
// This triggers state size polling if the watcher is in "head" mode.
func (s *Sentry) onHeadEventForStateSize(ctx context.Context) error {
	watcher := s.stateSizeWatcher.Load()
	if watcher == nil {
		return nil
	}

	return watcher.OnHeadEvent(ctx)
}
