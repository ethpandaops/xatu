package consumoor

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/clickhouse"
	"github.com/ethpandaops/xatu/pkg/consumoor/source"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validChGoConfig returns a ChGoConfig that passes validation.
func validChGoConfig() clickhouse.ChGoConfig {
	return clickhouse.ChGoConfig{
		QueryTimeout:        30 * time.Second,
		DialTimeout:         5 * time.Second,
		ReadTimeout:         30 * time.Second,
		MaxRetries:          3,
		RetryBaseDelay:      100 * time.Millisecond,
		RetryMaxDelay:       2 * time.Second,
		MaxConns:            32,
		MinConns:            1,
		ConnMaxLifetime:     time.Hour,
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckPeriod:   30 * time.Second,
		PoolMetricsInterval: 15 * time.Second,
	}
}

// validClickHouseConfig returns a clickhouse.Config that passes validation.
func validClickHouseConfig() *clickhouse.Config {
	return &clickhouse.Config{
		DSN:  "clickhouse://localhost:9000/default",
		ChGo: validChGoConfig(),
	}
}

// validKafkaConfig returns a KafkaConfig that passes validation.
func validKafkaConfig() *source.KafkaConfig {
	return &source.KafkaConfig{
		Brokers:          []string{"localhost:9092"},
		Topics:           []string{"^test-.+"},
		ConsumerGroup:    "test-group",
		Encoding:         "json",
		OffsetDefault:    "earliest",
		SessionTimeoutMs: 30000,
		RebalanceTimeout: 15 * time.Second,
		CommitInterval:   5 * time.Second,
		ShutdownTimeout:  30 * time.Second,
		MaxInFlight:      64,
	}
}

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
		cfg := validClickHouseConfig()
		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects invalid pool bounds", func(t *testing.T) {
		cfg := validClickHouseConfig()
		cfg.ChGo.MaxConns = 2
		cfg.ChGo.MinConns = 3

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minConns")
	})

	t.Run("rejects negative retries", func(t *testing.T) {
		cfg := validClickHouseConfig()
		cfg.ChGo.MaxRetries = -1

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxRetries")
	})

	t.Run("rejects unsupported insertSettings value type", func(t *testing.T) {
		cfg := validClickHouseConfig()
		cfg.Defaults.InsertSettings = map[string]any{
			"bad_setting": []int{1, 2},
		}

		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported value type")
	})
}

func TestTableConfigForMergesInsertSettings(t *testing.T) {
	cfg := &clickhouse.Config{
		Defaults: clickhouse.TableConfig{
			InsertSettings: map[string]any{
				"insert_quorum":         2,
				"insert_quorum_timeout": 30000,
			},
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

	assert.Equal(
		t,
		map[string]any{
			"insert_quorum":         3,
			"insert_quorum_timeout": 30000,
		},
		got.InsertSettings,
	)
}

func TestKafkaConfigValidateOffsetDefault(t *testing.T) {
	t.Run("accepts earliest", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.OffsetDefault = "earliest"
		require.NoError(t, cfg.Validate())
	})

	t.Run("accepts latest", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.OffsetDefault = "latest"
		require.NoError(t, cfg.Validate())
	})

	t.Run("rejects oldest", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.OffsetDefault = "oldest"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "offsetDefault")
	})

	t.Run("rejects empty string", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.OffsetDefault = ""
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "offsetDefault")
	})
}

func TestSASLConfigValidateMechanism(t *testing.T) {
	validSASL := func(mechanism string) *source.KafkaConfig {
		cfg := validKafkaConfig()
		cfg.SASLConfig = &source.SASLConfig{
			Mechanism: mechanism,
			User:      "alice",
			Password:  "secret",
		}

		return cfg
	}

	for _, mech := range []string{
		"PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512", "OAUTHBEARER",
		"plain", "scram-sha-256", " PLAIN ",
	} {
		t.Run("accepts "+mech, func(t *testing.T) {
			require.NoError(t, validSASL(mech).Validate())
		})
	}

	t.Run("accepts empty mechanism (defaults to PLAIN)", func(t *testing.T) {
		require.NoError(t, validSASL("").Validate())
	})

	for _, mech := range []string{"AWS_MSK_IAM", "GSSAPI", "nonsense"} {
		t.Run("rejects "+mech, func(t *testing.T) {
			err := validSASL(mech).Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported mechanism")
			assert.Contains(t, err.Error(), mech)
		})
	}
}

func TestKafkaConfigValidateSessionTimeout(t *testing.T) {
	t.Run("rejects zero", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.SessionTimeoutMs = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sessionTimeoutMs")
	})

	t.Run("accepts positive", func(t *testing.T) {
		cfg := validKafkaConfig()
		cfg.SessionTimeoutMs = 30000
		require.NoError(t, cfg.Validate())
	})
}

func TestTableConfigForAppliesCanonicalDefaults(t *testing.T) {
	t.Run("adds auto quorum for canonical tables when unset", func(t *testing.T) {
		cfg := &clickhouse.Config{}

		got := cfg.TableConfigFor("canonical_beacon_block")
		assert.Equal(t, map[string]any{"insert_quorum": "auto"}, got.InsertSettings)
	})

	t.Run("does not add quorum for non-canonical tables", func(t *testing.T) {
		cfg := &clickhouse.Config{}

		got := cfg.TableConfigFor("beacon_api_eth_v1_events_head")
		assert.Nil(t, got.InsertSettings)
	})

	t.Run("preserves explicit per-table quorum", func(t *testing.T) {
		cfg := &clickhouse.Config{
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
				InsertSettings: map[string]any{
					"insert_quorum": 2,
				},
			},
		}

		got := cfg.TableConfigFor("canonical_beacon_block")
		assert.Equal(t, map[string]any{"insert_quorum": 2}, got.InsertSettings)
	})
}
