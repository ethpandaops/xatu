package consumoor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	"github.com/redpanda-data/benthos/v4/public/service"
	_ "github.com/redpanda-data/connect/v4/public/components/kafka"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	yaml "gopkg.in/yaml.v3"
)

const benthosOutputType = "xatu_clickhouse"
const unknownKafkaTopic = "unknown"

type xatuClickHouseOutput struct {
	log      logrus.FieldLogger
	encoding string
	router   *Router
	writer   Writer
	metrics  *Metrics

	mu      sync.Mutex
	started bool
}

func (o *xatuClickHouseOutput) Connect(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.started {
		return nil
	}

	if err := o.writer.Start(ctx); err != nil {
		return err
	}

	o.started = true

	return nil
}

func (o *xatuClickHouseOutput) WriteBatch(ctx context.Context, msgs service.MessageBatch) error {
	if len(msgs) == 0 {
		return nil
	}

	rowsByTable := make(map[string][]map[string]any, 32)

	for _, msg := range msgs {
		topic := kafkaTopicMetadata(msg)
		o.metrics.messagesConsumed.WithLabelValues(topic).Inc()

		raw, err := msg.AsBytes()
		if err != nil {
			o.metrics.decodeErrors.WithLabelValues(topic).Inc()
			o.log.WithError(err).WithField("topic", topic).Warn("Failed to read message bytes")

			continue
		}

		event, err := decodeDecoratedEvent(o.encoding, raw)
		if err != nil {
			o.metrics.decodeErrors.WithLabelValues(topic).Inc()
			o.log.WithError(err).WithField("topic", topic).Warn("Failed to decode message")

			continue
		}

		outcome := o.router.Route(event)
		if outcome.Status != DeliveryStatusDelivered {
			continue
		}

		for _, result := range outcome.Results {
			if len(result.Rows) == 0 {
				continue
			}

			rowsByTable[result.Table] = append(rowsByTable[result.Table], result.Rows...)
		}
	}

	if len(rowsByTable) == 0 {
		return nil
	}

	for table, rows := range rowsByTable {
		o.writer.Write(table, rows)
	}

	if err := o.writer.FlushAll(ctx); err != nil {
		if isPermanentRowConversionError(err) {
			o.log.WithError(err).Warn("Dropped permanently invalid rows during batch flush")

			return nil
		}

		return err
	}

	return nil
}

func (o *xatuClickHouseOutput) Close(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.started {
		return nil
	}

	err := o.writer.Stop(ctx)
	o.started = false

	return err
}

func NewBenthosStream(
	log logrus.FieldLogger,
	config *Config,
	metrics *Metrics,
	router *Router,
	writer Writer,
) (*service.Stream, error) {
	env := service.NewEnvironment()

	if err := env.RegisterBatchOutput(
		benthosOutputType,
		service.NewConfigSpec(),
		func(_ *service.ParsedConfig, _ *service.Resources) (out service.BatchOutput, policy service.BatchPolicy, maxInFlight int, err error) {
			return &xatuClickHouseOutput{
				log:      log.WithField("component", "benthos_clickhouse_output"),
				encoding: config.Kafka.Encoding,
				router:   router,
				writer:   writer,
				metrics:  metrics,
			}, service.BatchPolicy{}, 1, nil
		},
	); err != nil {
		return nil, fmt.Errorf("registering output plugin: %w", err)
	}

	streamConfigBytes, err := benthosConfigYAML(config)
	if err != nil {
		return nil, err
	}

	builder := env.NewStreamBuilder()
	if setErr := builder.SetYAML(string(streamConfigBytes)); setErr != nil {
		return nil, fmt.Errorf("parsing benthos stream config: %w", setErr)
	}

	stream, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("building benthos stream: %w", err)
	}

	return stream, nil
}

