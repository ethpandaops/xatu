//nolint:wsl_v5 // Tests are easier to scan with compact setup blocks.
package source

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	yaml "gopkg.in/yaml.v3"

	consrouter "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var testMetricNSCounter uint64

type testRoute struct {
	eventName  xatu.Event_Name
	table      string
	rows       []map[string]any
	flattenErr error
}

func (r testRoute) EventNames() []xatu.Event_Name {
	return []xatu.Event_Name{r.eventName}
}

func (r testRoute) TableName() string {
	return r.table
}

func (r testRoute) Flatten(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata) ([]map[string]any, error) {
	if r.flattenErr != nil {
		return nil, r.flattenErr
	}

	out := make([]map[string]any, len(r.rows))
	for i := range r.rows {
		copyRow := make(map[string]any, len(r.rows[i]))
		for k, v := range r.rows[i] {
			copyRow[k] = v
		}
		out[i] = copyRow
	}

	return out, nil
}

func (r testRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool {
	return true
}

type testWriter struct {
	writes     map[string]int
	flushErrs  []error
	flushCalls int
}

func (w *testWriter) Start(context.Context) error {
	return nil
}

func (w *testWriter) Stop(context.Context) error {
	return nil
}

func (w *testWriter) Write(table string, rows []map[string]any) {
	if w.writes == nil {
		w.writes = make(map[string]int)
	}

	w.writes[table] += len(rows)
}

func (w *testWriter) FlushAll(context.Context) error {
	w.flushCalls++
	if len(w.flushErrs) == 0 {
		return nil
	}

	err := w.flushErrs[0]
	w.flushErrs = w.flushErrs[1:]

	return err
}

type testRejectSink struct {
	enabled bool
	err     error
	records []rejectedRecord
}

type testWriteError struct {
	table     string
	permanent bool
}

func (e *testWriteError) Error() string {
	return fmt.Sprintf("table=%s permanent=%t", e.table, e.permanent)
}

type testErrorClassifier struct{}

func (testErrorClassifier) IsPermanent(err error) bool {
	var twErr *testWriteError
	if errors.As(err, &twErr) {
		return twErr.permanent
	}

	return false
}

func (testErrorClassifier) Table(err error) string {
	var twErr *testWriteError
	if errors.As(err, &twErr) {
		return twErr.table
	}

	return ""
}

func (s *testRejectSink) Write(_ context.Context, record *rejectedRecord) error {
	if s.err != nil {
		return s.err
	}

	if record != nil {
		s.records = append(s.records, *record)
	}

	return nil
}

func (s *testRejectSink) Close() error {
	return nil
}

func (s *testRejectSink) Enabled() bool {
	return s.enabled
}

func TestBenthosConfigYAML(t *testing.T) {
	cfg := &KafkaConfig{
		Brokers:                []string{"kafka-1:9092", "kafka-2:9092"},
		Topics:                 []string{"^general-.+"},
		ConsumerGroup:          "xatu-consumoor",
		Encoding:               "protobuf",
		FetchMinBytes:          64,
		FetchWaitMaxMs:         250,
		MaxPartitionFetchBytes: 1048576,
		SessionTimeoutMs:       30000,
		HeartbeatIntervalMs:    3000,
		OffsetDefault:          "newest",
		CommitInterval:         7 * time.Second,
	}

	raw, err := benthosConfigYAML("debug", cfg)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &parsed))

	input, ok := parsed["input"].(map[string]any)
	require.True(t, ok)

	kafka, ok := input["kafka_franz"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "xatu-consumoor", kafka["consumer_group"])
	assert.Equal(t, "latest", kafka["start_offset"])
	assert.Equal(t, "7s", kafka["commit_period"])

	output, ok := parsed["output"].(map[string]any)
	require.True(t, ok)

	_, hasOutput := output[benthosOutputType]
	assert.True(t, hasOutput)
}

func TestBenthosSASLObjectUsesPasswordFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte("token-from-file\n"), 0o600))

	obj, err := benthosSASLObject(&SASLConfig{
		Mechanism:    "OAUTHBEARER",
		User:         "ignored",
		PasswordFile: secretPath,
	})
	require.NoError(t, err)

	assert.Equal(t, "OAUTHBEARER", obj["mechanism"])
	assert.Equal(t, "token-from-file", obj["token"])
	_, hasPassword := obj["password"]
	assert.False(t, hasPassword)
}

func TestKafkaMetadataFallback(t *testing.T) {
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(nil))

	msg := service.NewMessage(nil)
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(msg))

	msg.MetaSet("kafka_topic", "xatu-mainnet")
	msg.MetaSet("kafka_partition", "2")
	msg.MetaSet("kafka_offset", "42")

	meta := kafkaMetadata(msg)
	assert.Equal(t, "xatu-mainnet", meta.Topic)
	assert.Equal(t, int32(2), meta.Partition)
	assert.Equal(t, int64(42), meta.Offset)
}

func TestWriteBatchBatchModeRejectsMalformedWithoutRetry(t *testing.T) {
	writer := &testWriter{}
	rejectSink := &testRejectSink{}
	output := &xatuClickHouseOutput{
		log:          logrus.New(),
		encoding:     "json",
		deliveryMode: DeliveryModeBatch,
		router: newRouter(t, []flattener.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
				rows:      []map[string]any{{"slot": int64(1)}},
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		classifier: testErrorClassifier{},
		rejectSink: rejectSink,
	}

	msgs := service.MessageBatch{
		newKafkaMessage([]byte("{"), "topic-a", 0, 1),
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 2),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.NoError(t, err)
	assert.Equal(t, 1, writer.flushCalls)
	assert.Equal(t, 1, writer.writes["beacon_head"])
	require.Len(t, rejectSink.records, 1)
	assert.Equal(t, rejectReasonDecode, rejectSink.records[0].Reason)
}

func TestWriteBatchBatchModeTransientWriteFailureFailsImpactedMessages(t *testing.T) {
	writer := &testWriter{
		flushErrs: []error{
			&testWriteError{table: "table_b", permanent: false},
		},
	}

	output := &xatuClickHouseOutput{
		log:          logrus.New(),
		encoding:     "json",
		deliveryMode: DeliveryModeBatch,
		router: newRouter(t, []flattener.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_a",
				rows:      []map[string]any{{"slot": int64(1)}},
			},
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				table:     "table_b",
				rows:      []map[string]any{{"slot": int64(2)}},
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		classifier: testErrorClassifier{},
		rejectSink: &testRejectSink{},
	}

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 1),
		newKafkaMessage(mustEventJSON(t, "evt-2", xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK), "topic-a", 0, 2),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, []int{1}, failedIndexesFromBatchError(t, msgs, err))
}

