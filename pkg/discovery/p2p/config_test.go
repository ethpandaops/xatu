package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/ethpandaops/xatu/pkg/internal/configdecode"
)

func TestGetExecutionConfigDefaults(t *testing.T) {
	cfg := &Config{}

	exec := cfg.GetExecutionConfig()

	require.Equal(t, uint(1), exec.RetryAttempts)
	require.Equal(t, 60*time.Second, exec.RetryDelay)
	require.Equal(t, 15*time.Second, exec.DialTimeout)
}

func TestGetExecutionConfigPreservesExplicitValues(t *testing.T) {
	for _, p2pType := range []Type{TypeStatic, TypeXatu} {
		t.Run(string(p2pType), func(t *testing.T) {
			cfg := p2pConfig(t, p2pType, "  execution:\n    retryAttempts: 3\n    retryDelay: 5s\n    dialTimeout: 10s\n")

			exec := cfg.GetExecutionConfig()

			require.Equal(t, uint(3), exec.RetryAttempts)
			require.Equal(t, 5*time.Second, exec.RetryDelay)
			require.Equal(t, 10*time.Second, exec.DialTimeout)
		})
	}
}

func TestGetConsensusConfigDefaultsOmittedFields(t *testing.T) {
	for _, p2pType := range []Type{TypeStatic, TypeXatu} {
		t.Run(string(p2pType), func(t *testing.T) {
			cfg := p2pConfig(t, p2pType, "  consensus:\n    dialConcurrency: 20\n    retryDelay: 0s\n")

			cons := cfg.GetConsensusConfig()

			require.Equal(t, 20, cons.DialConcurrency)
			require.Equal(t, time.Duration(0), cons.RetryDelay)
			require.Equal(t, 1, cons.RetryAttempts)
			require.Equal(t, 5*time.Second, cons.DialTimeout)
			require.Equal(t, 10*time.Second, cons.CooloffDuration)
			require.Equal(t, 30*time.Second, cons.ConnectionTimeout)
		})
	}
}

func TestP2PConfigKeepsExplicitFalse(t *testing.T) {
	t.Run("static", func(t *testing.T) {
		conf := &static.Config{}

		require.NoError(t, configdecode.Decode(p2pConfig(t, TypeStatic, "  discV4: false\n").Config, conf))

		assert.False(t, conf.DiscV4)
		assert.True(t, conf.DiscV5)
		assert.Equal(t, 2*time.Minute, conf.Restart)
	})

	t.Run("xatu", func(t *testing.T) {
		conf := &xatu.Config{}

		require.NoError(t, configdecode.Decode(p2pConfig(t, TypeXatu, "  discV5: false\n").Config, conf))

		assert.True(t, conf.DiscV4)
		assert.False(t, conf.DiscV5)
		assert.Equal(t, "localhost:8080", conf.Address)
	})
}

func TestP2PConfigRejectsZeroDialSettings(t *testing.T) {
	tests := map[string]string{
		"restart":                     "  restart: 0s\n",
		"execution retryAttempts":     "  execution:\n    retryAttempts: 0\n",
		"execution dialTimeout":       "  execution:\n    dialTimeout: 0s\n",
		"consensus dialTimeout":       "  consensus:\n    dialTimeout: 0s\n",
		"consensus dialConcurrency":   "  consensus:\n    dialConcurrency: 0\n",
		"consensus connectionTimeout": "  consensus:\n    connectionTimeout: 0s\n",
	}

	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			staticConf := &static.Config{}
			require.NoError(t, configdecode.Decode(p2pConfig(t, TypeStatic, in).Config, staticConf))
			require.Error(t, staticConf.Validate())

			xatuConf := &xatu.Config{}
			require.NoError(t, configdecode.Decode(p2pConfig(t, TypeXatu, in).Config, xatuConf))
			require.Error(t, xatuConf.Validate())
		})
	}
}

func TestP2PConfigDefaultsAreValid(t *testing.T) {
	staticConf := &static.Config{}
	require.NoError(t, configdecode.Decode(p2pConfig(t, TypeStatic, "").Config, staticConf))
	require.NoError(t, staticConf.Validate())

	xatuConf := &xatu.Config{}
	require.NoError(t, configdecode.Decode(p2pConfig(t, TypeXatu, "").Config, xatuConf))
	require.NoError(t, xatuConf.Validate())
}

// p2pConfig builds a Config of p2pType whose block holds the fields a valid config of that type needs, plus extra.
func p2pConfig(t *testing.T, p2pType Type, extra string) *Config {
	t.Helper()

	required := "  bootNodes: [enr:-x]\n"
	if p2pType == TypeXatu {
		required = "  address: localhost:8080\n"
	}

	var out Config

	require.NoError(t, yaml.Unmarshal([]byte("type: "+string(p2pType)+"\nconfig:\n"+required+extra), &out))

	return &out
}