func benthosConfigYAML(config *Config) ([]byte, error) {
	if config == nil {
		return nil, fmt.Errorf("nil config")
	}

	inputKafka := map[string]any{
		"seed_brokers":              append([]string(nil), config.Kafka.Brokers...),
		"regexp_topics_include":     append([]string(nil), config.Kafka.Topics...),
		"consumer_group":            config.Kafka.ConsumerGroup,
		"start_offset":              benthosStartOffset(config.Kafka.OffsetDefault),
		"commit_period":             config.Kafka.CommitInterval.String(),
		"fetch_min_bytes":           fmt.Sprintf("%dB", config.Kafka.FetchMinBytes),
		"fetch_max_wait":            fmt.Sprintf("%dms", config.Kafka.FetchWaitMaxMs),
		"fetch_max_partition_bytes": fmt.Sprintf("%dB", config.Kafka.MaxPartitionFetchBytes),
		"session_timeout":           fmt.Sprintf("%dms", config.Kafka.SessionTimeoutMs),
		"heartbeat_interval":        fmt.Sprintf("%dms", config.Kafka.HeartbeatIntervalMs),
	}

	if config.Kafka.TLS {
		inputKafka["tls"] = map[string]any{
			"enabled": true,
		}
	}

	if config.Kafka.SASLConfig != nil {
		saslObject, err := benthosSASLObject(config.Kafka.SASLConfig)
		if err != nil {
			return nil, err
		}

		inputKafka["sasl"] = []map[string]any{saslObject}
	}

	streamConfig := map[string]any{
		"http": map[string]any{
			"enabled": false,
		},
		"metrics": map[string]any{
			"none": map[string]any{},
		},
		"logger": map[string]any{
			"level": benthosLogLevel(config.LoggingLevel),
		},
		"shutdown_timeout": "30s",
		"input": map[string]any{
			"kafka_franz": inputKafka,
		},
		"output": map[string]any{
			benthosOutputType: map[string]any{},
		},
	}

	return yaml.Marshal(streamConfig)
}

func benthosStartOffset(offsetDefault string) string {
	if strings.EqualFold(offsetDefault, "newest") {
		return "latest"
	}

	return "earliest"
}

func benthosLogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "panic", "fatal", "error":
		return "ERROR"
	case "warn", "warning":
		return "WARN"
	case "debug":
		return "DEBUG"
	case "trace":
		return "TRACE"
	case "off":
		return "OFF"
	default:
		return "INFO"
	}
}

func benthosSASLObject(cfg *SASLConfig) (map[string]any, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil sasl config")
	}

	password, err := resolveSASLSecret(cfg)
	if err != nil {
		return nil, err
	}

	mechanism := strings.TrimSpace(cfg.Mechanism)
	if mechanism == "" {
		mechanism = "PLAIN"
	}

	out := map[string]any{
		"mechanism": mechanism,
	}

	if strings.EqualFold(mechanism, "OAUTHBEARER") {
		out["token"] = password

		return out, nil
	}

	out["username"] = cfg.User
	out["password"] = password

	return out, nil
}

func resolveSASLSecret(cfg *SASLConfig) (string, error) {
	if cfg == nil {
		return "", nil
	}

	if cfg.Password != "" {
		return cfg.Password, nil
	}

	if cfg.PasswordFile == "" {
		return "", nil
	}

	data, err := os.ReadFile(cfg.PasswordFile)
	if err != nil {
		return "", fmt.Errorf("reading sasl password file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func decodeDecoratedEvent(encoding string, data []byte) (*xatu.DecoratedEvent, error) {
	event := &xatu.DecoratedEvent{}

	switch encoding {
	case "protobuf":
		if err := proto.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("protobuf unmarshal: %w", err)
		}
	default:
		if err := protojson.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}
	}

	return event, nil
}

func kafkaTopicMetadata(msg *service.Message) string {
	if msg == nil {
		return unknownKafkaTopic
	}

	topic, ok := msg.MetaGet("kafka_topic")
	if !ok || topic == "" {
		return unknownKafkaTopic
	}

	return topic
}

func isPermanentRowConversionError(err error) bool {
	var conversionErr *rowConversionError

	return errors.As(err, &conversionErr)
}
