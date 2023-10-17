package kafka

import (
	"errors"
	"time"
)

type Config struct {
	Brokers        string              `yaml:"brokers"`
	Topic          string              `yaml:"topic"`
	TLS            bool                `yaml:"tls" default:"false"`
	MaxQueueSize   int                 `yaml:"maxQueueSize" default:"51200"`
	FlushFrequency time.Duration       `yaml:"flushFrequency" default:"10s"`
	FlushMessages  int                 `yaml:"flushMessages" default:"500"`
	FlushBytes     int                 `yaml:"flushBytes" default:"1000000"`
	MaxRetries     int                 `yaml:"maxRetries" default:"3"`
	Compression    CompressionStrategy `yaml:"compression" default:"snappy"`
	RequiredAcks   RequiredAcks        `yaml:"requiredAcks" default:"leader"`
	Partitioning   PartitionStrategy   `yaml:"partitioning" default:"none"`
}

func (c *Config) Validate() error {

	if c.Brokers == "" {
		return errors.New("brokers is required")
	}
	if c.Topic == "" {
		return errors.New("topic is required")
	}

	return nil
}
