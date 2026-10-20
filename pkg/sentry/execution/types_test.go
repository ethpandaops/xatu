package execution

import (
	"testing"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(c *Config)
		wantErr string
	}{
		{
			name:   "defaults with an rpc address",
			modify: func(*Config) {},
		},
		{
			name:    "missing rpc address",
			modify:  func(c *Config) { c.RPCAddress = "" },
			wantErr: "rpcAddress is required",
		},
		{
			name:    "websocket without an address",
			modify:  func(c *Config) { c.WebsocketEnabled = true },
			wantErr: "wsAddress is required",
		},
		{
			name: "unknown state size trigger mode",
			modify: func(c *Config) {
				c.StateSize.Enabled = true
				c.StateSize.TriggerMode = "slot"
			},
			wantErr: "triggerMode must be",
		},
		{
			name: "non-positive state size interval",
			modify: func(c *Config) {
				c.StateSize.Enabled = true
				c.StateSize.TriggerMode = StateSizeTriggerInterval
				c.StateSize.IntervalSeconds = 0
			},
			wantErr: "intervalSeconds must be positive",
		},
		{
			name: "block trigger without a websocket",
			modify: func(c *Config) {
				c.StateSize.Enabled = true
				c.StateSize.TriggerMode = StateSizeTriggerBlock
			},
			wantErr: "needs websocketEnabled",
		},
		{
			name: "block trigger with a websocket",
			modify: func(c *Config) {
				c.WebsocketEnabled = true
				c.WSAddress = "ws://localhost:8546"
				c.StateSize.Enabled = true
				c.StateSize.TriggerMode = StateSizeTriggerBlock
			},
		},
		{
			name: "disabled state size is not checked",
			modify: func(c *Config) {
				c.StateSize.TriggerMode = "slot"
				c.StateSize.IntervalSeconds = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			require.NoError(t, defaults.Set(config))

			config.Enabled = true
			config.RPCAddress = "http://localhost:8545"
			tt.modify(config)

			err := config.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
