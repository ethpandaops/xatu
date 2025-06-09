package kafka

import (
	"errors"
	"time"
)

type Config struct {
	Brokers            string              `yaml:"brokers"`
	Topic              string              `yaml:"topic"`
	TLS                bool                `yaml:"tls" default:"false"`
	TLSClientConfig    *TLSClientConfig    `yaml:"tlsClientConfig"`
  SASLConfig         *SASLConfig         `yaml:"sasl"`
	MaxQueueSize       int                 `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout       time.Duration       `yaml:"batchTimeout" default:"5s"`
	MaxExportBatchSize int                 `yaml:"maxExportBatchSize" default:"512"`
	Workers            int                 `yaml:"workers" default:"5"`
	FlushFrequency     time.Duration       `yaml:"flushFrequency" default:"10s"`
	FlushMessages      int                 `yaml:"flushMessages" default:"500"`
	FlushBytes         int                 `yaml:"flushBytes" default:"1000000"`
	MaxRetries         int                 `yaml:"maxRetries" default:"3"`
	Compression        CompressionStrategy `yaml:"compression" default:"none"`
	RequiredAcks       RequiredAcks        `yaml:"requiredAcks" default:"leader"`
	Partitioning       PartitionStrategy   `yaml:"partitioning" default:"none"`
  Version            string              `yaml:"version"`
}

type TLSClientConfig struct {
	CertificatePath string `yaml:"certificatePath"`
	KeyPath         string `yaml:"keyPath"`
	CACertificate   string `yaml:"caCertificate"`
}

type SASLConfig struct {
	Mechanism    SASLMechanism `yaml:"mechanism" default:"PLAIN"`
	Version      int16         `yaml:"version" default:"1"`
	User         string        `yaml:"user"`
	Password     string        `yaml:"password"`
	PasswordFile string        `yaml:"passwordFile"`
}

func (c *Config) Validate() error {
	if c.Brokers == "" {
		return errors.New("brokers is required")
	}

	if c.Topic == "" {
		return errors.New("topic is required")
	}

	if c.TLSClientConfig != nil && c.SASLConfig != nil {
		return errors.New("only one of 'tlsClientConfig' and 'sasl' can be specified")
	}

	if err := c.TLSClientConfig.Validate(); err != nil {
		return err
	}

	if err := c.SASLConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *TLSClientConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.CertificatePath != "" && c.KeyPath == "" {
		return errors.New("client key is required")
	}

	return nil
}

func (c *SASLConfig) Validate() error {
	if c == nil {
		return nil
	}

	if c.User == "" {
		return errors.New("'user' is required")
	}

	if c.Password != "" && c.PasswordFile != "" {
		return errors.New("either 'password' or 'passwordFile' can be specified")
	}

	if c.Password == "" && c.PasswordFile == "" {
		return errors.New("'password' or 'passwordFile' is required")
	}

	return nil
}
