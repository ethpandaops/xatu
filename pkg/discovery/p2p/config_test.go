package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
)

func rawP2PConfig(t *testing.T, in string) *RawMessage {
	t.Helper()

	var out Config

	require.NoError(t, yaml.Unmarshal([]byte("config:\n"+in), &out))

	return out.Config
}

func TestDecodeP2PConfigKeepsExplicitFalse(t *testing.T) {
	t.Run("static", func(t *testing.T) {
		conf := &static.Config{}

		require.NoError(t, decodeP2PConfig(rawP2PConfig(t, "  bootNodes: [enr:-x]\n  discV4: false\n"), conf))

		assert.False(t, conf.DiscV4)
		assert.True(t, conf.DiscV5)
		assert.Equal(t, 2*time.Minute, conf.Restart)
	})

	t.Run("xatu", func(t *testing.T) {
		conf := &xatu.Config{}

		require.NoError(t, decodeP2PConfig(rawP2PConfig(t, "  address: localhost:8080\n  discV5: false\n"), conf))

		assert.True(t, conf.DiscV4)
		assert.False(t, conf.DiscV5)
		assert.Equal(t, "localhost:8080", conf.Address)
	})
}

func TestDecodeP2PConfigAppliesDefaults(t *testing.T) {
	conf := &static.Config{}

	require.NoError(t, decodeP2PConfig(rawP2PConfig(t, "  bootNodes: [enr:-x]\n"), conf))

	assert.True(t, conf.DiscV4)
	assert.True(t, conf.DiscV5)
	assert.Equal(t, []string{"enr:-x"}, conf.BootNodes)
}
