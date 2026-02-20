package kafkatopicrouter

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/output/kafka"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			event := &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: tt.eventName,
				},
			}

			got := resolveTopic(tt.pattern, event)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "missing brokers and topic pattern",
			config:  Config{},
			wantErr: "brokers is required",
		},
		{
			name: "missing topic pattern",
			config: Config{
				ProducerConfig: kafka.ProducerConfig{
					Brokers: "localhost:9092",
				},
			},
			wantErr: "topicPattern is required",
		},
		{
			name: "valid config",
			config: Config{
				ProducerConfig: kafka.ProducerConfig{
					Brokers: "localhost:9092",
				},
				TopicPattern: "xatu-${event-name}",
			},
		},
		{
			name: "valid config with static topic",
			config: Config{
				ProducerConfig: kafka.ProducerConfig{
					Brokers: "localhost:9092",
				},
				TopicPattern: "static-topic",
			},
		},
		{
			name: "producer config validation propagates",
			config: Config{
				ProducerConfig: kafka.ProducerConfig{
					Brokers: "localhost:9092",
					TLSClientConfig: &kafka.TLSClientConfig{
						CertificatePath: "/cert",
					},
				},
				TopicPattern: "xatu-${event-name}",
			},
			wantErr: "client key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
