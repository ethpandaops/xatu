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

	ch "github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/consumoor/router"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var testMetricNSCounter uint64

type testRoute struct {
	eventName xatu.Event_Name
	table     string
}

func (r testRoute) EventNames() []xatu.Event_Name {
	return []xatu.Event_Name{r.eventName}
}

func (r testRoute) TableName() string {
	return r.table
}

func (r testRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool {
	return true
}

func (r testRoute) NewBatch() route.ColumnarBatch {
	return nil
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

func (w *testWriter) FlushTableEvents(_ context.Context, tableEvents map[string][]*xatu.DecoratedEvent) error {
	w.flushCalls++

	if w.writes == nil {
		w.writes = make(map[string]int, len(tableEvents))
	}

	for table, events := range tableEvents {
		w.writes[table] += len(events)
	}

	for i, err := range w.flushErrs {
		if twe, ok := err.(*testWriteError); ok {
			if _, found := tableEvents[twe.table]; !found {
				continue
			}
		}

		w.flushErrs = append(w.flushErrs[:i], w.flushErrs[i+1:]...)

		return err
	}

	return nil
}

func (w *testWriter) Ping(context.Context) error {
	return nil
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
		FetchMaxBytes:          5242880,
		SessionTimeoutMs:       30000,
		OffsetDefault:          "latest",
		CommitInterval:         7 * time.Second,
		ConnectTimeout:         10 * time.Second,
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
	assert.Equal(t, "5242880B", kafka["fetch_max_bytes"])
	assert.Equal(t, "1048576B", kafka["fetch_max_partition_bytes"])

	tcp, ok := kafka["tcp"].(map[string]any)
	require.True(t, ok, "tcp block should be present when ConnectTimeout > 0")
	assert.Equal(t, "10s", tcp["connect_timeout"])

	output, ok := parsed["output"].(map[string]any)
	require.True(t, ok)

	_, hasOutput := output[benthosOutputType]
	assert.True(t, hasOutput)
}

func TestBenthosConfigYAML_NoTCPBlockWhenConnectTimeoutZero(t *testing.T) {
	cfg := &KafkaConfig{
		Brokers:                []string{"kafka-1:9092"},
		Topics:                 []string{"^test-.+"},
		ConsumerGroup:          "xatu-consumoor",
		Encoding:               "json",
		FetchMinBytes:          1,
		FetchWaitMaxMs:         250,
		MaxPartitionFetchBytes: 1048576,
		FetchMaxBytes:          10485760,
		SessionTimeoutMs:       30000,
		OffsetDefault:          "earliest",
		CommitInterval:         5 * time.Second,
		ConnectTimeout:         0,
	}

	raw, err := benthosConfigYAML("info", cfg)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &parsed))

	input, ok := parsed["input"].(map[string]any)
	require.True(t, ok)

	kafka, ok := input["kafka_franz"].(map[string]any)
	require.True(t, ok)

	_, hasTCP := kafka["tcp"]
	assert.False(t, hasTCP, "tcp block should be absent when ConnectTimeout is 0")
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
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
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
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_a",
			},
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
				table:     "table_b",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
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

