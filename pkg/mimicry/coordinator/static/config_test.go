package static

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRejectsNegativeMaxConcurrentPeers(t *testing.T) {
	cfg := &Config{
		NodeRecords:        []string{"enr:test"},
		MaxConcurrentPeers: -1,
	}

	require.ErrorContains(t, cfg.Validate(), "maxConcurrentPeers")
}
