package source

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	"github.com/redpanda-data/benthos/v4/public/service"
	_ "github.com/redpanda-data/connect/v4/public/components/kafka"

	chtransform "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v3"
)

const benthosOutputType = "xatu_clickhouse"

// NewBenthosStream creates a Benthos stream that consumes from Kafka and writes
// to ClickHouse via the custom xatu_clickhouse output plugin.
func NewBenthosStream(
	log logrus.FieldLogger,
	logLevel string,
	kafkaConfig *KafkaConfig,
	metrics *telemetry.Metrics,
	routeEngine *chtransform.Engine,
	writer Writer,
	classifier WriteErrorClassifier,
) (*service.Stream, error) {
	if kafkaConfig == nil {
		return nil, fmt.Errorf("nil kafka config")
	}

	rejectSink, err := newRejectSink(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("creating reject sink: %w", err)
	}

	env := service.NewEnvironment()

	if registerErr := env.RegisterBatchOutput(
		benthosOutputType,
		service.NewConfigSpec(),
		func(_ *service.ParsedConfig, _ *service.Resources) (out service.BatchOutput, policy service.BatchPolicy, maxInFlight int, err error) {
			return &xatuClickHouseOutput{
				log:          log.WithField("component", "benthos_clickhouse_output"),
				encoding:     kafkaConfig.Encoding,
				deliveryMode: kafkaConfig.DeliveryMode,
				router:       routeEngine,
				writer:       writer,
				metrics:      metrics,
				classifier:   classifier,
				rejectSink:   rejectSink,
			}, service.BatchPolicy{}, 1, nil
		},
	); registerErr != nil {
		return nil, fmt.Errorf("registering output plugin: %w", registerErr)
	}

	streamConfigBytes, err := benthosConfigYAML(logLevel, kafkaConfig)
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

func benthosConfigYAML(logLevel string, kafkaConfig *KafkaConfig) ([]byte, error) {
	if kafkaConfig == nil {
		return nil, fmt.Errorf("nil kafka config")
	}

	inputKafka := map[string]any{
		"seed_brokers":              append([]string(nil), kafkaConfig.Brokers...),
		"regexp_topics_include":     append([]string(nil), kafkaConfig.Topics...),
		"consumer_group":            kafkaConfig.ConsumerGroup,
		"start_offset":              benthosStartOffset(kafkaConfig.OffsetDefault),
		"commit_period":             kafkaConfig.CommitInterval.String(),
		"fetch_min_bytes":           fmt.Sprintf("%dB", kafkaConfig.FetchMinBytes),
		"fetch_max_wait":            fmt.Sprintf("%dms", kafkaConfig.FetchWaitMaxMs),
		"fetch_max_partition_bytes": fmt.Sprintf("%dB", kafkaConfig.MaxPartitionFetchBytes),
		"session_timeout":           fmt.Sprintf("%dms", kafkaConfig.SessionTimeoutMs),
		"heartbeat_interval":        fmt.Sprintf("%dms", kafkaConfig.HeartbeatIntervalMs),
	}

	if kafkaConfig.TLS {
		inputKafka["tls"] = map[string]any{
			"enabled": true,
		}
	}

	if kafkaConfig.SASLConfig != nil {
		saslObject, err := benthosSASLObject(kafkaConfig.SASLConfig)
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
			"level": benthosLogLevel(logLevel),
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
