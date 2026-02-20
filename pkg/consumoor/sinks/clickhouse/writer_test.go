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
