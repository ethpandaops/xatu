package consumoor

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Config is the configuration for the consumoor service.
type Config struct {
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging" default:"info"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metricsAddr" default:":9090"`
	// PProfAddr is the address to listen on for pprof.
	PProfAddr *string `yaml:"pprofAddr"`

	// Kafka is the Kafka consumer configuration.
	Kafka KafkaConfig `yaml:"kafka"`
	// ClickHouse is the ClickHouse writer configuration.
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`

	// DisabledEvents is a list of event names to drop without processing.
	DisabledEvents []string `yaml:"disabledEvents"`
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	if err := c.ClickHouse.Validate(); err != nil {
		return err
	}

	if _, err := c.DisabledEventEnums(); err != nil {
		return err
	}

	return nil
}

// DisabledEventEnums parses disabled event names into typed enum values.
// Unknown names are rejected to fail fast during startup.
func (c *Config) DisabledEventEnums() ([]xatu.Event_Name, error) {
	out := make([]xatu.Event_Name, 0, len(c.DisabledEvents))
	invalid := make([]string, 0)

	for _, name := range c.DisabledEvents {
		val, ok := xatu.Event_Name_value[name]
		if !ok {
			invalid = append(invalid, name)

			continue
		}

		out = append(out, xatu.Event_Name(val))
	}

	if len(invalid) > 0 {
		sort.Strings(invalid)

		return nil, fmt.Errorf("unknown disabledEvents: %s", strings.Join(invalid, ", "))
	}

	return out, nil
}

// KafkaConfig configures the Kafka consumer.
type KafkaConfig struct {
	// Brokers is a list of Kafka broker addresses.
	Brokers []string `yaml:"brokers"`
	// Topics is a list of topic patterns to subscribe to (supports regex).
	Topics []string `yaml:"topics"`
	// ConsumerGroup is the Kafka consumer group ID.
	ConsumerGroup string `yaml:"consumerGroup"`
	// Encoding is the message encoding format ("json" or "protobuf").
	Encoding string `yaml:"encoding" default:"json"`

	// TLS enables TLS for the Kafka connection.
	TLS bool `yaml:"tls" default:"false"`
	// SASLConfig is the SASL authentication configuration.
	SASLConfig *SASLConfig `yaml:"sasl"`

	// Version is the Kafka protocol version to use.
	Version string `yaml:"version"`

	// FetchMinBytes is the minimum number of bytes to fetch per request.
	FetchMinBytes int32 `yaml:"fetchMinBytes" default:"1"`
	// FetchWaitMaxMs is the maximum time to wait for fetch responses.
	FetchWaitMaxMs int `yaml:"fetchWaitMaxMs" default:"500"`
	// MaxPartitionFetchBytes is the max bytes per partition per request.
	MaxPartitionFetchBytes int32 `yaml:"maxPartitionFetchBytes" default:"10485760"`

	// SessionTimeoutMs is the consumer group session timeout.
	SessionTimeoutMs int `yaml:"sessionTimeoutMs" default:"30000"`
	// HeartbeatIntervalMs is the consumer group heartbeat interval.
	HeartbeatIntervalMs int `yaml:"heartbeatIntervalMs" default:"3000"`

	// OffsetDefault controls where to start consuming when no offset exists.
	// Valid values: "newest" or "oldest".
	OffsetDefault string `yaml:"offsetDefault" default:"oldest"`

	// CommitInterval is the maximum time between flush+commit cycles.
	CommitInterval time.Duration `yaml:"commitInterval" default:"5s"`
}

// SASLConfig configures SASL authentication for Kafka.
type SASLConfig struct {
	// Mechanism is the SASL mechanism to use.
	Mechanism string `yaml:"mechanism" default:"PLAIN"`
	// User is the SASL username.
	User string `yaml:"user"`
	// Password is the SASL password.
	Password string `yaml:"password"`
	// PasswordFile is the path to a file containing the SASL password.
	PasswordFile string `yaml:"passwordFile"`
}

// Validate checks the Kafka configuration for errors.
func (c *KafkaConfig) Validate() error {
	if len(c.Brokers) == 0 {
		return errors.New("kafka: at least one broker is required")
	}

	if len(c.Topics) == 0 {
		return errors.New("kafka: at least one topic pattern is required")
	}

	if c.ConsumerGroup == "" {
		return errors.New("kafka: consumerGroup is required")
	}

	if c.Encoding != "json" && c.Encoding != "protobuf" {
		return errors.New("kafka: encoding must be 'json' or 'protobuf'")
	}

	if c.OffsetDefault != "newest" && c.OffsetDefault != "oldest" {
		return errors.New("kafka: offsetDefault must be 'newest' or 'oldest'")
	}

	if c.CommitInterval <= 0 {
		return errors.New("kafka: commitInterval must be positive")
	}

	if c.SASLConfig != nil {
		if err := c.SASLConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks the SASL configuration for errors.
func (c *SASLConfig) Validate() error {
	if c.User == "" {
		return errors.New("kafka.sasl: user is required")
	}

	if c.Password == "" && c.PasswordFile == "" {
		return errors.New("kafka.sasl: password or passwordFile is required")
	}

	if c.Password != "" && c.PasswordFile != "" {
		return errors.New("kafka.sasl: only one of password or passwordFile can be set")
	}

	return nil
}

// ClickHouseConfig configures the ClickHouse writer.
type ClickHouseConfig struct {
	// DSN is the ClickHouse connection string.
	DSN string `yaml:"dsn"`

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
	// BatchBytes is the maximum byte size per batch insert.
	BatchBytes int `yaml:"batchBytes" default:"52428800"`
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
func (c *ClickHouseConfig) Validate() error {
	if c.DSN == "" {
		return errors.New("clickhouse: dsn is required")
	}

	if err := c.ChGo.Validate(); err != nil {
		return err
	}

	if c.Defaults.BatchSize <= 0 {
		return errors.New("clickhouse.defaults: batchSize must be > 0")
	}

	if c.Defaults.BatchBytes <= 0 {
		return errors.New("clickhouse.defaults: batchBytes must be > 0")
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

		if override.BatchBytes < 0 {
			return fmt.Errorf("clickhouse.tables.%s: batchBytes must be >= 0", table)
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
func (c *ClickHouseConfig) TableConfigFor(table string) TableConfig {
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

	if override.BatchBytes > 0 {
		cfg.BatchBytes = override.BatchBytes
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

// Override holds values that can be set via CLI flags or environment
// variables, overriding the config file.
type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
}

// ApplyOverrides applies CLI/env overrides to the configuration.
func (c *Config) ApplyOverrides(o *Override) {
	if o == nil {
		return
	}

	if o.MetricsAddr.Enabled {
		c.MetricsAddr = o.MetricsAddr.Value
	}
}
