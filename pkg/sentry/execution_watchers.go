package sentry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

const (
	executionWatcherMempool   = "mempool"
	executionWatcherStateSize = "state_size"

	executionWatcherInitialBackoff = 5 * time.Second
	executionWatcherMaxBackoff     = 5 * time.Minute
)

// startExecutionWatchers starts each enabled execution watcher in the background, retrying until it starts or the sentry stops.
func (s *Sentry) startExecutionWatchers(ctx context.Context) {
	if s.Config.Execution == nil || !s.Config.Execution.Enabled {
		s.log.WithContext(ctx).Info("Execution watchers disabled")

		return
	}

	networkName := string(s.beacon.Metadata().Network.Name)
	if networkName == "" {
		networkName = unknown
	}

	mempoolMetrics := execution.NewMetrics("execution", networkName)

	watchers := map[string]func(context.Context) error{
		executionWatcherMempool: func(ctx context.Context) error {
			return s.startMempoolTransactionWatcher(ctx, mempoolMetrics)
		},
	}

	if s.Config.Execution.StateSize != nil && s.Config.Execution.StateSize.Enabled {
		watchers[executionWatcherStateSize] = s.startExecutionStateSizeWatcher
	} else {
		s.log.WithContext(ctx).Info("Execution debug state size watcher disabled")
	}

	ctx, cancel := context.WithCancel(ctx)

	for name, start := range watchers {
		s.metrics.SetExecutionWatcherUp(name, false)

		s.executionWG.Go(func() {
			s.retryExecutionWatcher(ctx, name, start)
		})
	}

	s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
		cancel()
		s.executionWG.Wait()

		return s.stopExecution(ctx)
	})
}

// retryExecutionWatcher calls start with exponential backoff until it succeeds or ctx ends.
func (s *Sentry) retryExecutionWatcher(ctx context.Context, name string, start func(context.Context) error) {
	backoff := executionWatcherInitialBackoff

	for {
		err := start(ctx)
		if err == nil {
			s.metrics.SetExecutionWatcherUp(name, true)

			return
		}

		s.log.WithError(err).
			WithField("watcher", name).
			WithField("retry_in", backoff).
			WithContext(ctx).
			Warn("Failed to start execution watcher, will retry")

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, executionWatcherMaxBackoff)
	}
}

// executionClient returns the shared execution client, creating and starting it on first use.
func (s *Sentry) executionClient(ctx context.Context) (*execution.Client, error) {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	if s.execution != nil {
		return s.execution, nil
	}

	client, err := execution.NewClient(ctx, s.log, s.Config.Execution)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution client: %w", err)
	}

	if err := client.Start(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to start execution client: %w", err), client.Stop(ctx))
	}

	s.execution = client

	return client, nil
}

// addExecutionStop registers a running watcher's stop function.
func (s *Sentry) addExecutionStop(stop func(context.Context) error) {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	s.executionStops = append(s.executionStops, stop)
}

// stopExecution stops the watchers in reverse start order, then the shared client they use.
func (s *Sentry) stopExecution(ctx context.Context) error {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	var errs []error

	for i := len(s.executionStops) - 1; i >= 0; i-- {
		errs = append(errs, s.executionStops[i](ctx))
	}

	s.executionStops = nil

	if s.execution != nil {
		errs = append(errs, s.execution.Stop(ctx))
	}

	return errors.Join(errs...)
}
