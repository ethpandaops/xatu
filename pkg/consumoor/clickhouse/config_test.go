package clickhouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() Config {
	return Config{
		DSN:                   "clickhouse://localhost:9000/default",
		OrganicRetryInitDelay: 1 * time.Second,
		OrganicRetryMaxDelay:  30 * time.Second,
		DrainTimeout:          30 * time.Second,
		Defaults: TableConfig{
			BatchSize:     200000,
			FlushInterval: 1 * time.Second,
			BufferSize:    200000,
		},
		ChGo: validChGoConfig(),
	}
}

func validChGoConfig() ChGoConfig {
	return ChGoConfig{
		DialTimeout:         5 * time.Second,
		ReadTimeout:         30 * time.Second,
		QueryTimeout:        30 * time.Second,
		MaxRetries:          3,
		RetryBaseDelay:      100 * time.Millisecond,
		RetryMaxDelay:       2 * time.Second,
		MaxConns:            32,
		MinConns:            1,
		ConnMaxLifetime:     1 * time.Hour,
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckPeriod:   30 * time.Second,
		PoolMetricsInterval: 15 * time.Second,
	}
}

func TestChGoConfig_Validate_DialTimeout(t *testing.T) {
	tests := []struct {
		name        string
		dialTimeout time.Duration
		wantErr     string
	}{
		{
			name:        "valid",
			dialTimeout: 5 * time.Second,
		},
		{
			name:        "zero",
			dialTimeout: 0,
			wantErr:     "dialTimeout must be > 0",
		},
		{
			name:        "negative",
			dialTimeout: -1 * time.Second,
			wantErr:     "dialTimeout must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validChGoConfig()
			cfg.DialTimeout = tt.dialTimeout

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

func TestChGoConfig_Validate_ReadTimeout(t *testing.T) {
	tests := []struct {
		name        string
		readTimeout time.Duration
		wantErr     string
	}{
		{
			name:        "valid",
			readTimeout: 30 * time.Second,
		},
		{
			name:        "zero",
			readTimeout: 0,
			wantErr:     "readTimeout must be > 0",
		},
		{
			name:        "negative",
			readTimeout: -1 * time.Second,
			wantErr:     "readTimeout must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validChGoConfig()
			cfg.ReadTimeout = tt.readTimeout

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

func TestConfig_Validate_OrganicRetry(t *testing.T) {
	tests := []struct {
		name      string
		initDelay time.Duration
		maxDelay  time.Duration
		wantErr   string
	}{
		{
			name:      "valid",
			initDelay: 1 * time.Second,
			maxDelay:  30 * time.Second,
		},
		{
			name:      "equal init and max",
			initDelay: 5 * time.Second,
			maxDelay:  5 * time.Second,
		},
		{
			name:      "zero init delay",
			initDelay: 0,
			maxDelay:  30 * time.Second,
			wantErr:   "organicRetryInitDelay must be > 0",
		},
		{
			name:      "negative init delay",
			initDelay: -1 * time.Second,
			maxDelay:  30 * time.Second,
			wantErr:   "organicRetryInitDelay must be > 0",
		},
		{
			name:      "zero max delay",
			initDelay: 1 * time.Second,
			maxDelay:  0,
			wantErr:   "organicRetryMaxDelay must be > 0",
		},
		{
			name:      "init exceeds max",
			initDelay: 30 * time.Second,
			maxDelay:  1 * time.Second,
			wantErr:   "organicRetryInitDelay must be <= organicRetryMaxDelay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.OrganicRetryInitDelay = tt.initDelay
			cfg.OrganicRetryMaxDelay = tt.maxDelay

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