func TestWriteBatchRejectSinkFailureMakesMessageRetry(t *testing.T) {
	output := &xatuClickHouseOutput{
		log:      logrus.New(),
		encoding: "json",
		router:   newRouter(t, nil),
		writer:   &testWriter{},
		metrics:  newTestMetrics(),
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

func newRouter(t *testing.T, routes []route.Route) *router.Engine {
	t.Helper()

	return router.New(logrus.New(), routes, nil, newTestMetrics())
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

func TestWriteBatchMultiTableTransientFailureFailsAllImpactedMessages(t *testing.T) {
	// Same event type routes to two tables. When FlushTables returns a
	// joined error, all messages in the group must be marked as failed.
	writer := &testWriter{
		flushErrs: []error{
			errors.Join(
				&testWriteError{table: "table_a", permanent: false},
				&testWriteError{table: "table_b", permanent: false},
			),
		},
	}

	output := &xatuClickHouseOutput{
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_a",
			},
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_b",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		rejectSink: &testRejectSink{},
	}

	msgs := service.MessageBatch{
		newKafkaMessage(
			mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD),
			"topic-a", 0, 1,
		),
		newKafkaMessage(
			mustEventJSON(t, "evt-2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD),
			"topic-a", 0, 2,
		),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, []int{0, 1}, failedIndexesFromBatchError(t, msgs, err),
		"both messages should be marked as failed when group flush fails")
}

func TestWriteBatchMultiTablePermanentFailureRejectsAllImpactedMessages(t *testing.T) {
	// Same event type routes to two tables. When FlushTables returns a
	// permanent error, all messages in the group should be DLQ'd.
	writer := &testWriter{
		flushErrs: []error{
			&ch.Exception{Code: proto.ErrUnknownTable, Name: "DB::Exception", Message: "test permanent error"},
		},
	}

	rejectSink := &testRejectSink{}
	output := &xatuClickHouseOutput{
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_a",
			},
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "table_b",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		rejectSink: rejectSink,
	}

	msgs := service.MessageBatch{
		newKafkaMessage(
			mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD),
			"topic-a", 0, 1,
		),
		newKafkaMessage(
			mustEventJSON(t, "evt-2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD),
			"topic-a", 0, 2,
		),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.NoError(t, err,
		"permanent write errors should be rejected, not returned as batch error")
	require.Len(t, rejectSink.records, 2,
		"both messages should be sent to the reject sink")

	for _, rec := range rejectSink.records {
		assert.Equal(t, rejectReasonWritePermanent, rec.Reason)
	}
}

func TestWriteBatchCtxCancelledReturnsCtxErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	writer := &testWriter{
		flushErrs: []error{errors.New("some write error")},
	}

	output := &xatuClickHouseOutput{
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		rejectSink: &testRejectSink{},
	}

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 1),
	}

	err := output.WriteBatch(ctx, msgs)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWriteBatchUnknownFlushFailureFailsGroupMessages(t *testing.T) {
	// When flush returns an unknown failure, group processing produces a
	// BatchError marking all messages in the group as failed.
	writer := &testWriter{
		flushErrs: []error{errors.New("unknown failure")},
	}

	output := &xatuClickHouseOutput{
		log:      logrus.New(),
		encoding: "json",
		router: newRouter(t, []route.Route{
			testRoute{
				eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
				table:     "beacon_head",
			},
		}),
		writer:     writer,
		metrics:    newTestMetrics(),
		rejectSink: &testRejectSink{},
	}

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "evt-1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "topic-a", 0, 1),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)

	// Should be a BatchError with all messages marked as failed.
	failed := failedIndexesFromBatchError(t, msgs, err)
	assert.Equal(t, []int{0}, failed,
		"all messages in the batch should be marked as failed")
}

func TestKafkaTopicMetadataFallback(t *testing.T) {
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(nil))

	msg := service.NewMessage(nil)
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(msg))

	msg.MetaSet("kafka_topic", "xatu-mainnet")
	assert.Equal(t, "xatu-mainnet", kafkaTopicMetadata(msg))
}

