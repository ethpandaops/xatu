package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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
	producerConfig, err := Init(config)
	if err != nil {
		return nil, err
	}
	brokersList := strings.Split(config.Brokers, ",")

	return sarama.NewSyncProducer(brokersList, producerConfig)
}
func Init(config *Config) (*sarama.Config, error) {
	c := sarama.NewConfig()
	c.Producer.Flush.Bytes = config.FlushBytes
	c.Producer.Flush.Messages = config.FlushMessages
	c.Producer.Flush.Frequency = config.FlushFrequency
	c.Producer.Retry.Max = config.MaxRetries
	c.Producer.Return.Successes = true

	if config.TLSClientConfig.CertificatePath != "" {
		clientCertificate, err := tls.LoadX509KeyPair(config.TLSClientConfig.CertificatePath, config.TLSClientConfig.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read client certificate: %w", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{clientCertificate},
			MinVersion:   tls.VersionTLS12,
		}

		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(config.TLSClientConfig.CACertificate))
		tlsConfig.RootCAs = certPool

		c.Net.TLS.Enable = true
		c.Net.TLS.Config = tlsConfig
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
