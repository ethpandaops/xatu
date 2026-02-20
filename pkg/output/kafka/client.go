package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// CompressionStrategy defines the compression codec for Kafka messages.
type CompressionStrategy string

var (
	CompressionStrategyNone   CompressionStrategy = "none"
	CompressionStrategyGZIP   CompressionStrategy = "gzip"
	CompressionStrategySnappy CompressionStrategy = "snappy"
	CompressionStrategyLZ4    CompressionStrategy = "lz4"
	CompressionStrategyZSTD   CompressionStrategy = "zstd"
)

// RequiredAcks defines the acknowledgment level for produced messages.
type RequiredAcks string

var (
	RequiredAcksLeader RequiredAcks = "leader"
	RequiredAcksAll    RequiredAcks = "all"
	RequiredAcksNone   RequiredAcks = "none"
)

// PartitionStrategy defines the partitioning strategy for produced messages.
type PartitionStrategy string

var (
	PartitionStrategyNone   PartitionStrategy = "none"
	PartitionStrategyRandom PartitionStrategy = "random"
)

// SASLMechanism defines the SASL authentication mechanism.
type SASLMechanism string

var (
	SASLTypeOAuth       SASLMechanism = "OAUTHBEARER"
	SASLTypePlaintext   SASLMechanism = "PLAIN"
	SASLTypeSCRAMSHA256 SASLMechanism = "SCRAM-SHA-256"
	SASLTypeSCRAMSHA512 SASLMechanism = "SCRAM-SHA-512"
	SASLTypeGSSAPI      SASLMechanism = "GSSAPI"
)

// EncodingType defines the serialization format for Kafka message values.
type EncodingType string

var (
	EncodingTypeJSON     EncodingType = "json"
	EncodingTypeProtobuf EncodingType = "protobuf"
)

// ProducerConfig contains the fields needed to build a Sarama producer.
// It is shared by all Kafka-based sinks.
type ProducerConfig struct {
	Brokers         string              `yaml:"brokers"`
	TLS             bool                `yaml:"tls" default:"false"`
	TLSClientConfig *TLSClientConfig    `yaml:"tlsClientConfig"`
	SASLConfig      *SASLConfig         `yaml:"sasl"`
	FlushFrequency  time.Duration       `yaml:"flushFrequency" default:"10s"`
	FlushMessages   int                 `yaml:"flushMessages" default:"500"`
	FlushBytes      int                 `yaml:"flushBytes" default:"1000000"`
	MaxRetries      int                 `yaml:"maxRetries" default:"3"`
	Compression     CompressionStrategy `yaml:"compression" default:"none"`
	RequiredAcks    RequiredAcks        `yaml:"requiredAcks" default:"leader"`
	Partitioning    PartitionStrategy   `yaml:"partitioning" default:"none"`
	Encoding        EncodingType        `yaml:"encoding" default:"json"`
	Version         string              `yaml:"version"`
}

// Validate checks the ProducerConfig for correctness.
func (c *ProducerConfig) Validate() error {
	if c.Brokers == "" {
		return errors.New("brokers is required")
	}

	if c.TLSClientConfig != nil && c.SASLConfig != nil {
		return errors.New(
			"only one of 'tlsClientConfig' and 'sasl' can be specified",
		)
	}

	if err := c.TLSClientConfig.Validate(); err != nil {
		return err
	}

	if err := c.SASLConfig.Validate(); err != nil {
		return err
	}

	return nil
}

// MarshalEvent serializes a proto message using the configured encoding.
func (c *ProducerConfig) MarshalEvent(m proto.Message) ([]byte, error) {
	switch c.Encoding {
	case EncodingTypeProtobuf:
		return proto.Marshal(m)
	default:
		return protojson.Marshal(m)
	}
}

// TLSClientConfig holds mTLS client certificate configuration.
type TLSClientConfig struct {
	CertificatePath string `yaml:"certificatePath"`
	KeyPath         string `yaml:"keyPath"`
	CACertificate   string `yaml:"caCertificate"`
}

// Validate checks the TLSClientConfig for correctness.
func (c *TLSClientConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.CertificatePath != "" && c.KeyPath == "" {
		return errors.New("client key is required")
	}

	return nil
}

