package v2

import (
	"context"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mainnet slot 6110523 is a Bellatrix block carrying one attester slashing.
func TestAttesterSlashingDeriver_BellatrixBlock(t *testing.T) {
	block := loadEraBlock(t, "mainnet-6110523", spec.DataVersionBellatrix)

	slashings, err := (&AttesterSlashingDeriver{}).getAttesterSlashings(context.Background(), block)
	require.NoError(t, err)
	require.Len(t, slashings, 1)

	att1 := slashings[0].GetAttestation_1()
	require.Len(t, att1.GetAttestingIndices(), 1)
	assert.Equal(t, uint64(464190), att1.GetAttestingIndices()[0].GetValue())
	assert.Equal(t, uint64(6110522), att1.GetData().GetSlot().GetValue())
	assert.Equal(t, uint64(19), att1.GetData().GetIndex().GetValue())
	assert.Equal(t, "0x823834f093e87f2c379d18f9e599213bf133e564415f5ca0fb217db75f25524a", att1.GetData().GetBeaconBlockRoot())
	assert.Equal(t, uint64(190952), att1.GetData().GetSource().GetEpoch().GetValue())
	assert.Equal(t, uint64(190953), att1.GetData().GetTarget().GetEpoch().GetValue())
	assert.Equal(t, "0x29f401dd1f588476eaae53dfe475ca90e7b00018e3413d95d9b425df1df0f38b", att1.GetData().GetTarget().GetRoot())
	assert.Equal(t, "0xabd25298cbdc028da51769ecb8767f4c155a0bbabc6d623904880cbf7901db397df164e11929cf8a2ead3493729d468f019b072f7cc2c376b3dc4813912223b921847e3afdbc87d0dde71a8606d92d9469ef49bdaad0f1de508f3e8b210c5c5d", att1.GetSignature())

	att2 := slashings[0].GetAttestation_2()
	require.Len(t, att2.GetAttestingIndices(), 270)
	assert.Equal(t, uint64(319), att2.GetAttestingIndices()[0].GetValue())
	assert.Equal(t, uint64(6110522), att2.GetData().GetSlot().GetValue())
	assert.Equal(t, "0x07b2109c8814dc3aa23b3e0ccce82688ec6d69e8679caac654ac31040198b156", att2.GetData().GetBeaconBlockRoot())
}
