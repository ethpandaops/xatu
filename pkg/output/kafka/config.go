package kafka

import (
	"errors"
	"time"
)

// Config is the top-level configuration for the kafka output sink.
// It embeds ProducerConfig for Sarama producer settings and adds
// sink-specific fields like Topic and batching parameters.
type Config struct {
	ProducerConfig     `yaml:",inline"`
	Topic              string        `yaml:"topic"`
	MaxQueueSize       int           `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout       time.Duration `yaml:"batchTimeout" default:"5s"`
	MaxExportBatchSize int           `yaml:"maxExportBatchSize" default:"512"`
	Workers            int           `yaml:"workers" default:"5"`
}

// Validate checks the Config for correctness.
func (c *Config) Validate() error {
	if err := c.ProducerConfig.Validate(); err != nil {
		return err
	}

	if c.Topic == "" {
		return errors.New("topic is required")
	}

	return nil
}
