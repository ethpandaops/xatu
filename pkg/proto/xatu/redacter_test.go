package xatu

import (
	"testing"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestRedacter_Apply(t *testing.T) {
	tests := []struct {
		name                  string
		config                *RedacterConfig
		input                 *DecoratedEvent
		expected              *DecoratedEvent
		expectedRemovedFields []string
	}{
		{
			name: "Redact single field",
			config: &RedacterConfig{
				FieldPaths: []string{"meta.server.client.IP"},
			},
			input: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Client: &ServerMeta_Client{
							IP: "192.168.1.1",
							Geo: &ServerMeta_Geo{
								City:    "London",
								Country: "UK",
							},
						},
					},
				},
			},
			expected: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Client: &ServerMeta_Client{
							IP: "",
							Geo: &ServerMeta_Geo{
								City:    "London",
								Country: "UK",
							},
						},
					},
				},
			},
			expectedRemovedFields: []string{"meta.server.client.IP"},
		},
		{
			name: "Redact multiple fields",
			config: &RedacterConfig{
				FieldPaths: []string{"meta.server.event.received_date_time", "meta.server.client.IP"},
			},
			input: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Event: &ServerMeta_Event{
							ReceivedDateTime: &timestamp.Timestamp{
								Seconds: 1714857600,
							},
						},
						Client: &ServerMeta_Client{
							Geo: &ServerMeta_Geo{
								City:    "London",
								Country: "UK",
							},
							IP: "192.168.1.1",
						},
					},
				},
			},
			expected: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Event: &ServerMeta_Event{},
						Client: &ServerMeta_Client{
							IP: "",
							Geo: &ServerMeta_Geo{
								City:    "London",
								Country: "UK",
							},
						},
					},
				},
			},
			expectedRemovedFields: []string{"meta.server.event.received_date_time", "meta.server.client.IP"},
		},
		{
			name: "Redact non-existent field",
			config: &RedacterConfig{
				FieldPaths: []string{"meta.server.nonexistent"},
			},
			input: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Event: &ServerMeta_Event{
							ReceivedDateTime: &timestamp.Timestamp{
								Seconds: 1714857600,
							},
						},
					},
				},
			},
			expected: &DecoratedEvent{
				Meta: &Meta{
					Server: &ServerMeta{
						Event: &ServerMeta_Event{
							ReceivedDateTime: &timestamp.Timestamp{
								Seconds: 1714857600,
							},
						},
					},
				},
			},
			expectedRemovedFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redacter, err := NewRedacter(tt.config)
			assert.NoError(t, err)

			removedFields := redacter.Apply(tt.input)

			if !proto.Equal(tt.expected, tt.input) {
				t.Errorf("Expected %v, but got %v", tt.expected, tt.input)
			}

			assert.Equal(t, tt.expectedRemovedFields, removedFields)
		})
	}
}
