package consumoor

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse"
	"github.com/ethpandaops/xatu/pkg/consumoor/source"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisabledEventEnums(t *testing.T) {
	t.Run("parses valid names", func(t *testing.T) {
		cfg := &Config{
			DisabledEvents: []string{
				"BEACON_API_ETH_V1_EVENTS_HEAD",
				"LIBP2P_TRACE_CONNECTED",
			},
		}

		events, err := cfg.DisabledEventEnums()
		require.NoError(t, err)
		assert.Equal(
			t,
			[]xatu.Event_Name{
				xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				xatu.Event_LIBP2P_TRACE_CONNECTED,
			},
			events,
		)
	})

	t.Run("rejects unknown names", func(t *testing.T) {
		cfg := &Config{
			DisabledEvents: []string{
				"BEACON_API_ETH_V1_EVENTS_HEAD",
				"NOT_A_REAL_EVENT",
			},
		}

		_, err := cfg.DisabledEventEnums()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "NOT_A_REAL_EVENT")
	})
}

func TestClickHouseConfigValidateChGo(t *testing.T) {
	t.Run("accepts valid ch-go settings", func(t *testing.T) {
		cfg := &clickhouse.Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: clickhouse.TableConfig{
				BatchSize:     1,
				BatchBytes:    1,
				FlushInterval: time.Second,
				BufferSize:    1,
			},
			ChGo: clickhouse.ChGoConfig{
				QueryTimeout:        30 * time.Second,
				MaxRetries:          3,
				RetryBaseDelay:      100 * time.Millisecond,
				RetryMaxDelay:       2 * time.Second,
				MaxConns:            8,
				MinConns:            1,
				ConnMaxLifetime:     time.Hour,
				ConnMaxIdleTime:     10 * time.Minute,
				HealthCheckPeriod:   30 * time.Second,
				PoolMetricsInterval: 15 * time.Second,
			},
		}

		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects invalid pool bounds", func(t *testing.T) {
		cfg := &clickhouse.Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: clickhouse.TableConfig{
				BatchSize:     1,
				BatchBytes:    1,
				FlushInterval: time.Second,
				BufferSize:    1,
			},
			ChGo: clickhouse.ChGoConfig{
				RetryBaseDelay:    100 * time.Millisecond,
				RetryMaxDelay:     2 * time.Second,
				MaxConns:          2,
				MinConns:          3,
				ConnMaxLifetime:   time.Hour,
				ConnMaxIdleTime:   10 * time.Minute,
				HealthCheckPeriod: 30 * time.Second,
			},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minConns")
	})

	t.Run("rejects negative retries", func(t *testing.T) {
		cfg := &clickhouse.Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: clickhouse.TableConfig{
				BatchSize:     1,
				BatchBytes:    1,
				FlushInterval: time.Second,
				BufferSize:    1,
			},
			ChGo: clickhouse.ChGoConfig{
				MaxRetries:        -1,
				RetryBaseDelay:    100 * time.Millisecond,
				RetryMaxDelay:     2 * time.Second,
				MaxConns:          8,
				MinConns:          1,
				ConnMaxLifetime:   time.Hour,
				ConnMaxIdleTime:   10 * time.Minute,
				HealthCheckPeriod: 30 * time.Second,
			},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxRetries")
	})

	t.Run("rejects unsupported insertSettings value type", func(t *testing.T) {
		cfg := &clickhouse.Config{
			DSN: "clickhouse://localhost:9000/default",
			Defaults: clickhouse.TableConfig{
				BatchSize:     1,
				BatchBytes:    1,
				FlushInterval: time.Second,
				BufferSize:    1,
				InsertSettings: map[string]any{
					"bad_setting": []int{1, 2},
				},
			},
			ChGo: clickhouse.ChGoConfig{
				RetryBaseDelay:    100 * time.Millisecond,
				RetryMaxDelay:     2 * time.Second,
				MaxConns:          8,
				MinConns:          1,
				ConnMaxLifetime:   time.Hour,
				ConnMaxIdleTime:   10 * time.Minute,
				HealthCheckPeriod: 30 * time.Second,
			},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported value type")
	})
}

