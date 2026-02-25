package source

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validKafkaConfig() KafkaConfig {
	return KafkaConfig{
		Brokers:          []string{"localhost:9092"},
		Topics:           []string{"test-topic"},
		ConsumerGroup:    "test-group",
		Encoding:         "json",
		OffsetDefault:    "earliest",
		SessionTimeoutMs: 30000,
		RebalanceTimeout: 15 * time.Second,
		CommitInterval:   5 * time.Second,
		ShutdownTimeout:  30 * time.Second,
		MaxInFlight:      64,
	}
}

func intPtr(v int) *int                     { return &v }
func durPtr(v time.Duration) *time.Duration { return &v }

func TestKafkaConfig_ApplyTopicOverride(t *testing.T) {
	tests := []struct {
		name       string
		overrides  map[string]TopicOverride
		topic      string
		wantCount  int
		wantPeriod time.Duration
		wantFlight int
	}{
		{
			name:       "no override returns unchanged copy",
			overrides:  nil,
			topic:      "general-beacon-block",
			wantCount:  10000,
			wantPeriod: 1 * time.Second,
			wantFlight: 64,
		},
		{
			name: "partial override (only count)",
			overrides: map[string]TopicOverride{
				"general-beacon-block": {
					OutputBatchCount: intPtr(100),
				},
			},
			topic:      "general-beacon-block",
			wantCount:  100,
			wantPeriod: 1 * time.Second,
			wantFlight: 64,
		},
		{
			name: "full override",
			overrides: map[string]TopicOverride{
				"general-beacon-block": {
					OutputBatchCount:  intPtr(500),
					OutputBatchPeriod: durPtr(200 * time.Millisecond),
					MaxInFlight:       intPtr(4),
				},
			},
			topic:      "general-beacon-block",
			wantCount:  500,
			wantPeriod: 200 * time.Millisecond,
			wantFlight: 4,
		},
		{
			name: "explicit zero count",
			overrides: map[string]TopicOverride{
				"general-beacon-block": {
					OutputBatchCount: intPtr(0),
				},
			},
			topic:      "general-beacon-block",
			wantCount:  0,
			wantPeriod: 1 * time.Second,
			wantFlight: 64,
		},
		{
			name: "unmatched topic returns unchanged copy",
			overrides: map[string]TopicOverride{
				"general-beacon-attestation": {
					OutputBatchCount: intPtr(50000),
				},
			},
			topic:      "general-beacon-block",
			wantCount:  10000,
			wantPeriod: 1 * time.Second,
			wantFlight: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validKafkaConfig()
			cfg.OutputBatchCount = 10000
			cfg.OutputBatchPeriod = 1 * time.Second
			cfg.TopicOverrides = tt.overrides

			got := cfg.ApplyTopicOverride(tt.topic)

			assert.Equal(t, tt.wantCount, got.OutputBatchCount)
			assert.Equal(t, tt.wantPeriod, got.OutputBatchPeriod)
			assert.Equal(t, tt.wantFlight, got.MaxInFlight)
		})
	}
}

func TestKafkaConfig_Validate_ShutdownTimeout(t *testing.T) {
	tests := []struct {
		name            string
		shutdownTimeout time.Duration
		wantErr         string
	}{
		{
			name:            "valid",
			shutdownTimeout: 30 * time.Second,
		},
		{
			name:            "small positive",
			shutdownTimeout: 1 * time.Second,
		},
		{
			name:            "zero",
			shutdownTimeout: 0,
			wantErr:         "shutdownTimeout must be > 0",
		},
		{
			name:            "negative",
			shutdownTimeout: -1 * time.Second,
			wantErr:         "shutdownTimeout must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validKafkaConfig()
			cfg.ShutdownTimeout = tt.shutdownTimeout

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
