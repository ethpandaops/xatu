package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
)

func TestGetExecutionConfigDefaults(t *testing.T) {
	cfg := &Config{}

	exec := cfg.GetExecutionConfig()

	require.Equal(t, uint(1), exec.RetryAttempts)
	require.Equal(t, 60*time.Second, exec.RetryDelay)
	require.Equal(t, 15*time.Second, exec.DialTimeout)
}

func TestApplyExecutionDefaultsPreservesExplicitValues(t *testing.T) {
	cfg := &Config{}

	exec := cfg.applyExecutionDefaults(&static.ExecutionConfig{
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
		DialTimeout:   10 * time.Second,
	})

	require.Equal(t, uint(3), exec.RetryAttempts)
	require.Equal(t, 5*time.Second, exec.RetryDelay)
	require.Equal(t, 10*time.Second, exec.DialTimeout)
}
