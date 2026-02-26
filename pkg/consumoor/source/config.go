package source

import (
	"errors"
	"fmt"
	"strings"
	"time"

	xtls "github.com/ethpandaops/xatu/pkg/consumoor/tls"
)

// Supported SASL mechanism values.
const (
	SASLMechanismPLAIN       = "PLAIN"
	SASLMechanismSCRAMSHA256 = "SCRAM-SHA-256"
	SASLMechanismSCRAMSHA512 = "SCRAM-SHA-512"
	SASLMechanismOAUTHBEARER = "OAUTHBEARER"
)

// supportedSASLMechanisms is the set of SASL mechanisms accepted by Validate.
// An empty mechanism string is also valid and defaults to PLAIN.
var supportedSASLMechanisms = map[string]struct{}{
	SASLMechanismPLAIN:       {},
	SASLMechanismSCRAMSHA256: {},
	SASLMechanismSCRAMSHA512: {},
	SASLMechanismOAUTHBEARER: {},
}

// TopicOverride holds per-topic batch settings that override KafkaConfig defaults.
// Nil pointer fields inherit the global default.
type TopicOverride struct {
	OutputBatchCount  *int           `yaml:"outputBatchCount"`
	OutputBatchPeriod *time.Duration `yaml:"outputBatchPeriod"`
	MaxInFlight       *int           `yaml:"maxInFlight"`
}

