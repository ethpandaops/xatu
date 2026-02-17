//nolint:wsl_v5 // Pipeline branches are clearer when kept compact.
package source

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	"github.com/redpanda-data/benthos/v4/public/service"
	_ "github.com/redpanda-data/connect/v4/public/components/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/oauth"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"

	chtransform "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	yaml "gopkg.in/yaml.v3"
)

const benthosOutputType = "xatu_clickhouse"
const unknownKafkaTopic = "unknown"

const (
	rejectReasonDecode         = "decode_error"
	rejectReasonRouteRejected  = "route_rejected"
	rejectReasonWritePermanent = "write_permanent"
)

var errBatchWriteFailed = errors.New("clickhouse batch write failed")

type messageContext struct {
	raw   []byte
	event *xatu.DecoratedEvent
	kafka kafkaMessageMetadata
}

type kafkaMessageMetadata struct {
	Topic     string
	Partition int32
	Offset    int64
}

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

type xatuClickHouseOutput struct {
	log          logrus.FieldLogger
	encoding     string
	deliveryMode string
	router       *chtransform.Engine
	writer       Writer
	metrics      *telemetry.Metrics
	classifier   WriteErrorClassifier
	rejectSink   rejectSink

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

	if strings.EqualFold(o.deliveryMode, DeliveryModeMessage) {
		return o.writeMessageMode(ctx, msgs)
	}

	return o.writeBatchMode(ctx, msgs)
}

func (o *xatuClickHouseOutput) writeMessageMode(ctx context.Context, msgs service.MessageBatch) error {
	var batchErr *service.BatchError

	for i, msg := range msgs {
		mctx := o.buildMessageContext(msg)
		o.metrics.MessagesConsumed().WithLabelValues(mctx.kafka.Topic).Inc()

		raw, err := msg.AsBytes()
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to read message bytes")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason: rejectReasonDecode,
				Err:    err.Error(),
				Kafka:  mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		mctx.raw = append(mctx.raw[:0], raw...)

		event, err := decodeDecoratedEvent(o.encoding, raw)
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to decode message")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:  rejectReasonDecode,
				Err:     err.Error(),
				Payload: raw,
				Kafka:   mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		mctx.event = event
		outcome := o.router.Route(event)

		if outcome.Status == chtransform.StatusRejected {
			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonRouteRejected,
				Err:       "route rejected message",
				Payload:   raw,
				EventName: event.GetEvent().GetName().String(),
				Kafka:     mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		if outcome.Status == chtransform.StatusErrored {
			batchErr = addBatchFailure(batchErr, msgs, i, errors.New("route errored"))

			continue
		}

		if len(outcome.Results) == 0 {
			continue
		}

		hasRows := false

		for _, result := range outcome.Results {
			if len(result.Rows) == 0 {
				continue
			}

			hasRows = true

			o.writer.Write(result.Table, result.Rows)
		}

		if !hasRows {
			continue
		}

		if err := o.writer.FlushAll(ctx); err != nil {
			if o.isPermanentWriteError(err) {
				if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
					Reason:    rejectReasonWritePermanent,
					Err:       err.Error(),
					Payload:   raw,
					EventName: event.GetEvent().GetName().String(),
					Kafka:     mctx.kafka,
				}); rejectErr != nil {
					batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
				}

				o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Dropped permanently invalid message")

				continue
			}

			batchErr = addBatchFailure(batchErr, msgs, i, err)
			for j := i + 1; j < len(msgs); j++ {
				batchErr = addBatchFailure(batchErr, msgs, j, err)
			}

			return batchErr
		}
	}

	if batchErr != nil {
		return batchErr
	}

	return nil
}

