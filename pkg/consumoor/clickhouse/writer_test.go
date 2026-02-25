package clickhouse

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMetrics     *telemetry.Metrics
	testMetricsOnce sync.Once
)

// sharedTestMetrics returns a single Metrics instance shared across all tests
// to avoid duplicate Prometheus registration panics from promauto.
func sharedTestMetrics() *telemetry.Metrics {
	testMetricsOnce.Do(func() {
		testMetrics = telemetry.NewMetrics("test_chgo")
	})

	return testMetrics
}

// newTestWriter builds a minimal ChGoWriter suitable for unit tests that
// exercise doWithRetry and getOrCreateTableWriter without a real ClickHouse.
func newTestWriter(maxRetries int, baseDelay, maxDelay time.Duration) *ChGoWriter {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return &ChGoWriter{
		log: log.WithField("component", "test"),
		config: &Config{
			DSN:          "clickhouse://localhost:9000/default",
			DrainTimeout: 30 * time.Second,
			Defaults: TableConfig{
				BufferSize: 100,
			},
		},
		metrics: sharedTestMetrics(),
		chgoCfg: ChGoConfig{
			MaxRetries:     maxRetries,
			RetryBaseDelay: baseDelay,
			RetryMaxDelay:  maxDelay,
			// QueryTimeout=0 disables per-attempt timeout wrapping.
		},
		tables:         make(map[string]*chTableWriter, 16),
		batchFactories: make(map[string]func() route.ColumnarBatch, 8),
		done:           make(chan struct{}),
	}
}

func TestIsRetryableError(t *testing.T) {
	assert.False(t, isRetryableError(nil))
	assert.False(t, isRetryableError(context.Canceled))
	assert.False(t, isRetryableError(context.DeadlineExceeded))
	assert.False(t, isRetryableError(errors.New("invalid schema")))
	assert.True(t, isRetryableError(io.EOF))
	assert.True(t, isRetryableError(syscall.ECONNRESET))
	assert.True(t, isRetryableError(errors.New("server is overloaded")))
}

func TestChGoConfigValidateRejectsZeroValues(t *testing.T) {
	cfg := ChGoConfig{}
	assert.Error(t, cfg.Validate(), "zero-value ChGoConfig should fail validation")
}

func TestChGoConfigValidateAcceptsValidConfig(t *testing.T) {
	cfg := ChGoConfig{
		DialTimeout:       5 * time.Second,
		ReadTimeout:       30 * time.Second,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     2 * time.Second,
		MaxConns:          32,
		MinConns:          0,
		ConnMaxLifetime:   time.Hour,
		ConnMaxIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}
	assert.NoError(t, cfg.Validate())
}

func TestChGoConfigValidateHonorsExplicitZeroMinConns(t *testing.T) {
	cfg := ChGoConfig{
		DialTimeout:       5 * time.Second,
		ReadTimeout:       30 * time.Second,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     2 * time.Second,
		MaxConns:          5,
		MinConns:          0,
		ConnMaxLifetime:   time.Hour,
		ConnMaxIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}
	assert.NoError(t, cfg.Validate())
	assert.Equal(t, int32(0), cfg.MinConns)
	assert.Equal(t, int32(5), cfg.MaxConns)
}

func TestInsertQueryWithSettings(t *testing.T) {
	query, err := insertQueryWithSettings(
		"INSERT INTO canonical_beacon_block (slot, root) VALUES",
		map[string]any{
			"insert_quorum":         2,
			"insert_quorum_timeout": 60000,
		},
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		"INSERT INTO canonical_beacon_block (slot, root) SETTINGS insert_quorum = 2, insert_quorum_timeout = 60000 VALUES",
		query,
	)
}

