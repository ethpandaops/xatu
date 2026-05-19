package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// mockProducer implements sarama.SyncProducer for testing.
type mockProducer struct {
	mu       sync.Mutex
	messages []*sarama.ProducerMessage
	err      error
	closed   bool
}

func (m *mockProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msg)

	return 0, 0, m.err
}

func (m *mockProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msgs...)

	return m.err
}

func (m *mockProducer) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true

	return nil
}

func (m *mockProducer) snapshotMessages() []*sarama.ProducerMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]*sarama.ProducerMessage(nil), m.messages...)
}

func (m *mockProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return 0
}

func (m *mockProducer) IsTransactional() bool { return false }
func (m *mockProducer) BeginTxn() error       { return nil }
func (m *mockProducer) CommitTxn() error      { return nil }
func (m *mockProducer) AbortTxn() error       { return nil }

func (m *mockProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return nil
}

func (m *mockProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return nil
}

func newTestLogger() observability.ContextualLogger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)

	return l
}

func newTestEvent(name xatu.Event_Name, id string) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: name,
			Id:   id,
		},
	}
}

// --- resolveTopic tests ---

func TestResolveTopic(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		eventName xatu.Event_Name
		want      string
	}{
		{
			name:      "kebab-case substitution",
			pattern:   "xatu-${event-name}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "xatu-beacon-api-eth-v1-events-block",
		},
		{
			name:      "SCREAMING_SNAKE substitution",
			pattern:   "xatu-${EVENT_NAME}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "xatu-BEACON_API_ETH_V1_EVENTS_BLOCK",
		},
		{
			name:      "both substitutions",
			pattern:   "${EVENT_NAME}-as-${event-name}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			want:      "BEACON_API_ETH_V1_EVENTS_HEAD-as-beacon-api-eth-v1-events-head",
		},
		{
			name:      "no variables (static topic)",
			pattern:   "static-topic",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "static-topic",
		},
		{
			name:      "prefix and suffix around kebab",
			pattern:   "prefix-${event-name}-suffix",
			eventName: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			want:      "prefix-beacon-api-eth-v2-beacon-block-suffix",
		},
		{
			name:      "empty pattern",
			pattern:   "",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "",
		},
		{
			name:      "unknown event name uses numeric string",
			pattern:   "xatu-${event-name}",
			eventName: xatu.Event_Name(99999),
			want:      "xatu-99999",
		},
		{
			name:      "multiple kebab variables",
			pattern:   "${event-name}/${event-name}",
			eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			want:      "beacon-api-eth-v1-events-block/beacon-api-eth-v1-events-block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := newTestEvent(tt.eventName, "test-id")
			got := resolveTopic(tt.pattern, event)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- topicForEvent tests ---

func TestTopicForEvent(t *testing.T) {
	t.Run("static topic", func(t *testing.T) {
		e := &ItemExporter{
			config: &Config{Topic: "my-static-topic"},
		}
		event := newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")
		assert.Equal(t, "my-static-topic", e.topicForEvent(event))
	})

	t.Run("dynamic topic pattern", func(t *testing.T) {
		e := &ItemExporter{
			config: &Config{TopicPattern: "xatu-${event-name}"},
		}
		event := newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")
		assert.Equal(t, "xatu-beacon-api-eth-v1-events-block", e.topicForEvent(event))
	})
}

// --- ExportItems tests ---

func TestExportItemsStaticTopic(t *testing.T) {
	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "static-topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.NoError(t, err)
	require.Len(t, mock.messages, 2)

	for _, msg := range mock.messages {
		assert.Equal(t, "static-topic", msg.Topic)
	}

	key0, _ := mock.messages[0].Key.Encode()
	assert.Equal(t, "id-1", string(key0))

	key1, _ := mock.messages[1].Key.Encode()
	assert.Equal(t, "id-2", string(key1))
}

func TestExportItemsTopicPattern(t *testing.T) {
	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		TopicPattern:   "xatu-${event-name}",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.NoError(t, err)
	require.Len(t, mock.messages, 2)

	assert.Equal(t, "xatu-beacon-api-eth-v1-events-block", mock.messages[0].Topic)
	assert.Equal(t, "xatu-beacon-api-eth-v1-events-head", mock.messages[1].Topic)
}

func TestExportItemsOversizedMessageSkipped(t *testing.T) {
	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1},
		Topic:          "topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.NoError(t, err)
	assert.Empty(t, mock.messages, "oversized messages should be skipped")
}

func TestExportItemsEmptyBatchAfterFiltering(t *testing.T) {
	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1},
		Topic:          "topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.NoError(t, err)
	assert.Empty(t, mock.messages, "no SendMessages call when all messages are oversized")
}

func TestExportItemsDoesNotPropagateBatchSpan(t *testing.T) {
	setupKafkaTraceTest(t)

	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.NoError(t, err)
	require.Len(t, mock.messages, 2)

	for _, msg := range mock.messages {
		assert.False(t, spanContextFromHeaders(msg.Headers).IsValid(), "batch exporter span must not be injected into Kafka records")
	}
}

