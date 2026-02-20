package clickhouse

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Config configures the ClickHouse writer.
type Config struct {
	// DSN is the ClickHouse connection string.
	DSN string `yaml:"dsn"`

	// TableSuffix is appended to every table name before writing.
	// For example, set to "_local" to bypass Distributed tables and write
	// directly to ReplicatedMergeTree tables in a clustered setup.
	TableSuffix string `yaml:"tableSuffix"`

	// Defaults are the default batch settings for all tables.
	Defaults TableConfig `yaml:"defaults"`

	// Tables contains per-table overrides for batch settings.
	Tables map[string]TableConfig `yaml:"tables"`

	// ChGo configures ch-go backend retry/pooling behavior.
	ChGo ChGoConfig `yaml:"chgo"`
}

// TableConfig holds batching parameters for a ClickHouse table.
type TableConfig struct {
	// BatchSize is the maximum number of rows per batch insert.
	BatchSize int `yaml:"batchSize" default:"200000"`
	// FlushInterval is the maximum time between flushes.
	FlushInterval time.Duration `yaml:"flushInterval" default:"1s"`
	// BufferSize is the channel buffer capacity for pending rows.
	BufferSize int `yaml:"bufferSize" default:"200000"`
	// InsertSettings appends ClickHouse SETTINGS to INSERT statements.
	// Canonical tables (name prefix "canonical_") default to
	// insert_quorum=auto unless explicitly overridden.
	// Example:
	// insertSettings:
	//   insert_quorum: 2
	//   insert_quorum_timeout: 60000
	InsertSettings map[string]any `yaml:"insertSettings"`
}

// ChGoConfig configures the ch-go backend query retries and connection pooling.
type ChGoConfig struct {
	// QueryTimeout is the per-attempt timeout for ch-go operations.
	// Set to 0 to disable timeout wrapping.
	QueryTimeout time.Duration `yaml:"queryTimeout" default:"30s"`

	// MaxRetries is the number of retry attempts after the initial try.
	MaxRetries int `yaml:"maxRetries" default:"3"`
	// RetryBaseDelay is the initial delay before retry attempt 1.
	RetryBaseDelay time.Duration `yaml:"retryBaseDelay" default:"100ms"`
	// RetryMaxDelay caps exponential retry backoff.
	RetryMaxDelay time.Duration `yaml:"retryMaxDelay" default:"2s"`

	// MaxConns is the maximum number of pooled ClickHouse connections.
	MaxConns int32 `yaml:"maxConns" default:"8"`
	// MinConns is the minimum number of pooled ClickHouse connections.
	MinConns int32 `yaml:"minConns" default:"1"`
	// ConnMaxLifetime is the maximum lifetime for pooled connections.
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" default:"1h"`
	// ConnMaxIdleTime is the maximum idle time for pooled connections.
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" default:"10m"`
	// HealthCheckPeriod is the interval for pool health checks.
	HealthCheckPeriod time.Duration `yaml:"healthCheckPeriod" default:"30s"`

	// PoolMetricsInterval controls how often pool stats are sampled.
	// Set to 0 to disable pool metrics collection.
	PoolMetricsInterval time.Duration `yaml:"poolMetricsInterval" default:"15s"`
}

