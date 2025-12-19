package ethstats_test

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/auth"
	"github.com/ethpandaops/xatu/pkg/ethstats"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	validOutput := output.Config{
		Name:     "test-output",
		SinkType: "stdout",
	}

	tests := []struct {
		name    string
		config  ethstats.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid minimal config",
			config: ethstats.Config{
				Name:    "test-ethstats",
				Addr:    ":3000",
				Outputs: []output.Config{validOutput},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			config: ethstats.Config{
				Name:    "",
				Addr:    ":3000",
				Outputs: []output.Config{validOutput},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Missing addr",
			config: ethstats.Config{
				Name:    "test-ethstats",
				Addr:    "",
				Outputs: []output.Config{validOutput},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "addr is required",
		},
		{
			name: "Missing outputs",
			config: ethstats.Config{
				Name:    "test-ethstats",
				Addr:    ":3000",
				Outputs: []output.Config{},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "at least one output is required",
		},
		{
			name: "Valid config with authorization",
			config: ethstats.Config{
				Name:    "test-ethstats",
				Addr:    ":3000",
				Outputs: []output.Config{validOutput},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
				Authorization: auth.AuthorizationConfig{
					Enabled: true,
					Groups: auth.GroupsConfig{
						"group1": {
							Users: auth.UsersConfig{
								"user1": {Password: "password1"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid config with labels",
			config: ethstats.Config{
				Name:    "test-ethstats",
				Addr:    ":3000",
				Outputs: []output.Config{validOutput},
				WebSocket: ethstats.WebSocketConfig{
					ReadBufferSize:  1024,
					WriteBufferSize: 1024,
					PingInterval:    15 * time.Second,
					ReadLimit:       15728640,
					WriteWait:       10 * time.Second,
				},
				Labels: map[string]string{
					"environment": "test",
					"region":      "us-east-1",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWebSocketConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ethstats.WebSocketConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				PingInterval:    15 * time.Second,
				ReadLimit:       15728640,
				WriteWait:       10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Invalid readBufferSize (zero)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  0,
				WriteBufferSize: 1024,
				PingInterval:    15 * time.Second,
				ReadLimit:       15728640,
				WriteWait:       10 * time.Second,
			},
			wantErr: true,
			errMsg:  "readBufferSize must be positive",
		},
		{
			name: "Invalid readBufferSize (negative)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  -1,
				WriteBufferSize: 1024,
				PingInterval:    15 * time.Second,
				ReadLimit:       15728640,
				WriteWait:       10 * time.Second,
			},
			wantErr: true,
			errMsg:  "readBufferSize must be positive",
		},
		{
			name: "Invalid writeBufferSize (zero)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 0,
				PingInterval:    15 * time.Second,
				ReadLimit:       15728640,
				WriteWait:       10 * time.Second,
			},
			wantErr: true,
			errMsg:  "writeBufferSize must be positive",
		},
		{
			name: "Invalid pingInterval (zero)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				PingInterval:    0,
				ReadLimit:       15728640,
				WriteWait:       10 * time.Second,
			},
			wantErr: true,
			errMsg:  "pingInterval must be positive",
		},
		{
			name: "Invalid readLimit (zero)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				PingInterval:    15 * time.Second,
				ReadLimit:       0,
				WriteWait:       10 * time.Second,
			},
			wantErr: true,
			errMsg:  "readLimit must be positive",
		},
		{
			name: "Invalid writeWait (zero)",
			config: ethstats.WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				PingInterval:    15 * time.Second,
				ReadLimit:       15728640,
				WriteWait:       0,
			},
			wantErr: true,
			errMsg:  "writeWait must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ApplyDefaults(t *testing.T) {
	config := &ethstats.Config{}
	err := config.ApplyDefaults()
	require.NoError(t, err)

	// Check defaults were applied
	assert.Equal(t, "info", config.LoggingLevel)
	assert.Equal(t, ":9090", config.MetricsAddr)
	assert.Equal(t, ":3000", config.Addr)
	assert.Equal(t, "time.google.com", config.NTPServer)

	// WebSocket defaults
	assert.Equal(t, 1024, config.WebSocket.ReadBufferSize)
	assert.Equal(t, 1024, config.WebSocket.WriteBufferSize)
	assert.Equal(t, 15*time.Second, config.WebSocket.PingInterval)
	assert.Equal(t, int64(15728640), config.WebSocket.ReadLimit)
	assert.Equal(t, 10*time.Second, config.WebSocket.WriteWait)
}
