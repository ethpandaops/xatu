package ethereum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRejectsInvalidPrivateKey(t *testing.T) {
	cfg := &Config{PrivateKey: "not-a-private-key"}

	require.ErrorContains(t, cfg.Validate(), "invalid privateKey")
}
