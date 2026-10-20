package v2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDepositDeriver_HezeBlockHasNoDeposits(t *testing.T) {
	deposits, err := (&DepositDeriver{}).getDeposits(context.Background(), hezeBlock(1))
	require.NoError(t, err)
	require.Empty(t, deposits)
}
