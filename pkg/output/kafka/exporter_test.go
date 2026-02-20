package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/IBM/sarama"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProducer implements sarama.SyncProducer for testing.
type mockProducer struct {
	messages []*sarama.ProducerMessage
	err      error
	closed   bool
}

func (m *mockProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	m.messages = append(m.messages, msg)

	return 0, 0, m.err
}

func (m *mockProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	m.messages = append(m.messages, msgs...)

	return m.err
}

func (m *mockProducer) Close() error {
	m.closed = true

	return nil
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

func newTestLogger() logrus.FieldLogger {
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

// --- Config.Validate tests ---

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "static topic only",
			config: Config{
				ProducerConfig: ProducerConfig{Brokers: "localhost:9092"},
				Topic:          "my-topic",
			},
		},
		{
			name: "topic pattern only",
			config: Config{
				ProducerConfig: ProducerConfig{Brokers: "localhost:9092"},
				TopicPattern:   "xatu-${event-name}",
			},
		},
		{
			name: "neither topic nor pattern",
			config: Config{
				ProducerConfig: ProducerConfig{Brokers: "localhost:9092"},
			},
			wantErr: "exactly one of topic or topicPattern is required",
		},
		{
			name: "both topic and pattern",
			config: Config{
				ProducerConfig: ProducerConfig{Brokers: "localhost:9092"},
				Topic:          "my-topic",
				TopicPattern:   "xatu-${event-name}",
			},
			wantErr: "topic and topicPattern are mutually exclusive",
		},
		{
			name: "missing brokers",
			config: Config{
				Topic: "my-topic",
			},
			wantErr: "brokers is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
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
		ProducerConfig: ProducerConfig{FlushBytes: 1000000},
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
		ProducerConfig: ProducerConfig{FlushBytes: 1000000},
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
		ProducerConfig: ProducerConfig{FlushBytes: 1},
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
		ProducerConfig: ProducerConfig{FlushBytes: 1},
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

func TestExportItemsProducerErrors(t *testing.T) {
	prodErr := &sarama.ProducerError{
		Msg: &sarama.ProducerMessage{Topic: "t"},
		Err: errors.New("broker down"),
	}
	mock := &mockProducer{
		err: sarama.ProducerErrors{prodErr},
	}
	exporter := NewItemExporter("test", &Config{
		ProducerConfig: ProducerConfig{FlushBytes: 1000000},
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
		ProducerConfig: ProducerConfig{FlushBytes: 1000000},
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
		ProducerConfig: ProducerConfig{FlushBytes: 1000000},
		Topic:          "topic",
	}, newTestLogger(), mock)

	err := exporter.Shutdown(context.Background())
	require.NoError(t, err)
	assert.True(t, mock.closed, "Shutdown should call Close on the producer")
}
