package sentry

import (
	"context"
	"errors"
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

		start := func(context.Context) error {
			attempts = append(attempts, time.Now())
			if len(attempts) < 4 {
				return errors.New("execution node unreachable")
			}

			return nil
		}

		began := time.Now()

		s.retryExecutionWatcher(t.Context(), executionWatcherMempool, start)

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

		start := func(context.Context) error {
			attempts = append(attempts, time.Now())
			if len(attempts) == 10 {
				cancel()
			}

			return errors.New("execution node unreachable")
		}

		s.retryExecutionWatcher(ctx, executionWatcherStateSize, start)

		require.Len(t, attempts, 10)
		assert.Equal(t, executionWatcherMaxBackoff, attempts[9].Sub(attempts[8]))
		assert.InDelta(t, 0, watcherUp(t, s, executionWatcherStateSize), 0)
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
		s.addExecutionStop(func(context.Context) error {
			stopped = append(stopped, name)

			return nil
		})
	}

	require.NoError(t, s.stopExecution(t.Context()))
	assert.Equal(t, []string{"second", "first"}, stopped)
	assert.Empty(t, s.executionStops)
}
