package consumoor

import (
	"errors"
	"time"
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

	return nil
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
	// Backend selects the ClickHouse client implementation.
	// Valid values: "clickhouse-go" or "ch-go".
	Backend string `yaml:"backend" default:"clickhouse-go"`

	// DSN is the ClickHouse connection string.
	DSN string `yaml:"dsn"`

	// Defaults are the default batch settings for all tables.
	Defaults TableConfig `yaml:"defaults"`

	// Tables contains per-table overrides for batch settings.
	Tables map[string]TableConfig `yaml:"tables"`
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
}

// Validate checks the ClickHouse configuration for errors.
func (c *ClickHouseConfig) Validate() error {
	if c.Backend != "clickhouse-go" && c.Backend != "ch-go" {
		return errors.New("clickhouse: backend must be 'clickhouse-go' or 'ch-go'")
	}

	if c.DSN == "" {
		return errors.New("clickhouse: dsn is required")
	}

	return nil
}

// TableConfigFor returns the merged table config for a given table name,
// using per-table overrides on top of defaults.
func (c *ClickHouseConfig) TableConfigFor(table string) TableConfig {
	cfg := c.Defaults

	override, ok := c.Tables[table]
	if !ok {
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

	return cfg
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