func TestExportItemsPropagatesCallerContext(t *testing.T) {
	setupKafkaTraceTest(t)

	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	ctx, parent := otel.Tracer("test").Start(context.Background(), "caller")
	defer parent.End()

	err := exporter.ExportItems(ctx, []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
	})
	require.NoError(t, err)
	require.Len(t, mock.messages, 1)

	sc := spanContextFromHeaders(mock.messages[0].Headers)
	require.True(t, sc.IsValid())
	assert.Equal(t, parent.SpanContext().TraceID(), sc.TraceID())
	assert.Equal(t, parent.SpanContext().SpanID(), sc.SpanID())
}

func TestKafkaSinkProcessorPropagatesPerItemContexts(t *testing.T) {
	setupKafkaTraceTest(t)

	const name = "test"

	mock := &mockProducer{}
	exporter := NewItemExporter(name, &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	proc, err := processor.NewBatchItemProcessor[xatu.DecoratedEvent](
		exporter,
		"processor",
		newTestLogger(),
		processor.WithMaxExportBatchSize(2),
		processor.WithWorkers(1),
		processor.WithBatchTimeout(time.Hour),
	)
	require.NoError(t, err)

	filter, err := xatu.NewEventFilter(&xatu.EventFilterConfig{})
	require.NoError(t, err)

	sink := &Kafka{
		name:   name,
		config: &Config{},
		log:    newTestLogger(),
		proc:   proc,
		filter: filter,
	}

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	ctx1, span1 := otel.Tracer(name).Start(context.Background(), "event-1")
	defer span1.End()

	ctx2, span2 := otel.Tracer(name).Start(context.Background(), "event-2")
	defer span2.End()

	require.NoError(t, sink.HandleNewDecoratedEvent(ctx1, newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")))
	require.NoError(t, sink.HandleNewDecoratedEvent(ctx2, newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2")))

	require.Eventually(t, func() bool {
		return len(mock.snapshotMessages()) == 2
	}, time.Second, 10*time.Millisecond)

	messages := mock.snapshotMessages()
	sc1 := spanContextFromHeaders(messages[0].Headers)
	sc2 := spanContextFromHeaders(messages[1].Headers)

	require.True(t, sc1.IsValid())
	require.True(t, sc2.IsValid())
	assert.Equal(t, span1.SpanContext().TraceID(), sc1.TraceID())
	assert.Equal(t, span1.SpanContext().SpanID(), sc1.SpanID())
	assert.Equal(t, span2.SpanContext().TraceID(), sc2.TraceID())
	assert.Equal(t, span2.SpanContext().SpanID(), sc2.SpanID())
	assert.NotEqual(t, sc1.TraceID(), sc2.TraceID())
}

func TestKafkaSinkProcessorNoActiveSpanDoesNotInjectTraceparent(t *testing.T) {
	setupKafkaTraceTest(t)

	const name = "test"

	mock := &mockProducer{}
	exporter := NewItemExporter(name, &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	proc, err := processor.NewBatchItemProcessor[xatu.DecoratedEvent](
		exporter,
		"processor",
		newTestLogger(),
		processor.WithMaxExportBatchSize(1),
		processor.WithWorkers(1),
		processor.WithBatchTimeout(time.Hour),
	)
	require.NoError(t, err)

	filter, err := xatu.NewEventFilter(&xatu.EventFilterConfig{})
	require.NoError(t, err)

	sink := &Kafka{
		name:   name,
		config: &Config{},
		log:    newTestLogger(),
		proc:   proc,
		filter: filter,
	}

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	require.NoError(t, sink.HandleNewDecoratedEvent(context.Background(), newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")))

	require.Eventually(t, func() bool {
		return len(mock.snapshotMessages()) == 1
	}, time.Second, 10*time.Millisecond)

	messages := mock.snapshotMessages()
	assert.False(t, spanContextFromHeaders(messages[0].Headers).IsValid())
}

func TestExportItemsProducerErrors(t *testing.T) {
	prodErr := &sarama.ProducerError{
		Msg: &sarama.ProducerMessage{Topic: "t"},
		Err: errors.New("broker down"),
	}
	mock := &mockProducer{
		err: sarama.ProducerErrors{prodErr},
	}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.Error(t, err)

	var pe *sarama.ProducerError
	require.True(t, errors.As(err, &pe), "error should unwrap to *sarama.ProducerError")
	assert.Equal(t, "broker down", pe.Err.Error())
}

func TestExportItemsGenericError(t *testing.T) {
	mock := &mockProducer{
		err: errors.New("connection refused"),
	}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	events := []*xatu.DecoratedEvent{
		newTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1"),
	}

	err := exporter.ExportItems(context.Background(), events)
	require.EqualError(t, err, "connection refused")
}

func TestShutdownClosesProducer(t *testing.T) {
	mock := &mockProducer{}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{MaxMessageBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	err := exporter.Shutdown(context.Background())
	require.NoError(t, err)
	assert.True(t, mock.closed, "Shutdown should call Close on the producer")
}

func setupKafkaTraceTest(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	return recorder
}

func spanContextFromHeaders(headers []sarama.RecordHeader) trace.SpanContext {
	return trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(
			context.Background(),
			newSaramaHeaderCarrier(&headers),
		),
	)
}
