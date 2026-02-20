package source

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type rejectedRecord struct {
	Reason  string
	Err     string
	Payload []byte

	EventName string
	Kafka     kafkaMessageMetadata
}

type rejectSink interface {
	Write(ctx context.Context, record *rejectedRecord) error
	Close() error
	Enabled() bool
}

type noopRejectSink struct{}

func (noopRejectSink) Write(_ context.Context, _ *rejectedRecord) error {
	return nil
}

func (noopRejectSink) Close() error {
	return nil
}

func (noopRejectSink) Enabled() bool {
	return false
}

type kafkaRejectSink struct {
	topic string
	cl    *kgo.Client
}

func (s *kafkaRejectSink) Enabled() bool {
	return true
}

func (s *kafkaRejectSink) Close() error {
	if s.cl != nil {
		s.cl.Close()
	}

	return nil
}

func (s *kafkaRejectSink) Write(ctx context.Context, record *rejectedRecord) error {
	if record == nil {
		return errors.New("nil rejected record")
	}

	if s.cl == nil {
		return errors.New("dlq client is nil")
	}

	value, err := json.Marshal(map[string]any{
		"rejected_at":         time.Now().UTC().Format(time.RFC3339Nano),
		"reason":              record.Reason,
		"error":               record.Err,
		"event_name":          record.EventName,
		"source_topic":        record.Kafka.Topic,
		"source_partition":    record.Kafka.Partition,
		"source_offset":       record.Kafka.Offset,
		"payload_base64":      base64.StdEncoding.EncodeToString(record.Payload),
		"payload_bytes_count": len(record.Payload),
	})
	if err != nil {
		return fmt.Errorf("marshalling dlq envelope: %w", err)
	}

	key := fmt.Sprintf("%s:%d:%d", record.Kafka.Topic, record.Kafka.Partition, record.Kafka.Offset)
	kr := &kgo.Record{
		Topic: s.topic,
		Key:   []byte(key),
		Value: value,
	}

	if produceErr := s.cl.ProduceSync(ctx, kr).FirstErr(); produceErr != nil {
		return fmt.Errorf("producing dlq record: %w", produceErr)
	}

	return nil
}

func newRejectSink(cfg *KafkaConfig) (rejectSink, error) {
	if cfg == nil {
		return noopRejectSink{}, nil
	}

	if strings.TrimSpace(cfg.RejectedTopic) == "" {
		return noopRejectSink{}, nil
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.MaxBufferedRecords(256),
	}

	if cfg.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}))
	}

	if cfg.SASLConfig != nil {
		mechanism, err := franzSASLMechanism(cfg.SASLConfig)
		if err != nil {
			return nil, err
		}

		opts = append(opts, kgo.SASL(mechanism))
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating dlq kafka client: %w", err)
	}

	return &kafkaRejectSink{
		topic: cfg.RejectedTopic,
		cl:    cl,
	}, nil
}

func franzSASLMechanism(cfg *SASLConfig) (sasl.Mechanism, error) {
	if cfg == nil {
		return nil, errors.New("nil sasl config")
	}

	password, err := resolveSASLSecret(cfg)
	if err != nil {
		return nil, err
	}

	mechanism := strings.ToUpper(strings.TrimSpace(cfg.Mechanism))
	switch mechanism {
	case "", "PLAIN":
		return plain.Auth{
			User: cfg.User,
			Pass: password,
		}.AsMechanism(), nil
	case "SCRAM-SHA-256":
		return scram.Auth{
			User: cfg.User,
			Pass: password,
		}.AsSha256Mechanism(), nil
	case "SCRAM-SHA-512":
		return scram.Auth{
			User: cfg.User,
			Pass: password,
		}.AsSha512Mechanism(), nil
	case "OAUTHBEARER":
		return oauth.Auth{
			Token: password,
		}.AsMechanism(), nil
	default:
		return nil, fmt.Errorf("unsupported kafka sasl mechanism %q", cfg.Mechanism)
	}
}