func TestKafkaConfig_Validate_OutputBatch(t *testing.T) {
	validKafka := func() *KafkaConfig {
		return &KafkaConfig{
			Brokers:                []string{"kafka-1:9092"},
			Topics:                 []string{"^test-.+"},
			ConsumerGroup:          "xatu-consumoor",
			Encoding:               "json",
			FetchMinBytes:          1,
			FetchWaitMaxMs:         250,
			MaxPartitionFetchBytes: 1048576,
			FetchMaxBytes:          10485760,
			SessionTimeoutMs:       30000,
			OffsetDefault:          "earliest",
			CommitInterval:         5 * time.Second,
			ShutdownTimeout:        30 * time.Second,
			OutputBatchCount:       1000,
			OutputBatchPeriod:      1 * time.Second,
			MaxInFlight:            64,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*KafkaConfig)
		wantErr string
	}{
		{
			name:   "defaults valid",
			mutate: func(_ *KafkaConfig) {},
		},
		{
			name:   "zero count valid (opt-out)",
			mutate: func(c *KafkaConfig) { c.OutputBatchCount = 0 },
		},
		{
			name:   "zero period valid (opt-out)",
			mutate: func(c *KafkaConfig) { c.OutputBatchPeriod = 0 },
		},
		{
			name:    "negative count",
			mutate:  func(c *KafkaConfig) { c.OutputBatchCount = -1 },
			wantErr: "outputBatchCount must be >= 0",
		},
		{
			name:    "negative period",
			mutate:  func(c *KafkaConfig) { c.OutputBatchPeriod = -1 * time.Second },
			wantErr: "outputBatchPeriod must be >= 0",
		},
		{
			name:   "maxInFlight 1 valid",
			mutate: func(c *KafkaConfig) { c.MaxInFlight = 1 },
		},
		{
			name:    "maxInFlight 0 rejected",
			mutate:  func(c *KafkaConfig) { c.MaxInFlight = 0 },
			wantErr: "maxInFlight must be >= 1",
		},
		{
			name:    "maxInFlight negative rejected",
			mutate:  func(c *KafkaConfig) { c.MaxInFlight = -1 },
			wantErr: "maxInFlight must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validKafka()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestKafkaConfig_Validate_TopicOverrides(t *testing.T) {
	validKafka := func() *KafkaConfig {
		return &KafkaConfig{
			Brokers:                []string{"kafka-1:9092"},
			Topics:                 []string{"^test-.+"},
			ConsumerGroup:          "xatu-consumoor",
			Encoding:               "json",
			FetchMinBytes:          1,
			FetchWaitMaxMs:         250,
			MaxPartitionFetchBytes: 1048576,
			FetchMaxBytes:          10485760,
			SessionTimeoutMs:       30000,
			OffsetDefault:          "earliest",
			CommitInterval:         5 * time.Second,
			ShutdownTimeout:        30 * time.Second,
			OutputBatchCount:       1000,
			OutputBatchPeriod:      1 * time.Second,
			MaxInFlight:            64,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*KafkaConfig)
		wantErr string
	}{
		{
			name: "valid override passes",
			mutate: func(c *KafkaConfig) {
				c.TopicOverrides = map[string]TopicOverride{
					"test-blocks": {
						OutputBatchCount:  intPtr(100),
						OutputBatchPeriod: durPtr(100 * time.Millisecond),
						MaxInFlight:       intPtr(4),
					},
				}
			},
		},
		{
			name: "negative outputBatchCount in override rejected",
			mutate: func(c *KafkaConfig) {
				c.TopicOverrides = map[string]TopicOverride{
					"test-blocks": {OutputBatchCount: intPtr(-1)},
				}
			},
			wantErr: "kafka.topicOverrides.test-blocks: outputBatchCount must be >= 0",
		},
		{
			name: "negative outputBatchPeriod in override rejected",
			mutate: func(c *KafkaConfig) {
				c.TopicOverrides = map[string]TopicOverride{
					"test-blocks": {OutputBatchPeriod: durPtr(-1 * time.Second)},
				}
			},
			wantErr: "kafka.topicOverrides.test-blocks: outputBatchPeriod must be >= 0",
		},
		{
			name: "zero maxInFlight in override rejected",
			mutate: func(c *KafkaConfig) {
				c.TopicOverrides = map[string]TopicOverride{
					"test-blocks": {MaxInFlight: intPtr(0)},
				}
			},
			wantErr: "kafka.topicOverrides.test-blocks: maxInFlight must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validKafka()
			tt.mutate(cfg)

			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBenthosOutputBatchPolicy(t *testing.T) {
	tests := []struct {
		name       string
		count      int
		period     time.Duration
		wantCount  int
		wantPeriod string
	}{
		{
			name:       "defaults",
			count:      1000,
			period:     1 * time.Second,
			wantCount:  1000,
			wantPeriod: "1s",
		},
		{
			name:       "zero count disables count batching",
			count:      0,
			period:     1 * time.Second,
			wantCount:  0,
			wantPeriod: "1s",
		},
		{
			name:       "zero period disables period flushing",
			count:      500,
			period:     0,
			wantCount:  500,
			wantPeriod: "",
		},
		{
			name:       "both zero preserves opt-out",
			count:      0,
			period:     0,
			wantCount:  0,
			wantPeriod: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mirror the batch policy construction from NewBenthosStream.
			policy := service.BatchPolicy{}
			if tt.count > 0 {
				policy.Count = tt.count
			}

			if tt.period > 0 {
				policy.Period = tt.period.String()
			}

			assert.Equal(t, tt.wantCount, policy.Count)
			assert.Equal(t, tt.wantPeriod, policy.Period)
		})
	}
}

// mustNilEventJSON returns a valid JSON-encoded DecoratedEvent with a nil
// Event field. The router treats this as StatusRejected.
func mustNilEventJSON(t *testing.T) []byte {
	t.Helper()

	raw, err := protojson.Marshal(&xatu.DecoratedEvent{})
	require.NoError(t, err)

	return raw
}

// succeededIndexes returns the message indexes that are NOT marked as failed
// in the BatchError. When err is nil, all indexes are returned.
func succeededIndexes(msgs service.MessageBatch, err error) []int {
	failed := make(map[int]struct{}, len(msgs))

	var batchErr *service.BatchError
	if errors.As(err, &batchErr) {
		//nolint:staticcheck // Tests intentionally use direct indexes from this batch.
		batchErr.WalkMessages(func(idx int, _ *service.Message, msgErr error) bool {
			if msgErr != nil {
				failed[idx] = struct{}{}
			}

			return true
		})
	}

	out := make([]int, 0, len(msgs))

	for i := range msgs {
		if _, isFailed := failed[i]; !isFailed {
			out = append(out, i)
		}
	}

	return out
}

// ---------------------------------------------------------------------------
// At-Least-Once Delivery Guarantee Tests
// ---------------------------------------------------------------------------
//
// These tests verify the fundamental at-least-once delivery invariant:
//
//   A Kafka offset is committed (message ACK'd) if and only if the message
//   has been durably handled: successfully written to ClickHouse, sent to the
//   DLQ, or intentionally dropped by the routing layer.
//
//   Any other outcome (transient failures, DLQ write errors, context
//   cancellation) must cause WriteBatch to return an error, preventing
//   Benthos from committing the offset so Kafka redelivers the message.
//
// Tests are organized into three categories:
//
//   ack/   — WriteBatch returns nil; all messages are safe to commit.
//   nak/   — WriteBatch returns an error; impacted messages must be redelivered.
//   mixed/ — Some messages ACK, others NAK within the same batch.

func TestWriteBatchAtLeastOnceDelivery(t *testing.T) {
	headRoute := testRoute{
		eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
		table:     "beacon_head",
	}
	blockRoute := testRoute{
		eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
		table:     "beacon_block",
	}

	// -----------------------------------------------------------------
	// ACK cases: WriteBatch returns nil — all messages offset-committed.
	// -----------------------------------------------------------------
	t.Run("ack", func(t *testing.T) {
		t.Run("successful_flush_commits_all_offsets", func(t *testing.T) {
			writer := &testWriter{}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     writer,
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
				newKafkaMessage(mustEventJSON(t, "e3", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 3),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err, "successful flush must ACK all messages")
			assert.Equal(t, 1, writer.flushCalls)
			assert.Equal(t, 3, writer.writes["beacon_head"])
		})

		t.Run("decode_error_with_working_dlq_commits_offset", func(t *testing.T) {
			rejectSink := &testRejectSink{}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: rejectSink,
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("not-valid-json{"), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"decode error with working DLQ must ACK (message is durably in DLQ)")
			require.Len(t, rejectSink.records, 1)
			assert.Equal(t, rejectReasonDecode, rejectSink.records[0].Reason)
		})

		t.Run("permanent_write_error_with_working_dlq_commits_offset", func(t *testing.T) {
			rejectSink := &testRejectSink{}
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{
						&ch.Exception{
							Code:    proto.ErrUnknownTable,
							Name:    "DB::Exception",
							Message: "table gone",
						},
					},
				},
				metrics:    newTestMetrics(),
				rejectSink: rejectSink,
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"permanent write error with working DLQ must ACK (message is in DLQ)")
			require.Len(t, rejectSink.records, 1)
			assert.Equal(t, rejectReasonWritePermanent, rejectSink.records[0].Reason)
		})

		t.Run("unrouted_event_type_silently_commits", func(t *testing.T) {
			// Event type has no registered route. The router returns
			// StatusDelivered with empty Results, so the message is
			// silently skipped — safe to commit.
			writer := &testWriter{}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     writer,
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				// BLOCK event has no registered route (only HEAD is registered).
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"unrouted event types are intentionally dropped — safe to commit")
			assert.Equal(t, 0, writer.flushCalls,
				"no flush should occur for unrouted events")
		})

		t.Run("nil_event_rejected_with_working_dlq_commits", func(t *testing.T) {
			// A valid proto with nil Event field triggers StatusRejected
			// in the router. With a working DLQ, the message is sent
			// there and ACK'd.
			rejectSink := &testRejectSink{}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: rejectSink,
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustNilEventJSON(t), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"route-rejected with working DLQ must ACK")
			require.Len(t, rejectSink.records, 1)
			assert.Equal(t, rejectReasonRouteRejected, rejectSink.records[0].Reason)
		})

		t.Run("nil_event_rejected_without_dlq_still_commits", func(t *testing.T) {
			// Route rejections are intentional — the event type is not
			// configured. Even without a DLQ the message is ACK'd to
			// avoid infinite redelivery of permanently unroutable events.
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: nil, // no DLQ
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustNilEventJSON(t), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"route-rejected without DLQ must still ACK (intentional drop)")
		})

		t.Run("empty_batch_returns_nil", func(t *testing.T) {
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, nil),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			err := output.WriteBatch(context.Background(), service.MessageBatch{})
			require.NoError(t, err)
		})
	})

	// -----------------------------------------------------------------
	// NAK cases: WriteBatch returns error — messages must be redelivered.
	// -----------------------------------------------------------------
	t.Run("nak", func(t *testing.T) {
		t.Run("transient_flush_error_naks_impacted_messages", func(t *testing.T) {
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{
						&testWriteError{table: "beacon_head", permanent: false},
					},
				},
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0, 1}, failedIndexesFromBatchError(t, msgs, err),
				"transient flush failure must NAK all messages in the group")
		})

		t.Run("unknown_flush_error_naks_all_messages", func(t *testing.T) {
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{errors.New("unexpected infrastructure error")},
				},
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err),
				"unclassified flush error must NAK — cannot assume data was written")
		})

		t.Run("context_cancelled_before_processing_naks_batch", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
			}

			err := output.WriteBatch(ctx, msgs)
			require.Error(t, err)
			assert.ErrorIs(t, err, context.Canceled,
				"cancelled context must NAK the entire batch")
		})

		t.Run("decode_error_without_dlq_naks_for_redelivery", func(t *testing.T) {
			// When no DLQ is configured (nil rejectSink), decode errors
			// must NAK to avoid silent data loss. The message will be
			// redelivered by Kafka indefinitely.
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     &testWriter{},
				metrics:    newTestMetrics(),
				rejectSink: nil,
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("corrupt"), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err),
				"decode error without DLQ must NAK to prevent silent data loss")
		})

		t.Run("permanent_write_error_without_dlq_naks_for_redelivery", func(t *testing.T) {
			// Without a DLQ, even permanent write errors must NAK. The
			// message will be redelivered, which will fail again — but
			// this is preferable to silent data loss.
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{
						&ch.Exception{
							Code:    proto.ErrUnknownTable,
							Name:    "DB::Exception",
							Message: "table gone",
						},
					},
				},
				metrics:    newTestMetrics(),
				rejectSink: nil,
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err),
				"permanent write error without DLQ must NAK")
		})

		t.Run("decode_error_with_broken_dlq_naks", func(t *testing.T) {
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer:   &testWriter{},
				metrics:  newTestMetrics(),
				rejectSink: &testRejectSink{
					err: errors.New("kafka produce timeout"),
				},
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("{{{"), "t", 0, 1),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err),
				"DLQ write failure must NAK — message is not durably handled")
		})

		t.Run("permanent_write_error_with_broken_dlq_naks", func(t *testing.T) {
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{
						&ch.Exception{
							Code:    proto.ErrUnknownTable,
							Name:    "DB::Exception",
							Message: "table gone",
						},
					},
				},
				metrics: newTestMetrics(),
				rejectSink: &testRejectSink{
					err: errors.New("kafka produce timeout"),
				},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0, 1}, failedIndexesFromBatchError(t, msgs, err),
				"permanent error + DLQ failure must NAK all impacted messages")
		})
	})

	// -----------------------------------------------------------------
	// Mixed cases: some messages ACK, others NAK in the same batch.
	// -----------------------------------------------------------------
	t.Run("mixed", func(t *testing.T) {
		t.Run("valid_and_malformed_with_working_dlq_all_ack", func(t *testing.T) {
			// Malformed message goes to DLQ (ACK), valid message is
			// flushed to ClickHouse (ACK). Everything commits.
			writer := &testWriter{}
			rejectSink := &testRejectSink{}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute}),
				writer:     writer,
				metrics:    newTestMetrics(),
				rejectSink: rejectSink,
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("{bad"), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.NoError(t, err,
				"malformed→DLQ + valid→CH: both are durably handled, all ACK")
			assert.Equal(t, 1, writer.flushCalls)
			assert.Equal(t, 1, writer.writes["beacon_head"])
			require.Len(t, rejectSink.records, 1)
			assert.Equal(t, rejectReasonDecode, rejectSink.records[0].Reason)
		})

		t.Run("two_event_groups_one_transient_fail_only_that_group_naks", func(t *testing.T) {
			// Two event types in the same batch. One group flushes
			// successfully, the other hits a transient error. Only the
			// failed group's messages should be NAK'd.
			writer := &testWriter{
				flushErrs: []error{
					&testWriteError{table: "beacon_block", permanent: false},
				},
			}
			output := &xatuClickHouseOutput{
				log:        logrus.New(),
				encoding:   "json",
				router:     newRouter(t, []route.Route{headRoute, blockRoute}),
				writer:     writer,
				metrics:    newTestMetrics(),
				rejectSink: &testRejectSink{},
			}

			msgs := service.MessageBatch{
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK), "t", 0, 2),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err, "batch should fail because one group had a transient error")

			failed := failedIndexesFromBatchError(t, msgs, err)
			succeeded := succeededIndexes(msgs, err)

			// HEAD (index 0) should succeed, BLOCK (index 1) should fail.
			assert.Equal(t, []int{1}, failed,
				"only the BLOCK group (transient failure) should be NAK'd")
			assert.Equal(t, []int{0}, succeeded,
				"the HEAD group (successful flush) should be ACK'd")
		})

		t.Run("malformed_with_broken_dlq_naks_valid_messages_still_ack", func(t *testing.T) {
			// Decode error + DLQ failure → NAK that message.
			// Valid message → successful flush → ACK.
			writer := &testWriter{}
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer:   writer,
				metrics:  newTestMetrics(),
				rejectSink: &testRejectSink{
					err: errors.New("dlq unavailable"),
				},
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("{bad"), "t", 0, 1),
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)

			failed := failedIndexesFromBatchError(t, msgs, err)
			succeeded := succeededIndexes(msgs, err)

			assert.Equal(t, []int{0}, failed,
				"malformed message with DLQ failure must NAK")
			assert.Equal(t, []int{1}, succeeded,
				"valid message with successful flush must ACK")
		})

		t.Run("decode_dlq_failure_plus_flush_failure_naks_both", func(t *testing.T) {
			// Edge case: Phase 1 produces a NAK (decode error + DLQ
			// failure), AND Phase 2 produces a NAK (transient flush).
			// Both messages should be NAK'd.
			//
			// NOTE: WriteBatch currently overwrites the Phase 1
			// BatchError when Phase 2 fails, losing the decode
			// failure's NAK. This test documents the expected correct
			// behavior — if it fails, it reveals a bug where Phase 1
			// failures are silently dropped.
			output := &xatuClickHouseOutput{
				log:      logrus.New(),
				encoding: "json",
				router:   newRouter(t, []route.Route{headRoute}),
				writer: &testWriter{
					flushErrs: []error{
						&testWriteError{table: "beacon_head", permanent: false},
					},
				},
				metrics: newTestMetrics(),
				rejectSink: &testRejectSink{
					err: errors.New("dlq unavailable"),
				},
			}

			msgs := service.MessageBatch{
				newKafkaMessage([]byte("{bad"), "t", 0, 1),                                                   // decode error → DLQ fails
				newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2), // flush fails
			}

			err := output.WriteBatch(context.Background(), msgs)
			require.Error(t, err)
			assert.Equal(t, []int{0, 1}, failedIndexesFromBatchError(t, msgs, err),
				"both the decode-DLQ-failed message and flush-failed message must NAK")
		})
	})
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
