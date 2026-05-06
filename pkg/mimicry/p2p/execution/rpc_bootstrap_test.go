package execution

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	eth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/stretchr/testify/require"
)

func TestValidatePeerStatus(t *testing.T) {
	genesis := common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	expectedForkID := forkid.ID{Hash: [4]byte{0x07, 0xc9, 0x46, 0x2e}, Next: 0}

	require.NoError(t, validatePeerStatus(nil, 1, genesis, expectedForkID))

	require.NoError(t, validatePeerStatus(
		statusForTest(1, genesis, expectedForkID),
		1,
		genesis,
		expectedForkID,
	))

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(56, genesis, expectedForkID),
		1,
		genesis,
		expectedForkID,
	), "network id")

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, common.HexToHash("0x01"), expectedForkID),
		1,
		genesis,
		expectedForkID,
	), "genesis")

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, genesis, forkid.ID{Hash: [4]byte{0xc3, 0x76, 0xcf, 0x8b}, Next: 0}),
		1,
		genesis,
		expectedForkID,
	), "fork id")

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, genesis, forkid.ID{Hash: expectedForkID.Hash, Next: 1}),
		1,
		genesis,
		expectedForkID,
	), "fork id")
}

func statusForTest(networkID uint64, genesis common.Hash, forkID forkid.ID) mimicry.Status {
	return &mimicry.Status69{
		StatusPacket: eth.StatusPacket{
			NetworkID: networkID,
			Genesis:   genesis,
			ForkID:    forkID,
		},
	}
}
