package consumoor

import (
	"context"
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToTimeNumericValuesUseUnixSecondsByDefault(t *testing.T) {
	expected := time.Unix(1_700_000_000, 0).UTC()

	cases := []any{
		int64(1_700_000_000),
		int(1_700_000_000),
		uint64(1_700_000_000),
		float64(1_700_000_000),
		"1700000000",
	}

	for _, value := range cases {
		got, err := toTime(value, "updated_date_time")
		require.NoError(t, err)
		assert.True(t, got.Equal(expected), "value=%T %v", value, value)
	}
}

func TestToTimeEventDateTimeUsesUnixMilliseconds(t *testing.T) {
	expected := time.UnixMilli(1_700_000_000_000).UTC()

	cases := []any{
		int64(1_700_000_000_000),
		int(1_700_000_000_000),
		uint64(1_700_000_000_000),
		float64(1_700_000_000_000),
		"1700000000000",
	}

	for _, value := range cases {
		got, err := toTime(value, "event_date_time")
		require.NoError(t, err)
		assert.True(t, got.Equal(expected), "value=%T %v", value, value)
	}
}

func TestSortedColumnsUsesAllRows(t *testing.T) {
	cols := sortedColumns([]map[string]any{
		{"validator_id": uint64(1)},
		{"balance": uint64(32_000_000_000)},
	})

	assert.Equal(t, []string{"balance", "validator_id"}, cols)
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

func TestResolvedChGoConfigAppliesRequiredDefaults(t *testing.T) {
	cfg := resolvedChGoConfig(ChGoConfig{})

	assert.Equal(t, defaultChGoRetryBaseDelay, cfg.RetryBaseDelay)
	assert.Equal(t, defaultChGoRetryMaxDelay, cfg.RetryMaxDelay)
	assert.Equal(t, defaultChGoMaxConns, cfg.MaxConns)
	assert.Equal(t, int32(0), cfg.MinConns)
	assert.Equal(t, defaultChGoConnMaxLife, cfg.ConnMaxLifetime)
	assert.Equal(t, defaultChGoConnMaxIdle, cfg.ConnMaxIdleTime)
	assert.Equal(t, defaultChGoHealthCheck, cfg.HealthCheckPeriod)
}

func TestResolvedChGoConfigHonorsExplicitZeroMinConns(t *testing.T) {
	cfg := resolvedChGoConfig(ChGoConfig{MinConns: 0, MaxConns: 5})

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