func (o *xatuClickHouseOutput) writeBatchMode(ctx context.Context, msgs service.MessageBatch) error {
	msgContexts := make([]messageContext, len(msgs))
	rowsByTable := make(map[string][]map[string]any, 32)
	tableToMessageIndexes := make(map[string]map[int]struct{}, 32)

	var batchErr *service.BatchError

	for i, msg := range msgs {
		mctx := o.buildMessageContext(msg)
		msgContexts[i] = mctx
		o.metrics.MessagesConsumed().WithLabelValues(mctx.kafka.Topic).Inc()

		raw, err := msg.AsBytes()
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to read message bytes")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason: rejectReasonDecode,
				Err:    err.Error(),
				Kafka:  mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		msgContexts[i].raw = append(msgContexts[i].raw[:0], raw...)

		event, err := decodeDecoratedEvent(o.encoding, raw)
		if err != nil {
			o.metrics.DecodeErrors().WithLabelValues(mctx.kafka.Topic).Inc()
			o.log.WithError(err).WithField("topic", mctx.kafka.Topic).Warn("Failed to decode message")

			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:  rejectReasonDecode,
				Err:     err.Error(),
				Payload: raw,
				Kafka:   mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		msgContexts[i].event = event
		outcome := o.router.Route(event)

		if outcome.Status == chtransform.StatusRejected {
			if rejectErr := o.rejectMessage(ctx, &rejectedRecord{
				Reason:    rejectReasonRouteRejected,
				Err:       "route rejected message",
				Payload:   raw,
				EventName: event.GetEvent().GetName().String(),
				Kafka:     mctx.kafka,
			}); rejectErr != nil {
				batchErr = addBatchFailure(batchErr, msgs, i, rejectErr)
			}

			continue
		}

		if outcome.Status == chtransform.StatusErrored {
			batchErr = addBatchFailure(batchErr, msgs, i, errors.New("route errored"))

			continue
		}

		for _, result := range outcome.Results {
			if len(result.Rows) == 0 {
				continue
			}

			rowsByTable[result.Table] = append(rowsByTable[result.Table], result.Rows...)
			addTableMessageIndex(tableToMessageIndexes, result.Table, i)
		}
	}

	if len(rowsByTable) == 0 {
		if batchErr != nil {
			return batchErr
		}

		return nil
	}

	for table, rows := range rowsByTable {
		o.writer.Write(table, rows)
	}

	if err := o.writer.FlushAll(ctx); err != nil {
		failedIndexes := failedIndexesForWriteError(tableToMessageIndexes, err, o.classifier)
		if len(failedIndexes) == 0 {
			failedIndexes = allTableMessageIndexes(tableToMessageIndexes)
		}

		if o.isPermanentWriteError(err) {
			for _, idx := range failedIndexes {
				rejectErr := o.rejectMessage(ctx, &rejectedRecord{
					Reason:    rejectReasonWritePermanent,
					Err:       err.Error(),
					Payload:   msgContexts[idx].raw,
					EventName: eventNameFromContext(msgContexts[idx]),
					Kafka:     msgContexts[idx].kafka,
				})
				if rejectErr != nil {
					batchErr = addBatchFailure(batchErr, msgs, idx, rejectErr)

					continue
				}
			}

			o.log.WithError(err).Warn("Dropped permanently invalid rows during batch flush")
		} else {
			for _, idx := range failedIndexes {
				batchErr = addBatchFailure(batchErr, msgs, idx, err)
			}
		}
	}

	if batchErr != nil {
		return batchErr
	}

	return nil
}

func (o *xatuClickHouseOutput) Close(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.started {
		return nil
	}

	writerErr := o.writer.Stop(ctx)

	var rejectErr error

	if o.rejectSink != nil {
		rejectErr = o.rejectSink.Close()
	}

	o.started = false

	if writerErr != nil {
		return writerErr
	}

	return rejectErr
}

func (o *xatuClickHouseOutput) buildMessageContext(msg *service.Message) messageContext {
	return messageContext{
		kafka: kafkaMetadata(msg),
	}
}

func (o *xatuClickHouseOutput) rejectMessage(ctx context.Context, record *rejectedRecord) error {
	if record == nil {
		return errors.New("nil rejected record")
	}

	o.metrics.MessagesRejected().WithLabelValues(record.Reason).Inc()

	if o.rejectSink == nil {
		return nil
	}

	if err := o.rejectSink.Write(ctx, record); err != nil {
		o.metrics.DLQErrors().WithLabelValues(record.Reason).Inc()

		return fmt.Errorf("writing rejected message to dlq: %w", err)
	}

	if o.rejectSink.Enabled() {
		o.metrics.DLQWrites().WithLabelValues(record.Reason).Inc()
	}

	return nil
}

func addBatchFailure(
	batchErr *service.BatchError,
	msgs service.MessageBatch,
	index int,
	err error,
) *service.BatchError {
	if batchErr == nil {
		batchErr = service.NewBatchError(msgs, errBatchWriteFailed)
	}

	return batchErr.Failed(index, err)
}

func addTableMessageIndex(indexes map[string]map[int]struct{}, table string, idx int) {
	perTable, ok := indexes[table]
	if !ok {
		perTable = make(map[int]struct{}, 8)
		indexes[table] = perTable
	}

	perTable[idx] = struct{}{}
}

func failedIndexesForWriteError(
	tableToMessageIndexes map[string]map[int]struct{},
	err error,
	classifier WriteErrorClassifier,
) []int {
	if classifier == nil {
		return nil
	}

	table := classifier.Table(err)
	if table == "" {
		return nil
	}

	perTable, ok := tableToMessageIndexes[table]
	if !ok {
		return nil
	}

	out := make([]int, 0, len(perTable))
	for idx := range perTable {
		out = append(out, idx)
	}

	sort.Ints(out)

	return out
}

func (o *xatuClickHouseOutput) isPermanentWriteError(err error) bool {
	if o.classifier == nil {
		return false
	}

	return o.classifier.IsPermanent(err)
}

func allTableMessageIndexes(tableToMessageIndexes map[string]map[int]struct{}) []int {
	seen := make(map[int]struct{}, 32)
	for _, perTable := range tableToMessageIndexes {
		for idx := range perTable {
			seen[idx] = struct{}{}
		}
	}

	out := make([]int, 0, len(seen))
	for idx := range seen {
		out = append(out, idx)
	}

	sort.Ints(out)

	return out
}

func eventNameFromContext(mctx messageContext) string {
	if mctx.event == nil || mctx.event.GetEvent() == nil {
		return ""
	}

	return mctx.event.GetEvent().GetName().String()
}

func kafkaMetadata(msg *service.Message) kafkaMessageMetadata {
	meta := kafkaMessageMetadata{
		Topic:     unknownKafkaTopic,
		Partition: -1,
		Offset:    -1,
	}

	if msg == nil {
		return meta
	}

	meta.Topic = kafkaTopicMetadata(msg)

	if partition, ok := kafkaInt32Metadata(msg, "kafka_partition"); ok {
		meta.Partition = partition
	}

	if offset, ok := kafkaInt64Metadata(msg, "kafka_offset"); ok {
		meta.Offset = offset
	}

	return meta
}

func kafkaInt64Metadata(msg *service.Message, key string) (int64, bool) {
	if msg == nil {
		return 0, false
	}

	raw, ok := msg.MetaGet(key)
	if !ok || raw == "" {
		return 0, false
	}

	out, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return out, true
}

func kafkaInt32Metadata(msg *service.Message, key string) (int32, bool) {
	if msg == nil {
		return 0, false
	}

	raw, ok := msg.MetaGet(key)
	if !ok || raw == "" {
		return 0, false
	}

	out, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false
	}

	return int32(out), true
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
