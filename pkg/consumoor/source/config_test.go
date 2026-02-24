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
		CommitInterval:   5 * time.Second,
		ShutdownTimeout:  30 * time.Second,
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