// Validate checks the ClickHouse configuration for errors.
func (c *Config) Validate() error {
	if c.DSN == "" {
		return errors.New("clickhouse: dsn is required")
	}

	if err := c.ChGo.Validate(); err != nil {
		return err
	}

	if c.Defaults.BatchSize <= 0 {
		return errors.New("clickhouse.defaults: batchSize must be > 0")
	}

	if c.Defaults.FlushInterval <= 0 {
		return errors.New("clickhouse.defaults: flushInterval must be > 0")
	}

	if c.Defaults.BufferSize <= 0 {
		return errors.New("clickhouse.defaults: bufferSize must be > 0")
	}

	if err := validateInsertSettings(c.Defaults.InsertSettings, "clickhouse.defaults.insertSettings"); err != nil {
		return err
	}

	for table, override := range c.Tables {
		if override.BatchSize < 0 {
			return fmt.Errorf("clickhouse.tables.%s: batchSize must be >= 0", table)
		}

		if override.FlushInterval < 0 {
			return fmt.Errorf("clickhouse.tables.%s: flushInterval must be >= 0", table)
		}

		if override.BufferSize < 0 {
			return fmt.Errorf("clickhouse.tables.%s: bufferSize must be >= 0", table)
		}

		path := fmt.Sprintf("clickhouse.tables.%s.insertSettings", table)
		if err := validateInsertSettings(override.InsertSettings, path); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks the ch-go backend configuration for errors.
func (c *ChGoConfig) Validate() error {
	if c.QueryTimeout < 0 {
		return errors.New("clickhouse.chgo: queryTimeout must be >= 0")
	}

	if c.MaxRetries < 0 {
		return errors.New("clickhouse.chgo: maxRetries must be >= 0")
	}

	if c.RetryBaseDelay <= 0 {
		return errors.New("clickhouse.chgo: retryBaseDelay must be > 0")
	}

	if c.RetryMaxDelay <= 0 {
		return errors.New("clickhouse.chgo: retryMaxDelay must be > 0")
	}

	if c.MaxConns <= 0 {
		return errors.New("clickhouse.chgo: maxConns must be > 0")
	}

	if c.MinConns < 0 {
		return errors.New("clickhouse.chgo: minConns must be >= 0")
	}

	if c.MinConns > c.MaxConns {
		return errors.New("clickhouse.chgo: minConns must be <= maxConns")
	}

	if c.ConnMaxLifetime <= 0 {
		return errors.New("clickhouse.chgo: connMaxLifetime must be > 0")
	}

	if c.ConnMaxIdleTime <= 0 {
		return errors.New("clickhouse.chgo: connMaxIdleTime must be > 0")
	}

	if c.HealthCheckPeriod <= 0 {
		return errors.New("clickhouse.chgo: healthCheckPeriod must be > 0")
	}

	if c.PoolMetricsInterval < 0 {
		return errors.New("clickhouse.chgo: poolMetricsInterval must be >= 0")
	}

	return nil
}

// TableConfigFor returns the merged table config for a given table name,
// using per-table overrides on top of defaults.
func (c *Config) TableConfigFor(table string) TableConfig {
	cfg := c.Defaults
	cfg.InsertSettings = cloneInsertSettings(c.Defaults.InsertSettings)

	override, ok := c.Tables[table]
	if !ok {
		applyCanonicalTableDefaults(table, &cfg)

		return cfg
	}

	if override.BatchSize > 0 {
		cfg.BatchSize = override.BatchSize
	}

	if override.FlushInterval > 0 {
		cfg.FlushInterval = override.FlushInterval
	}

	if override.BufferSize > 0 {
		cfg.BufferSize = override.BufferSize
	}

	if len(override.InsertSettings) > 0 {
		if cfg.InsertSettings == nil {
			cfg.InsertSettings = make(map[string]any, len(override.InsertSettings))
		}

		for k, v := range override.InsertSettings {
			cfg.InsertSettings[k] = v
		}
	}

	applyCanonicalTableDefaults(table, &cfg)

	return cfg
}

func applyCanonicalTableDefaults(table string, cfg *TableConfig) {
	if !strings.HasPrefix(table, "canonical_") {
		return
	}

	if cfg.InsertSettings == nil {
		cfg.InsertSettings = make(map[string]any, 1)
	}

	if _, exists := cfg.InsertSettings["insert_quorum"]; !exists {
		cfg.InsertSettings["insert_quorum"] = "auto"
	}
}

func cloneInsertSettings(settings map[string]any) map[string]any {
	if len(settings) == 0 {
		return nil
	}

	out := make(map[string]any, len(settings))
	for k, v := range settings {
		out[k] = v
	}

	return out
}

func validateInsertSettings(settings map[string]any, path string) error {
	for name, value := range settings {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%s: setting name cannot be empty", path)
		}

		switch value.(type) {
		case string,
			bool,
			int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			// supported
		default:
			return fmt.Errorf("%s.%s: unsupported value type %T", path, name, value)
		}
	}

	return nil
}