// SASLConfig holds SASL authentication configuration.
type SASLConfig struct {
	Mechanism    SASLMechanism `yaml:"mechanism" default:"PLAIN"`
	Version      int16         `yaml:"version" default:"1"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	PasswordFile string        `yaml:"passwordFile"`
}

// Validate checks the SASLConfig for correctness.
func (c *SASLConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.User == "" {
		return errors.New("'user' is required")
	}

	if c.Password != "" && c.PasswordFile != "" {
		return errors.New(
			"either 'password' or 'passwordFile' can be specified",
		)
	}

	if c.Password == "" && c.PasswordFile == "" {
		return errors.New("'password' or 'passwordFile' is required")
	}

	return nil
}

// NewSyncProducer creates a new Sarama SyncProducer from ProducerConfig.
func NewSyncProducer(config *ProducerConfig) (sarama.SyncProducer, error) {
	saramaConfig, err := InitSaramaConfig(config)
	if err != nil {
		return nil, err
	}

	brokersList := strings.Split(config.Brokers, ",")

	return sarama.NewSyncProducer(brokersList, saramaConfig)
}

// InitSaramaConfig builds a *sarama.Config from ProducerConfig.
func InitSaramaConfig(config *ProducerConfig) (*sarama.Config, error) {
	c := sarama.NewConfig()
	c.Producer.Flush.Bytes = config.FlushBytes
	c.Producer.Flush.Messages = config.FlushMessages
	c.Producer.Flush.Frequency = config.FlushFrequency
	c.Producer.Retry.Max = config.MaxRetries
	c.Producer.Return.Successes = true
	c.Net.TLS.Enable = config.TLS
	c.Metadata.Full = false

	if config.Version != "" {
		version, err := sarama.ParseKafkaVersion(config.Version)
		if err != nil {
			return nil, err
		}

		c.Version = version
	}

	if config.TLSClientConfig != nil {
		clientCertificate, err := tls.LoadX509KeyPair(
			config.TLSClientConfig.CertificatePath,
			config.TLSClientConfig.KeyPath,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to read client certificate: %w", err,
			)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{clientCertificate},
			MinVersion:   tls.VersionTLS12,
		}

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(
			[]byte(config.TLSClientConfig.CACertificate),
		)
		tlsConfig.RootCAs = certPool

		c.Net.TLS.Enable = true
		c.Net.TLS.Config = tlsConfig
	}

	if config.SASLConfig != nil {
		var password string
		if config.SASLConfig.Password != "" {
			password = config.SASLConfig.Password
		} else if config.SASLConfig.PasswordFile != "" {
			passwordFile, err := os.ReadFile(config.SASLConfig.PasswordFile)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to read client password: %w", err,
				)
			}

			password = strings.TrimSpace(string(passwordFile))
		}

		c.Net.SASL.Enable = true
		c.Net.SASL.Version = config.SASLConfig.Version
		c.Net.SASL.User = config.SASLConfig.User
		c.Net.SASL.Password = password

		switch config.SASLConfig.Mechanism {
		case SASLTypeOAuth:
			c.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		case SASLTypeSCRAMSHA256:
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case SASLTypeSCRAMSHA512:
			c.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		case SASLTypeGSSAPI:
			c.Net.SASL.Mechanism = sarama.SASLTypeGSSAPI
		default:
			c.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}

	switch config.RequiredAcks {
	case RequiredAcksNone:
		c.Producer.RequiredAcks = sarama.NoResponse
	case RequiredAcksAll:
		c.Producer.RequiredAcks = sarama.WaitForAll
	default:
		c.Producer.RequiredAcks = sarama.WaitForLocal
	}

	switch config.Compression {
	case CompressionStrategyLZ4:
		c.Producer.Compression = sarama.CompressionLZ4
	case CompressionStrategyGZIP:
		c.Producer.Compression = sarama.CompressionGZIP
	case CompressionStrategySnappy:
		c.Producer.Compression = sarama.CompressionSnappy
	case CompressionStrategyZSTD:
		c.Producer.Compression = sarama.CompressionZSTD
	default:
		c.Producer.Compression = sarama.CompressionNone
	}

	switch config.Partitioning {
	case PartitionStrategyNone:
		c.Producer.Partitioner = sarama.NewHashPartitioner
	default:
		c.Producer.Partitioner = sarama.NewRandomPartitioner
	}

	return c, nil
}
