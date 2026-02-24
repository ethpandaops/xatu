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
				BatchSize:     100,
				FlushInterval: time.Second,
				BufferSize:    100,
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
		MaxConns:          8,
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

func TestWriteErrorTable(t *testing.T) {
	assert.Equal(t, "table_b", WriteErrorTable(&tableWriteError{
		table: "table_b",
		cause: errors.New("boom"),
	}))
	assert.Equal(t, "", WriteErrorTable(errors.New("boom")))
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

func TestBaseTableUsedInWriteError(t *testing.T) {
	err := &tableWriteError{
		table: "beacon_head",
		cause: errors.New("boom"),
	}
	assert.Equal(t, "beacon_head", WriteErrorTable(err))

	// Simulates suffixed writer: baseTable != table.
	errSuffixed := &tableWriteError{
		table: "beacon_head", // baseTable, not "beacon_head_local"
		cause: errors.New("boom"),
	}
	assert.Equal(t, "beacon_head", WriteErrorTable(errSuffixed))
}

func TestWriteErrorTableJoinedErrors(t *testing.T) {
	errA := &tableWriteError{table: "table_a", cause: errors.New("boom a")}
	errB := &tableWriteError{table: "table_b", cause: errors.New("boom b")}

	joined := errors.Join(errA, errB)

	// errors.As on a joined error finds the first match, so
	// WriteErrorTable only returns one table for the top-level error.
	// The source package handles unwrapping all sub-errors.
	table := WriteErrorTable(joined)
	assert.Contains(t, []string{"table_a", "table_b"}, table,
		"should extract a table name from joined error via errors.As")
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

func TestBufferWarningThresholdValidation(t *testing.T) {
	validChGo := ChGoConfig{
		DialTimeout:       5 * time.Second,
		ReadTimeout:       30 * time.Second,
		RetryBaseDelay:    100 * time.Millisecond,
		RetryMaxDelay:     2 * time.Second,
		MaxConns:          8,
		MinConns:          1,
		ConnMaxLifetime:   time.Hour,
		ConnMaxIdleTime:   10 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}

	t.Run("accepts 0", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: TableConfig{
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			OrganicRetryInitDelay:  time.Second,
			OrganicRetryMaxDelay:   30 * time.Second,
			DrainTimeout:           30 * time.Second,
			BufferWarningThreshold: 0,
			ChGo:                   validChGo,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("accepts 0.8", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: TableConfig{
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			OrganicRetryInitDelay:  time.Second,
			OrganicRetryMaxDelay:   30 * time.Second,
			DrainTimeout:           30 * time.Second,
			BufferWarningThreshold: 0.8,
			ChGo:                   validChGo,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("accepts 1", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: TableConfig{
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			OrganicRetryInitDelay:  time.Second,
			OrganicRetryMaxDelay:   30 * time.Second,
			DrainTimeout:           30 * time.Second,
			BufferWarningThreshold: 1.0,
			ChGo:                   validChGo,
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects negative", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: TableConfig{
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			OrganicRetryInitDelay:  time.Second,
			OrganicRetryMaxDelay:   30 * time.Second,
			DrainTimeout:           30 * time.Second,
			BufferWarningThreshold: -0.1,
			ChGo:                   validChGo,
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bufferWarningThreshold")
	})

	t.Run("rejects greater than 1", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: TableConfig{
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			OrganicRetryInitDelay:  time.Second,
			OrganicRetryMaxDelay:   30 * time.Second,
			DrainTimeout:           30 * time.Second,
			BufferWarningThreshold: 1.5,
			ChGo:                   validChGo,
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bufferWarningThreshold")
	})
}

func TestCheckBufferWarning(t *testing.T) {
	t.Run("no warning below threshold", func(t *testing.T) {
		tw := &chTableWriter{
			log:    logrus.New().WithField("test", true),
			table:  "test_table",
			config: TableConfig{BufferSize: 100},
			writer: &ChGoWriter{
				config: &Config{BufferWarningThreshold: 0.8},
			},
			buffer: make(chan eventEntry, 100),
		}

		// Fill to 50% -- below 80% threshold
		for i := 0; i < 50; i++ {
			tw.buffer <- eventEntry{}
		}

		tw.checkBufferWarning()
		assert.Equal(t, int64(0), tw.lastWarnAt.Load(),
			"should not warn when below threshold")
	})

	t.Run("warns above threshold", func(t *testing.T) {
		tw := &chTableWriter{
			log:    logrus.New().WithField("test", true),
			table:  "test_table",
			config: TableConfig{BufferSize: 100},
			writer: &ChGoWriter{
				config: &Config{BufferWarningThreshold: 0.8},
			},
			buffer: make(chan eventEntry, 100),
		}

		// Fill to 85% -- above 80% threshold
		for i := 0; i < 85; i++ {
			tw.buffer <- eventEntry{}
		}

		tw.checkBufferWarning()
		assert.NotEqual(t, int64(0), tw.lastWarnAt.Load(),
			"should warn when above threshold")
	})

	t.Run("rate limits warnings", func(t *testing.T) {
		tw := &chTableWriter{
			log:    logrus.New().WithField("test", true),
			table:  "test_table",
			config: TableConfig{BufferSize: 100},
			writer: &ChGoWriter{
				config: &Config{BufferWarningThreshold: 0.8},
			},
			buffer: make(chan eventEntry, 100),
		}

		// Fill above threshold
		for i := 0; i < 85; i++ {
			tw.buffer <- eventEntry{}
		}

		tw.checkBufferWarning()
		firstWarn := tw.lastWarnAt.Load()
		require.NotEqual(t, int64(0), firstWarn)

		// Second call should not update the timestamp (rate limited)
		tw.checkBufferWarning()
		assert.Equal(t, firstWarn, tw.lastWarnAt.Load(),
			"should rate-limit warnings within the interval")
	})

	t.Run("disabled when threshold is 0", func(t *testing.T) {
		tw := &chTableWriter{
			log:    logrus.New().WithField("test", true),
			table:  "test_table",
			config: TableConfig{BufferSize: 100},
			writer: &ChGoWriter{
				config: &Config{BufferWarningThreshold: 0},
			},
			buffer: make(chan eventEntry, 100),
		}

		// Fill to 100%
		for i := 0; i < 100; i++ {
			tw.buffer <- eventEntry{}
		}

		tw.checkBufferWarning()
		assert.Equal(t, int64(0), tw.lastWarnAt.Load(),
			"should not warn when threshold is disabled (0)")
	})
}

func TestTableConfigMergeSkipFlattenErrors(t *testing.T) {
	t.Run("default inherits from defaults", func(t *testing.T) {
		cfg := &Config{
			DSN: "clickhouse://localhost",
			Defaults: TableConfig{
				BatchSize:         1000,
				FlushInterval:     time.Second,
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
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
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
				BatchSize:         1000,
				FlushInterval:     time.Second,
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
				BatchSize:     1000,
				FlushInterval: time.Second,
				BufferSize:    1000,
			},
			Tables: map[string]TableConfig{
				"some_table": {},
			},
		}
		tc := cfg.TableConfigFor("some_table")
		assert.False(t, tc.SkipFlattenErrors)
	})
}