func TestWriteBatchMessageModePermanentWriteErrorRejectsAndContinues(t *testing.T) {
	writer := &testWriter{
		flushErrs: []error{
			&testWriteError{table: "beacon_head", permanent: true},
		},
	}
	rejectSink := &testRejectSink{}
	output := &xatuClickHouseOutput{
		log:          logrus.New(),
		encoding:     "json",
		deliveryMode: DeliveryModeMessage,
		router: newRouter(t, []flattener.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
				rows:      []map[string]any{{"slot": int64(1)}},
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		classifier: testErrorClassifier{},
		rejectSink: rejectSink,
	}

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 1),
		newKafkaMessage(mustEventJSON(t, "evt-2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 2),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.NoError(t, err)
	assert.Equal(t, 2, writer.flushCalls)
	require.Len(t, rejectSink.records, 1)
	assert.Equal(t, rejectReasonWritePermanent, rejectSink.records[0].Reason)
	assert.Equal(t, int64(1), rejectSink.records[0].Kafka.Offset)
}

func TestWriteBatchMessageModeTransientFailureFailsRemaining(t *testing.T) {
	writer := &testWriter{
		flushErrs: []error{errors.New("clickhouse unavailable")},
	}
	output := &xatuClickHouseOutput{
		log:          logrus.New(),
		encoding:     "json",
		deliveryMode: DeliveryModeMessage,
		router: newRouter(t, []flattener.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
				rows:      []map[string]any{{"slot": int64(1)}},
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		classifier: testErrorClassifier{},
		rejectSink: &testRejectSink{},
	}

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 1),
		newKafkaMessage(mustEventJSON(t, "evt-2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 2),
		newKafkaMessage(mustEventJSON(t, "evt-3", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 3),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, []int{0, 1, 2}, failedIndexesFromBatchError(t, msgs, err))
	assert.Equal(t, 1, writer.flushCalls)
}

func TestWriteBatchRejectSinkFailureMakesMessageRetry(t *testing.T) {
	output := &xatuClickHouseOutput{
		log:          logrus.New(),
		encoding:     "json",
		deliveryMode: DeliveryModeBatch,
		router:       newRouter(t, nil),
		writer:       &testWriter{},
		metrics:      newTestMetrics(),
		classifier:   testErrorClassifier{},
		rejectSink: &testRejectSink{
			err: errors.New("dlq unavailable"),
		},
	}

	msgs := service.MessageBatch{
		newKafkaMessage([]byte("{"), "topic-a", 0, 1),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err))
}

func failedIndexesFromBatchError(t *testing.T, msgs service.MessageBatch, err error) []int {
	t.Helper()

	var batchErr *service.BatchError
	require.ErrorAs(t, err, &batchErr)

	out := make([]int, 0, 4)
	_ = msgs

	//nolint:staticcheck // Tests intentionally use direct indexes from this batch.
	batchErr.WalkMessages(func(index int, _ *service.Message, msgErr error) bool {
		if msgErr != nil {
			out = append(out, index)
		}

		return true
	})

	sort.Ints(out)

	return out
}

func newRouter(t *testing.T, routes []flattener.Route) *consrouter.Engine {
	t.Helper()

	return consrouter.New(logrus.New(), routes, nil, newTestMetrics())
}

func newTestMetrics() *telemetry.Metrics {
	ns := atomic.AddUint64(&testMetricNSCounter, 1)

	return telemetry.NewMetrics(fmt.Sprintf("xatu_consumoor_test_%d", ns))
}

func mustEventJSON(t *testing.T, id string, name xatu.Event_Name) []byte {
	t.Helper()

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Id:       id,
			Name:     name,
			DateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
		},
	}

	raw, err := protojson.Marshal(event)
	require.NoError(t, err)

	return raw
}

func newKafkaMessage(payload []byte, topic string, partition int32, offset int64) *service.Message {
	msg := service.NewMessage(payload)
	msg.MetaSet("kafka_topic", topic)
	msg.MetaSet("kafka_partition", strconv.FormatInt(int64(partition), 10))
	msg.MetaSet("kafka_offset", strconv.FormatInt(offset, 10))

	return msg
}

func TestKafkaTopicMetadataFallback(t *testing.T) {
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(nil))

	msg := service.NewMessage(nil)
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(msg))

	msg.MetaSet("kafka_topic", "xatu-mainnet")
	assert.Equal(t, "xatu-mainnet", kafkaTopicMetadata(msg))
}

func TestFranzSASLMechanism(t *testing.T) {
	t.Run("defaults to PLAIN", func(t *testing.T) {
		mech, err := franzSASLMechanism(&SASLConfig{
			User:     "user",
			Password: "pass",
		})
		require.NoError(t, err)
		assert.Equal(t, "PLAIN", mech.Name())
	})

	t.Run("rejects unsupported mechanism", func(t *testing.T) {
		_, err := franzSASLMechanism(&SASLConfig{
			User:      "user",
			Password:  "pass",
			Mechanism: "KERBEROS",
		})
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "unsupported"))
	})
}
