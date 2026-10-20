package v2

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttestationPayloadPresent(t *testing.T) {
	tests := []struct {
		name    string
		version spec.DataVersion
		index   phase0.CommitteeIndex
		wantSet bool
		want    bool
	}{
		{name: "electra has no payload vote", version: spec.DataVersionElectra, index: 0},
		{name: "fulu has no payload vote", version: spec.DataVersionFulu, index: 0},
		{name: "gloas index 0 votes payload absent", version: spec.DataVersionGloas, index: 0, wantSet: true},
		{name: "gloas index 1 votes payload present", version: spec.DataVersionGloas, index: 1, wantSet: true, want: true},
		{name: "heze index 1 votes payload present", version: spec.DataVersionHeze, index: 1, wantSet: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attestationPayloadPresent(tt.version, &phase0.AttestationData{Index: tt.index})
			if !tt.wantSet {
				assert.Nil(t, got)

				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tt.want, got.GetValue())
		})
	}
}

func TestNewSetCodeAuthorizations(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	delegate := common.HexToAddress("0x63c0c19a282a1b52b07dd5a65b58948a07dae32b")

	signed, err := types.SignSetCode(key, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(1),
		Address: delegate,
		Nonce:   7,
	})
	require.NoError(t, err)

	unsigned := types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(0),
		Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Nonce:   0,
	}

	got := newSetCodeAuthorizations([]types.SetCodeAuthorization{signed, unsigned})
	require.Len(t, got, 2)

	assert.Equal(t, "1", got[0].GetChainId())
	assert.Equal(t, delegate.Hex(), got[0].GetAddress())
	assert.Equal(t, uint64(7), got[0].GetNonce().GetValue())
	assert.Equal(t, crypto.PubkeyToAddress(key.PublicKey).Hex(), got[0].GetAuthority())

	assert.Equal(t, "0", got[1].GetChainId())
	assert.Equal(t, uint64(0), got[1].GetNonce().GetValue())
	assert.Empty(t, got[1].GetAuthority(), "an unrecoverable signature has no authority")
}
