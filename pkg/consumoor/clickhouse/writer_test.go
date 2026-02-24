package clickhouse

import (
	"context"
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
