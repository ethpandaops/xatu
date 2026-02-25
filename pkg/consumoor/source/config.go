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
	MaxPartitionFetchBytes int32 `yaml:"maxPartitionFetchBytes" default:"10485760"`

	// SessionTimeoutMs is the consumer group session timeout.
	SessionTimeoutMs int `yaml:"sessionTimeoutMs" default:"30000"`

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