func TestTableConfigForMergesInsertSettings(t *testing.T) {
	cfg := &clickhouse.Config{
		Defaults: clickhouse.TableConfig{
			BatchSize:     100,
			BatchBytes:    200,
			FlushInterval: time.Second,
			BufferSize:    300,
			InsertSettings: map[string]any{
				"insert_quorum":         2,
				"insert_quorum_timeout": 30000,
			},
		},
		Tables: map[string]clickhouse.TableConfig{
			"canonical_beacon_block": {
				BatchSize: 500,
				InsertSettings: map[string]any{
					"insert_quorum": 3,
				},
			},
		},
	}

	got := cfg.TableConfigFor("canonical_beacon_block")

	assert.Equal(t, 500, got.BatchSize)
	assert.Equal(t, 200, got.BatchBytes)
	assert.Equal(t, 300, got.BufferSize)
	assert.Equal(
		t,
		map[string]any{
			"insert_quorum":         3,
			"insert_quorum_timeout": 30000,
		},
		got.InsertSettings,
	)
}

func TestTableConfigForAppliesCanonicalDefaults(t *testing.T) {
	t.Run("adds auto quorum for canonical tables when unset", func(t *testing.T) {
		cfg := &clickhouse.Config{
			Defaults: clickhouse.TableConfig{
				BatchSize:     100,
				BatchBytes:    200,
				FlushInterval: time.Second,
				BufferSize:    300,
			},
		}

		got := cfg.TableConfigFor("canonical_beacon_block")
		assert.Equal(t, map[string]any{"insert_quorum": "auto"}, got.InsertSettings)
	})

	t.Run("does not add quorum for non-canonical tables", func(t *testing.T) {
		cfg := &clickhouse.Config{
			Defaults: clickhouse.TableConfig{
				BatchSize:     100,
				BatchBytes:    200,
				FlushInterval: time.Second,
				BufferSize:    300,
			},
		}

		got := cfg.TableConfigFor("beacon_api_eth_v1_events_head")
		assert.Nil(t, got.InsertSettings)
	})

	t.Run("preserves explicit per-table quorum", func(t *testing.T) {
		cfg := &clickhouse.Config{
			Defaults: clickhouse.TableConfig{
				BatchSize:     100,
				BatchBytes:    200,
				FlushInterval: time.Second,
				BufferSize:    300,
			},
			Tables: map[string]clickhouse.TableConfig{
				"canonical_beacon_block": {
					InsertSettings: map[string]any{
						"insert_quorum": 3,
					},
				},
			},
		}

		got := cfg.TableConfigFor("canonical_beacon_block")
		assert.Equal(t, map[string]any{"insert_quorum": 3}, got.InsertSettings)
	})

	t.Run("preserves explicit default quorum", func(t *testing.T) {
		cfg := &clickhouse.Config{
			Defaults: clickhouse.TableConfig{
				BatchSize:     100,
				BatchBytes:    200,
				FlushInterval: time.Second,
				BufferSize:    300,
				InsertSettings: map[string]any{
					"insert_quorum": 2,
				},
			},
		}

		got := cfg.TableConfigFor("canonical_beacon_block")
		assert.Equal(t, map[string]any{"insert_quorum": 2}, got.InsertSettings)
	})
}

func TestKafkaConfigValidateDeliveryMode(t *testing.T) {
	base := source.KafkaConfig{
		Brokers:        []string{"localhost:9092"},
		Topics:         []string{"^general-.+"},
		ConsumerGroup:  "xatu-consumoor",
		Encoding:       "json",
		OffsetDefault:  "oldest",
		CommitInterval: 5 * time.Second,
	}

	t.Run("defaults empty mode to batch", func(t *testing.T) {
		cfg := base
		cfg.DeliveryMode = ""

		require.NoError(t, cfg.Validate())
		assert.Equal(t, source.DeliveryModeBatch, cfg.DeliveryMode)
	})

	t.Run("accepts message mode", func(t *testing.T) {
		cfg := base
		cfg.DeliveryMode = source.DeliveryModeMessage

		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects unknown mode", func(t *testing.T) {
		cfg := base
		cfg.DeliveryMode = "ultra-fast"

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deliveryMode")
	})
}
