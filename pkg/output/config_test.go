package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/internal/configdecode"
	"github.com/ethpandaops/xatu/pkg/output/http"
	"github.com/ethpandaops/xatu/pkg/output/xatu"
)

func rawSinkConfig(t *testing.T, in string) *RawMessage {
	t.Helper()

	var out Config

	require.NoError(t, yaml.Unmarshal([]byte("name: test\ntype: xatu\nconfig:\n"+in), &out))

	return out.Config
}

func TestSinkConfigKeepsExplicitFalse(t *testing.T) {
	conf := &xatu.Config{}

	require.NoError(t, configdecode.Decode(rawSinkConfig(t, "  address: localhost:8080\n  retry:\n    enabled: false\n"), conf))

	assert.False(t, conf.Retry.Enabled)
	assert.Equal(t, 3, conf.Retry.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, conf.Retry.Scalar)
	assert.Equal(t, "localhost:8080", conf.Address)
}

func TestSinkConfigAppliesDefaults(t *testing.T) {
	conf := &xatu.Config{}

	require.NoError(t, configdecode.Decode(rawSinkConfig(t, "  address: localhost:8080\n  maxQueueSize: 10\n"), conf))

	assert.True(t, conf.Retry.Enabled)
	assert.Equal(t, 10, conf.MaxQueueSize)
	assert.Equal(t, 512, conf.MaxExportBatchSize)
	require.NotNil(t, conf.KeepAlive.Enabled)
	assert.True(t, *conf.KeepAlive.Enabled)
}

func TestSinkConfigWithoutConfigBlock(t *testing.T) {
	conf := &http.Config{}

	require.NoError(t, configdecode.Decode((*RawMessage)(nil), conf))

	require.NotNil(t, conf.KeepAlive)
	assert.True(t, *conf.KeepAlive)
}