// Validate checks the per-topic override for errors.
func (o *TopicOverride) Validate(topic string) error {
	if o.OutputBatchCount != nil && *o.OutputBatchCount < 0 {
		return fmt.Errorf("kafka.topicOverrides.%s: outputBatchCount must be >= 0", topic)
	}

	if o.OutputBatchPeriod != nil && *o.OutputBatchPeriod < 0 {
		return fmt.Errorf("kafka.topicOverrides.%s: outputBatchPeriod must be >= 0", topic)
	}

	if o.MaxInFlight != nil && *o.MaxInFlight < 1 {
		return fmt.Errorf("kafka.topicOverrides.%s: maxInFlight must be >= 1", topic)
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

	// TLS configures TLS for the Kafka connection.
	TLS xtls.Config `yaml:"tls"`
	// SASLConfig is the SASL authentication configuration.
	SASLConfig *SASLConfig `yaml:"sasl"`

	// FetchMinBytes is the minimum number of bytes to fetch per request.
	FetchMinBytes int32 `yaml:"fetchMinBytes" default:"1"`
	// FetchWaitMaxMs is the maximum time to wait for fetch responses.
	FetchWaitMaxMs int `yaml:"fetchWaitMaxMs" default:"250"`
	// MaxPartitionFetchBytes is the max bytes per partition per request.
	MaxPartitionFetchBytes int32 `yaml:"maxPartitionFetchBytes" default:"3145728"`
	// FetchMaxBytes is the max total bytes per fetch request across all
	// partitions from a single broker. With many independent consumers
	// (one per topic) this is the primary lever for capping in-flight
	// memory from Kafka fetch buffers. Default: 10 MiB.
	FetchMaxBytes int32 `yaml:"fetchMaxBytes" default:"10485760"`

	// SessionTimeoutMs is the consumer group session timeout.
	SessionTimeoutMs int `yaml:"sessionTimeoutMs" default:"30000"`
	// RebalanceTimeout is the maximum time group members are allowed to
	// take when a rebalance has begun (finish work, commit offsets, rejoin).
	// Lower values speed up partition reassignment when scaling. Default: 15s.
	RebalanceTimeout time.Duration `yaml:"rebalanceTimeout" default:"15s"`

	// OffsetDefault controls where to start consuming when no offset exists.
	// Valid values: "earliest" or "latest".
	OffsetDefault string `yaml:"offsetDefault" default:"earliest"`

	// CommitInterval controls Kafka offset commit cadence for kafka_franz.
	CommitInterval time.Duration `yaml:"commitInterval" default:"5s"`
	// ShutdownTimeout is the maximum time the Benthos stream waits for
	// in-flight messages to complete during graceful shutdown.
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" default:"30s"`
	// RejectedTopic is an optional Kafka topic where permanently rejected
	// messages are emitted as JSON envelopes.
	RejectedTopic string `yaml:"rejectedTopic"`

	// TopicRefreshInterval controls how often Kafka metadata is refreshed to
	// discover new topics matching the configured regex patterns. Defaults to
	// 60s. Set to 0 to disable periodic refresh (startup-only discovery).
	TopicRefreshInterval time.Duration `yaml:"topicRefreshInterval" default:"60s"`
	// LagPollInterval controls how often consumer lag is polled from Kafka.
	// Set to 0 to disable lag monitoring. Default: 30s.
	LagPollInterval time.Duration `yaml:"lagPollInterval" default:"30s"`
	// ConnectTimeout is the maximum time a TCP dial to a broker will wait
	// for a connection to complete. A reasonable value (e.g. 10s) prevents
	// hung dials from generating noisy warnings when some brokers are
	// temporarily unreachable. Default: 10s. Set to 0 to disable.
	ConnectTimeout time.Duration `yaml:"connectTimeout" default:"10s"`

	// OutputBatchCount is the number of messages Benthos accumulates before
	// calling WriteBatch on the output plugin. Higher values increase INSERT
	// throughput by writing more rows per ClickHouse INSERT. Set to 0 to
	// disable count-based batching. Default: 10000.
	OutputBatchCount int `yaml:"outputBatchCount" default:"10000"`
	// OutputBatchPeriod is the maximum time Benthos waits to fill a batch
	// before flushing a partial batch. Ensures low-volume topics still make
	// progress. Default: 5s. Set to 0 to disable period-based flushing.
	OutputBatchPeriod time.Duration `yaml:"outputBatchPeriod" default:"5s"`

	// MaxInFlight is the maximum number of concurrent WriteBatch calls
	// Benthos makes for each stream's output. Higher values increase
	// throughput by allowing concurrent ClickHouse INSERTs and bigger
	// natural batches. Default: 8.
	MaxInFlight int `yaml:"maxInFlight" default:"64"`

	// TopicOverrides contains per-topic batch settings keyed by exact topic name.
	// Overrides are matched against discovered concrete topic names. Unset fields
	// inherit the global defaults from this KafkaConfig.
	TopicOverrides map[string]TopicOverride `yaml:"topicOverrides"`
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

	if c.OffsetDefault != "earliest" && c.OffsetDefault != "latest" {
		return errors.New("kafka: offsetDefault must be 'earliest' or 'latest'")
	}

	if c.SessionTimeoutMs <= 0 {
		return errors.New("kafka: sessionTimeoutMs must be > 0")
	}

	if c.RebalanceTimeout < 100*time.Millisecond {
		return errors.New("kafka: rebalanceTimeout must be >= 100ms")
	}

	if c.CommitInterval <= 0 {
		return errors.New("kafka: commitInterval must be positive")
	}

	if c.ShutdownTimeout <= 0 {
		return errors.New("kafka: shutdownTimeout must be > 0")
	}

	if c.TopicRefreshInterval < 0 {
		return errors.New(
			"kafka: topicRefreshInterval must be >= 0",
		)
	}

	if c.OutputBatchCount < 0 {
		return errors.New("kafka: outputBatchCount must be >= 0")
	}

	if c.OutputBatchPeriod < 0 {
		return errors.New("kafka: outputBatchPeriod must be >= 0")
	}

	if c.MaxInFlight < 1 {
		return errors.New("kafka: maxInFlight must be >= 1")
	}

	for topic, override := range c.TopicOverrides {
		if err := override.Validate(topic); err != nil {
			return err
		}
	}

	if err := c.TLS.Validate(); err != nil {
		return fmt.Errorf("kafka.%w", err)
	}

	if c.SASLConfig != nil {
		if err := c.SASLConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ApplyTopicOverride returns a shallow copy with per-topic overrides merged in.
// Fields not set in the override keep the global default.
func (c *KafkaConfig) ApplyTopicOverride(topic string) KafkaConfig {
	out := *c

	override, ok := c.TopicOverrides[topic]
	if !ok {
		return out
	}

	if override.OutputBatchCount != nil {
		out.OutputBatchCount = *override.OutputBatchCount
	}

	if override.OutputBatchPeriod != nil {
		out.OutputBatchPeriod = *override.OutputBatchPeriod
	}

	if override.MaxInFlight != nil {
		out.MaxInFlight = *override.MaxInFlight
	}

	return out
}

// heartbeatIntervalMs derives the heartbeat interval from the session timeout.
// Kafka's standard practice is sessionTimeout / 10.
func (c *KafkaConfig) heartbeatIntervalMs() int {
	return c.SessionTimeoutMs / 10
}

// Validate checks the SASL configuration for errors.
func (c *SASLConfig) Validate() error {
	mechanism := strings.ToUpper(strings.TrimSpace(c.Mechanism))
	if mechanism != "" {
		if _, ok := supportedSASLMechanisms[mechanism]; !ok {
			return fmt.Errorf(
				"kafka.sasl: unsupported mechanism %q (supported: %s)",
				c.Mechanism,
				"PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, OAUTHBEARER",
			)
		}
	}

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
