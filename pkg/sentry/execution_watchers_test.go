package sentry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

func newWatcherTestSentry(config *execution.Config) *Sentry {
	return &Sentry{
		Config: &Config{Execution: config},
		log:    logrus.New(),
		metrics: &Metrics{
			executionWatcherUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "execution_watcher_up", Help: "test"}, []string{"watcher"}),
		},
	}
}

func watcherUp(t *testing.T, s *Sentry, watcher string) float64 {
	t.Helper()

	metric := &dto.Metric{}
	require.NoError(t, s.metrics.executionWatcherUp.WithLabelValues(watcher).Write(metric))

	return metric.GetGauge().GetValue()
}

func TestRetryExecutionWatcherBacksOffUntilStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newWatcherTestSentry(&execution.Config{})
		s.metrics.SetExecutionWatcherUp(executionWatcherMempool, false)

		var attempts []time.Time

		start := func(context.Context) (*runningWatcher, error) {
			attempts = append(attempts, time.Now())
			if len(attempts) < 4 {
				return nil, errors.New("execution node unreachable")
			}

			return &runningWatcher{}, nil
		}

		began := time.Now()

		require.NotNil(t, s.retryExecutionWatcher(t.Context(), executionWatcherMempool, start))

		require.Len(t, attempts, 4)
		assert.Equal(t, time.Duration(0), attempts[0].Sub(began))
		assert.Equal(t, 5*time.Second, attempts[1].Sub(attempts[0]))
		assert.Equal(t, 10*time.Second, attempts[2].Sub(attempts[1]))
		assert.Equal(t, 20*time.Second, attempts[3].Sub(attempts[2]))
		assert.InDelta(t, 1, watcherUp(t, s, executionWatcherMempool), 0)
	})
}

func TestRetryExecutionWatcherCapsBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newWatcherTestSentry(&execution.Config{})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var attempts []time.Time

		start := func(context.Context) (*runningWatcher, error) {
			attempts = append(attempts, time.Now())
			if len(attempts) == 10 {
				cancel()
			}

			return nil, errors.New("execution node unreachable")
		}

		assert.Nil(t, s.retryExecutionWatcher(ctx, executionWatcherStateSize, start))

		require.Len(t, attempts, 10)
		assert.Equal(t, executionWatcherMaxBackoff, attempts[9].Sub(attempts[8]))
		assert.InDelta(t, 0, watcherUp(t, s, executionWatcherStateSize), 0)
	})
}

func TestWatchExecutionHealthRebuildsAfterSustainedFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newWatcherTestSentry(&execution.Config{})

		var healthy atomic.Bool
		healthy.Store(true)

		running := map[string]*runningWatcher{executionWatcherMempool: {healthy: healthy.Load}}

		rebuild := make(chan bool, 1)

		go func() { rebuild <- s.watchExecutionHealth(t.Context(), running) }()

		time.Sleep(executionHealthInterval + time.Second)
		assert.InDelta(t, 1, watcherUp(t, s, executionWatcherMempool), 0)

		healthy.Store(false)

		failedAt := time.Now()

		time.Sleep(executionHealthInterval)
		assert.InDelta(t, 0, watcherUp(t, s, executionWatcherMempool), 0)

		require.True(t, <-rebuild)
		assert.GreaterOrEqual(t, time.Since(failedAt), executionRebuildAfter)
		assert.Less(t, time.Since(failedAt), executionRebuildAfter+2*executionHealthInterval)
	})
}

func TestWatchExecutionHealthRecoveryResetsTheClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newWatcherTestSentry(&execution.Config{})

		var healthy atomic.Bool

		running := map[string]*runningWatcher{executionWatcherStateSize: {healthy: healthy.Load}}

		ctx, cancel := context.WithCancel(t.Context())
		rebuild := make(chan bool, 1)

		go func() { rebuild <- s.watchExecutionHealth(ctx, running) }()

		for range 3 {
			healthy.Store(false)
			time.Sleep(executionRebuildAfter - executionHealthInterval)
			healthy.Store(true)
			time.Sleep(2 * executionHealthInterval)
		}

		assert.InDelta(t, 1, watcherUp(t, s, executionWatcherStateSize), 0)

		cancel()
		assert.False(t, <-rebuild)
	})
}

func TestExecutionClientFailedStartLeavesNoClient(t *testing.T) {
	s := newWatcherTestSentry(&execution.Config{RPCAddress: "http://127.0.0.1:1"})

	client, err := s.executionClient(t.Context())

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Nil(t, s.execution)
}

func TestStopExecutionStopsWatchersInReverseOrder(t *testing.T) {
	s := newWatcherTestSentry(&execution.Config{})

	var stopped []string

	for _, name := range []string{"first", "second"} {
		require.True(t, s.addExecutionStop(func() error {
			stopped = append(stopped, name)

			return nil
		}))
	}

	require.NoError(t, s.stopExecution(t.Context(), false))
	assert.Equal(t, []string{"second", "first"}, stopped)
	assert.Empty(t, s.executionStops)

	require.True(t, s.addExecutionStop(func() error { return nil }), "a rebuild leaves watchers free to start again")
}

func TestWatcherStartedAfterFinalStopStopsItself(t *testing.T) {
	s := newWatcherTestSentry(&execution.Config{})

	require.NoError(t, s.stopExecution(t.Context(), true))

	stopped := false
	w := &runningWatcher{stop: func() error {
		stopped = true

		return nil
	}}

	_, err := s.startExecutionWatcher(t.Context(), executionWatcherSpec{
		name:  executionWatcherMempool,
		start: func(context.Context, *execution.Client) (*runningWatcher, error) { return w, nil },
	})
	require.ErrorIs(t, err, errExecutionStopped)
	assert.False(t, stopped, "no client is handed out once stopped, so the watcher never starts")

	assert.False(t, s.addExecutionStop(w.stop))
}

func TestShutdownFuncsRejectedOnceShutdownBegins(t *testing.T) {
	s := newWatcherTestSentry(&execution.Config{})

	require.True(t, s.addShutdownFunc(func(context.Context) error { return nil }))
	assert.Len(t, s.takeShutdownFuncs(), 1)
	assert.False(t, s.addShutdownFunc(func(context.Context) error { return nil }))
}
