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

func TestAdaptiveLimiterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AdaptiveLimiterConfig
		wantErr string
	}{
		{
			name: "disabled passes",
			cfg:  AdaptiveLimiterConfig{Enabled: false},
		},
		{
			name: "valid enabled",
			cfg: AdaptiveLimiterConfig{
				Enabled:                     true,
				MinLimit:                    1,
				MaxLimit:                    50,
				InitialLimit:                8,
				QueueInitialRejectionFactor: 2,
				QueueMaxRejectionFactor:     3,
			},
		},
		{
			name: "zero minLimit",
			cfg: AdaptiveLimiterConfig{
				Enabled:      true,
				MinLimit:     0,
				MaxLimit:     50,
				InitialLimit: 8,
			},
			wantErr: "minLimit must be > 0",
		},
		{
			name: "zero maxLimit",
			cfg: AdaptiveLimiterConfig{
				Enabled:      true,
				MinLimit:     1,
				MaxLimit:     0,
				InitialLimit: 1,
			},
			wantErr: "maxLimit must be > 0",
		},
		{
			name: "minLimit exceeds maxLimit",
			cfg: AdaptiveLimiterConfig{
				Enabled:      true,
				MinLimit:     10,
				MaxLimit:     5,
				InitialLimit: 5,
			},
			wantErr: "minLimit must be <= maxLimit",
		},
		{
			name: "initialLimit below minLimit",
			cfg: AdaptiveLimiterConfig{
				Enabled:      true,
				MinLimit:     5,
				MaxLimit:     50,
				InitialLimit: 2,
			},
			wantErr: "initialLimit must be between minLimit and maxLimit",
		},
		{
			name: "initialLimit above maxLimit",
			cfg: AdaptiveLimiterConfig{
				Enabled:      true,
				MinLimit:     1,
				MaxLimit:     10,
				InitialLimit: 20,
			},
			wantErr: "initialLimit must be between minLimit and maxLimit",
		},
		{
			name: "zero queueInitialRejectionFactor",
			cfg: AdaptiveLimiterConfig{
				Enabled:                     true,
				MinLimit:                    1,
				MaxLimit:                    50,
				InitialLimit:                8,
				QueueInitialRejectionFactor: 0,
				QueueMaxRejectionFactor:     3,
			},
			wantErr: "queueInitialRejectionFactor must be > 0",
		},
		{
			name: "zero queueMaxRejectionFactor",
			cfg: AdaptiveLimiterConfig{
				Enabled:                     true,
				MinLimit:                    1,
				MaxLimit:                    50,
				InitialLimit:                8,
				QueueInitialRejectionFactor: 2,
				QueueMaxRejectionFactor:     0,
			},
			wantErr: "queueMaxRejectionFactor must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
