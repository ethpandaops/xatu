package sentry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

const (
	executionWatcherMempool   = "mempool"
	executionWatcherStateSize = "state_size"

	executionWatcherInitialBackoff = 5 * time.Second
	executionWatcherMaxBackoff     = 5 * time.Minute

	// executionHealthInterval is how often running watchers are health checked.
	executionHealthInterval = 15 * time.Second
	// executionRebuildAfter is how long a watcher may stay unhealthy before the client and every watcher are rebuilt.
	executionRebuildAfter = 2 * time.Minute
)

var errExecutionStopped = errors.New("execution watchers are stopped")

// runningWatcher is a started execution watcher; registered, when set, runs once it is registered to be stopped.
type runningWatcher struct {
	healthy    func() bool
	stop       func() error
	registered func()
}

// executionWatcherSpec starts one kind of execution watcher on the shared client.
type executionWatcherSpec struct {
	name  string
	start func(ctx context.Context, client *execution.Client) (*runningWatcher, error)
}

// startExecutionWatchers supervises the enabled execution watchers in the background: each start retries until it
// succeeds, and a watcher that stays unhealthy gets the client and every watcher rebuilt.
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

	specs := []executionWatcherSpec{{
		name: executionWatcherMempool,
		start: func(ctx context.Context, client *execution.Client) (*runningWatcher, error) {
			return s.startMempoolTransactionWatcher(ctx, client, mempoolMetrics)
		},
	}}

	if s.Config.Execution.StateSize != nil && s.Config.Execution.StateSize.Enabled {
		specs = append(specs, executionWatcherSpec{name: executionWatcherStateSize, start: s.startExecutionStateSizeWatcher})
	} else {
		s.log.WithContext(ctx).Info("Execution debug state size watcher disabled")
	}

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	registered := s.addShutdownFunc(func(ctx context.Context) error {
		cancel()

		select {
		case <-done:
		case <-ctx.Done():
		}

		return s.stopExecution(ctx, true)
	})
	if !registered {
		cancel()
		s.log.WithContext(ctx).Info("Shutdown already started, not starting execution watchers")

		return
	}

	for _, spec := range specs {
		s.metrics.SetExecutionWatcherUp(spec.name, false)
	}

	go func() {
		defer close(done)

		s.superviseExecutionWatchers(ctx, specs)
	}()
}

// superviseExecutionWatchers starts every watcher, then health checks them, rebuilding the set when one stays unhealthy.
func (s *Sentry) superviseExecutionWatchers(ctx context.Context, specs []executionWatcherSpec) {
	for {
		running := s.startExecutionWatcherSet(ctx, specs)
		if ctx.Err() != nil {
			return
		}

		if !s.watchExecutionHealth(ctx, running) {
			return
		}

		for _, spec := range specs {
			s.metrics.SetExecutionWatcherUp(spec.name, false)
		}

		if err := s.stopExecution(ctx, false); err != nil {
			s.log.WithError(err).WithContext(ctx).Warn("Failed to stop execution watchers for a rebuild")
		}
	}
}

// startExecutionWatcherSet starts every watcher concurrently and returns once all are running or ctx ends.
func (s *Sentry) startExecutionWatcherSet(ctx context.Context, specs []executionWatcherSpec) map[string]*runningWatcher {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		running = make(map[string]*runningWatcher, len(specs))
	)

	for _, spec := range specs {
		wg.Go(func() {
			w := s.retryExecutionWatcher(ctx, spec.name, func(ctx context.Context) (*runningWatcher, error) {
				return s.startExecutionWatcher(ctx, spec)
			})
			if w == nil {
				return
			}

			mu.Lock()
			running[spec.name] = w
			mu.Unlock()
		})
	}

	wg.Wait()

	return running
}

// retryExecutionWatcher calls start with exponential backoff until it succeeds, or returns nil once ctx ends.
func (s *Sentry) retryExecutionWatcher(ctx context.Context, name string, start func(context.Context) (*runningWatcher, error)) *runningWatcher {
	backoff := executionWatcherInitialBackoff

	for {
		w, err := start(ctx)
		if err == nil {
			s.metrics.SetExecutionWatcherUp(name, true)

			return w
		}

		s.log.WithError(err).
			WithField("watcher", name).
			WithField("retry_in", backoff).
			WithContext(ctx).
			Warn("Failed to start execution watcher, will retry")

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, executionWatcherMaxBackoff)
	}
}

// startExecutionWatcher starts one watcher on the shared client and registers it to be stopped.
func (s *Sentry) startExecutionWatcher(ctx context.Context, spec executionWatcherSpec) (*runningWatcher, error) {
	client, err := s.executionClient(ctx)
	if err != nil {
		return nil, err
	}

	w, err := spec.start(ctx, client)
	if err != nil {
		return nil, err
	}

	if !s.addExecutionStop(w.stop) {
		return nil, errors.Join(errExecutionStopped, w.stop())
	}

	if w.registered != nil {
		w.registered()
	}

	return w, nil
}

// watchExecutionHealth reports each watcher's health on its gauge, and returns true once one has been unhealthy for
// executionRebuildAfter, or false when ctx ends.
func (s *Sentry) watchExecutionHealth(ctx context.Context, running map[string]*runningWatcher) bool {
	ticker := time.NewTicker(executionHealthInterval)
	defer ticker.Stop()

	unhealthySince := make(map[string]time.Time, len(running))

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}

		for name, w := range running {
			if w.healthy() {
				s.metrics.SetExecutionWatcherUp(name, true)
				delete(unhealthySince, name)

				continue
			}

			s.metrics.SetExecutionWatcherUp(name, false)

			since, ok := unhealthySince[name]
			if !ok {
				unhealthySince[name] = time.Now()

				continue
			}

			if time.Since(since) >= executionRebuildAfter {
				s.log.WithField("watcher", name).
					WithField("unhealthy_for", time.Since(since).Round(time.Second)).
					WithContext(ctx).
					Warn("Execution watcher unhealthy, rebuilding the execution client and watchers")

				return true
			}
		}
	}
}

// executionClient returns the shared execution client, creating and starting it on first use.
func (s *Sentry) executionClient(ctx context.Context) (*execution.Client, error) {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	if s.executionStopped {
		return nil, errExecutionStopped
	}

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

// addExecutionStop registers a running watcher's stop function, and reports false once the watchers are stopped for good.
func (s *Sentry) addExecutionStop(stop func() error) bool {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	if s.executionStopped {
		return false
	}

	s.executionStops = append(s.executionStops, stop)

	return true
}

// stopExecution stops the watchers in reverse start order, then the shared client; final keeps new ones from starting.
func (s *Sentry) stopExecution(ctx context.Context, final bool) error {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()

	if final {
		s.executionStopped = true
	}

	s.stateSizeWatcher.Store(nil)

	var errs []error

	for i := len(s.executionStops) - 1; i >= 0; i-- {
		errs = append(errs, s.executionStops[i]())
	}

	s.executionStops = nil

	if s.execution != nil {
		errs = append(errs, s.execution.Stop(ctx))
		s.execution = nil
	}

	return errors.Join(errs...)
}