func TestInsertQueryWithSettingsRejectsUnsupportedValue(t *testing.T) {
	_, err := insertQueryWithSettings(
		"INSERT INTO canonical_beacon_block (slot, root) VALUES",
		map[string]any{
			"bad": map[string]any{"nested": true},
		},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type")
}

func TestIsPermanentWriteError(t *testing.T) {
	assert.True(t, IsPermanentWriteError(&tableWriteError{
		table: "beacon_head",
		cause: &inputPrepError{cause: errors.New("no batch factory")},
	}))

	assert.True(t, IsPermanentWriteError(&tableWriteError{
		table: "beacon_head",
		cause: &ch.Exception{
			Code:    proto.ErrUnknownIdentifier,
			Name:    "UNKNOWN_IDENTIFIER",
			Message: "unknown column",
		},
	}))

	assert.False(t, IsPermanentWriteError(errors.New("dial tcp timeout")))
}

func TestFlattenErrorIsNotPermanent(t *testing.T) {
	err := &tableWriteError{
		table: "beacon_head",
		cause: &flattenError{cause: errors.New("bad proto field")},
	}

	assert.False(t, IsPermanentWriteError(err),
		"flattenError must NOT be classified as permanent")

	var flatErr *flattenError
	assert.True(t, errors.As(err, &flatErr),
		"flattenError should be extractable via errors.As")
}

func TestFlattenErrorMessage(t *testing.T) {
	err := &flattenError{cause: errors.New("missing field")}
	assert.Equal(t, "flatten failed: missing field", err.Error())
	assert.Equal(t, "missing field", errors.Unwrap(err).Error())
}

func TestIsPermanentWriteErrorJoined(t *testing.T) {
	permanent := &tableWriteError{
		table: "table_a",
		cause: &inputPrepError{cause: errors.New("schema mismatch")},
	}
	transient := &tableWriteError{
		table: "table_b",
		cause: errors.New("connection reset"),
	}

	t.Run("joined with permanent sub-error", func(t *testing.T) {
		joined := errors.Join(permanent, transient)
		assert.True(t, IsPermanentWriteError(joined),
			"joined error containing a permanent sub-error should be permanent")
	})

	t.Run("joined with only transient sub-errors", func(t *testing.T) {
		transient2 := &tableWriteError{
			table: "table_c",
			cause: errors.New("timeout"),
		}
		joined := errors.Join(transient, transient2)
		assert.False(t, IsPermanentWriteError(joined),
			"joined error with only transient sub-errors should not be permanent")
	})
}

// ---------------------------------------------------------------------------
// doWithRetry tests
// ---------------------------------------------------------------------------

func TestDoWithRetry_SucceedsFirstTry(t *testing.T) {
	w := newTestWriter(3, time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		calls.Add(1)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoWithRetry_SucceedsAfterRetries(t *testing.T) {
	w := newTestWriter(3, time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		n := calls.Add(1)
		if n < 3 {
			return io.EOF // retryable
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}

func TestDoWithRetry_ExhaustsAllRetries(t *testing.T) {
	w := newTestWriter(2, time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		calls.Add(1)

		return io.EOF // always retryable
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded max retries (2)")
	assert.ErrorIs(t, err, io.EOF)
	// 1 initial + 2 retries = 3 total calls.
	assert.Equal(t, int32(3), calls.Load())
}

func TestDoWithRetry_ContextCancellationMidRetry(t *testing.T) {
	w := newTestWriter(5, 50*time.Millisecond, 200*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	var calls atomic.Int32

	err := w.doWithRetry(ctx, "test_op", func(_ context.Context) error {
		n := calls.Add(1)
		if n == 1 {
			// Cancel after first failure so the retry backoff select
			// sees ctx.Done().
			cancel()
		}

		return io.EOF
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	// Should have been called once, then cancel fires during backoff.
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoWithRetry_NonRetryableExitsImmediately(t *testing.T) {
	w := newTestWriter(5, time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32

	permanent := errors.New("invalid schema")

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		calls.Add(1)

		return permanent
	})

	require.Error(t, err)
	assert.Equal(t, permanent, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestDoWithRetry_LimiterRejectionExitsImmediately(t *testing.T) {
	w := newTestWriter(5, time.Millisecond, 10*time.Millisecond)

	var calls atomic.Int32

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		calls.Add(1)

		return &limiterRejectedError{cause: errors.New("limit exceeded")}
	})

	require.Error(t, err)
	assert.True(t, IsLimiterRejected(err), "error should be classified as limiter rejected")
	assert.Equal(t, int32(1), calls.Load(), "should not retry on limiter rejection")
}

func TestDoWithRetry_BackoffTiming(t *testing.T) {
	baseDelay := 20 * time.Millisecond
	maxDelay := 100 * time.Millisecond
	w := newTestWriter(3, baseDelay, maxDelay)

	var (
		calls      atomic.Int32
		timestamps []time.Time
		mu         sync.Mutex
	)

	err := w.doWithRetry(context.Background(), "test_op", func(_ context.Context) error {
		mu.Lock()

		timestamps = append(timestamps, time.Now())

		mu.Unlock()

		calls.Add(1)

		return io.EOF
	})

	require.Error(t, err)
	assert.Equal(t, int32(4), calls.Load()) // 1 initial + 3 retries

	mu.Lock()
	defer mu.Unlock()

	// Verify delays between attempts increase (with tolerance for scheduling).
	for i := 1; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		// Each gap should be at least ~half the expected delay to account
		// for scheduling jitter. Expected delays: 20ms, 40ms, 80ms (capped).
		expectedMin := baseDelay * time.Duration(1<<(i-1)) / 2
		if expectedMin > maxDelay {
			expectedMin = maxDelay / 2
		}

		assert.GreaterOrEqual(t, gap, expectedMin,
			"attempt %d delay %v should be >= %v", i, gap, expectedMin)
	}
}

// ---------------------------------------------------------------------------
// getOrCreateTableWriter tests
// ---------------------------------------------------------------------------

func TestGetOrCreateTableWriter_ReturnsExisting(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)

	tw1 := w.getOrCreateTableWriter("beacon_block")
	tw2 := w.getOrCreateTableWriter("beacon_block")

	assert.Same(t, tw1, tw2, "second call should return the same writer")
}

func TestGetOrCreateTableWriter_ConcurrentAccess(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)

	const goroutines = 50

	results := make([]*chTableWriter, goroutines)

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			results[idx] = w.getOrCreateTableWriter("beacon_block")
		}(i)
	}

	wg.Wait()

	// All goroutines should have gotten the same writer.
	for i := 1; i < goroutines; i++ {
		assert.Same(t, results[0], results[i],
			"goroutine %d got a different writer", i)
	}

	// Only one writer should exist in the map.
	w.mu.RLock()
	defer w.mu.RUnlock()

	assert.Len(t, w.tables, 1)
}

func TestGetOrCreateTableWriter_AppliesTableSuffix(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)
	w.config.TableSuffix = "_local"

	tw := w.getOrCreateTableWriter("beacon_block")

	assert.Equal(t, "beacon_block_local", tw.table)
}

func TestGetOrCreateTableWriter_DifferentTablesGetDifferentWriters(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)

	tw1 := w.getOrCreateTableWriter("beacon_block")
	tw2 := w.getOrCreateTableWriter("beacon_blob_sidecar")

	assert.NotSame(t, tw1, tw2)

	w.mu.RLock()
	defer w.mu.RUnlock()

	assert.Len(t, w.tables, 2)
}

func TestFlushTables_ConcurrentDrain(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)
	w.poolDoFn = func(_ context.Context, _ ch.Query) error {
		return nil
	}

	const table = "beacon_block"

	// Register a batch factory so flush actually works.
	w.batchFactories[table] = func() route.ColumnarBatch {
		return &stubBatch{}
	}

	tw := w.getOrCreateTableWriter(table)

	// Fill the buffer with some events.
	const eventCount = 50

	for i := 0; i < eventCount; i++ {
		tw.buffer <- eventEntry{}
	}

	// Launch multiple concurrent FlushTables calls.
	const flushers = 10

	errs := make([]error, flushers)

	var wg sync.WaitGroup

	wg.Add(flushers)

	for i := 0; i < flushers; i++ {
		go func(idx int) {
			defer wg.Done()

			errs[idx] = w.FlushTables(context.Background(), []string{table})
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "flusher %d returned error", i)
	}

	// Buffer should be fully drained.
	assert.Equal(t, 0, len(tw.buffer))
}

func TestStop_DrainsRemainingEvents(t *testing.T) {
	w := newTestWriter(0, time.Millisecond, time.Millisecond)

	var flushed atomic.Int32

	w.poolDoFn = func(_ context.Context, _ ch.Query) error {
		flushed.Add(1)

		return nil
	}

	const table = "beacon_block"

	w.batchFactories[table] = func() route.ColumnarBatch {
		return &stubBatch{}
	}

	tw := w.getOrCreateTableWriter(table)

	// Buffer some events.
	const eventCount = 25

	for i := 0; i < eventCount; i++ {
		tw.buffer <- eventEntry{}
	}

	// Stop should drain and flush remaining events.
	err := w.Stop(context.Background())
	require.NoError(t, err)

	assert.Greater(t, flushed.Load(), int32(0), "should have flushed at least once")
	assert.Equal(t, 0, len(tw.buffer), "buffer should be empty after stop")
}

// stubBatch is a minimal ColumnarBatch implementation for unit tests.
type stubBatch struct {
	rows int
}

func (s *stubBatch) FlattenTo(_ *xatu.DecoratedEvent) error {
	s.rows++

	return nil
}

func (s *stubBatch) Rows() int { return s.rows }

func (s *stubBatch) Input() proto.Input { return proto.Input{} }

func (s *stubBatch) Reset() { s.rows = 0 }

func TestTableConfigMergeSkipFlattenErrors(t *testing.T) {
	t.Run("default inherits from defaults", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost",
			Defaults: TableConfig{
				BufferSize:        1000,
				SkipFlattenErrors: true,
			},
		}
		tc := cfg.TableConfigFor("some_table")
		assert.True(t, tc.SkipFlattenErrors)
	})

	t.Run("override enables skip", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost",
			Defaults: TableConfig{
				BufferSize: 1000,
			},
			Tables: map[string]TableConfig{
				"some_table": {SkipFlattenErrors: true},
			},
		}
		tc := cfg.TableConfigFor("some_table")
		assert.True(t, tc.SkipFlattenErrors)
	})

	t.Run("either side true wins", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost",
			Defaults: TableConfig{
				BufferSize:        1000,
				SkipFlattenErrors: true,
			},
			Tables: map[string]TableConfig{
				"some_table": {SkipFlattenErrors: false},
			},
		}
		tc := cfg.TableConfigFor("some_table")
		assert.True(t, tc.SkipFlattenErrors, "defaults=true || override=false should be true")
	})

	t.Run("both false stays false", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost",
			Defaults: TableConfig{
				BufferSize: 1000,
			},
			Tables: map[string]TableConfig{
				"some_table": {},
			},
		}
		tc := cfg.TableConfigFor("some_table")
		assert.False(t, tc.SkipFlattenErrors)
	})
}
