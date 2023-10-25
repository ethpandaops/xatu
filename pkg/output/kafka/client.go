package kafka

import (
	"strings"

	"github.com/IBM/sarama"
)

type CompressionStrategy string

var (
	CompressionStrategyNone   CompressionStrategy = "none"
	CompressionStrategyGZIP   CompressionStrategy = "gzip"
	CompressionStrategySnappy CompressionStrategy = "snappy"
	CompressionStrategyLZ4    CompressionStrategy = "lz4"
	CompressionStrategyZSTD   CompressionStrategy = "zstd"
)

type RequiredAcks string

var (
	RequiredAcksLeader RequiredAcks = "leader"
	RequiredAcksAll    RequiredAcks = "all"
	RequiredAcksNone   RequiredAcks = "none"
)

type PartitionStrategy string

var (
	PartitionStrategyNone   PartitionStrategy = "none"
	PartitionStrategyRandom PartitionStrategy = "random"
)

func NewSyncProducer(config *Config) (sarama.SyncProducer, error) {
	producerConfig := Init(config)
	brokersList := strings.Split(config.Brokers, ",")

	return sarama.NewSyncProducer(brokersList, producerConfig)
}
func Init(config *Config) *sarama.Config {
	c := sarama.NewConfig()
	c.Producer.Flush.Bytes = config.FlushBytes
	c.Producer.Flush.Messages = config.FlushMessages
	c.Producer.Flush.Frequency = config.FlushFrequency
	c.Producer.Retry.Max = config.MaxRetries
	c.Producer.Return.Successes = true

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

	return c
}
