package kafkatopicrouter

import (
	"errors"
	"time"

	"github.com/ethpandaops/xatu/pkg/output/kafka"
)

// Config is the configuration for the kafkaTopicRouter output sink.
type Config struct {
	kafka.ProducerConfig `yaml:",inline"`
	TopicPattern         string        `yaml:"topicPattern"`
	MaxQueueSize         int           `yaml:"maxQueueSize" default:"51200"`
	BatchTimeout         time.Duration `yaml:"batchTimeout" default:"5s"`
	MaxExportBatchSize   int           `yaml:"maxExportBatchSize" default:"512"`
	Workers              int           `yaml:"workers" default:"5"`
}

// Validate checks the Config for correctness.
func (c *Config) Validate() error {
	if err := c.ProducerConfig.Validate(); err != nil {
		return err
	}

	if c.TopicPattern == "" {
		return errors.New("topicPattern is required")
	}

	return nil
}
