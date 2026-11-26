package v1

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/go-eth2-client/spec/pq"
	"github.com/stretchr/testify/require"
)

func TestValidatorPubkey(t *testing.T) {
	var xmss pq.XMSSPubKey

	copy(xmss[:], []byte{0x6f, 0x5e, 0x0b, 0x23})
	xmss[31] = 0x03

	var bls phase0.BLSPubKey

	bls[0] = 0xa9
	bls[47] = 0x01

	require.Equal(t, "0x6f5e0b2300000000000000000000000000000000000000000000000000000003", validatorPubkey(xmss.DepositPubKey()))
	require.Equal(t, bls.String(), validatorPubkey(bls))
}
