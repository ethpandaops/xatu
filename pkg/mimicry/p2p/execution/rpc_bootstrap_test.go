package execution

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	eth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/stretchr/testify/require"
)

func TestValidatePeerStatus(t *testing.T) {
	snapshot := snapshotForTest(t, 1, 23_000_000, 1_750_000_000)

	require.NoError(t, validatePeerStatus(nil, snapshot))

	require.NoError(t, validatePeerStatus(
		statusForTest(1, snapshot.genesis, snapshot.forkID),
		snapshot,
	))

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(56, snapshot.genesis, snapshot.forkID),
		snapshot,
	), "network id")

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, common.HexToHash("0x01"), snapshot.forkID),
		snapshot,
	), "genesis")

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, snapshot.genesis, forkid.ID{Hash: [4]byte{0xff, 0xff, 0xff, 0xff}, Next: 0}),
		snapshot,
	), "fork id")

	invalidNext := snapshot.forkID
	invalidNext.Next = 1

	require.ErrorContains(t, validatePeerStatus(
		statusForTest(1, snapshot.genesis, invalidNext),
		snapshot,
	), "fork id")
}

func TestValidatePeerStatusAcceptsCompatibleForkID(t *testing.T) {
	snapshot := snapshotForTest(t, 1, 23_000_000, 1_750_000_000)
	chainConfig, genesis, _, err := bootstrapNetwork(1)
	require.NoError(t, err)

	syncingForkID := forkid.NewID(chainConfig, genesis, 0, genesis.Time())

	require.NoError(t, validatePeerStatus(
		statusForTest(1, snapshot.genesis, syncingForkID),
		snapshot,
	))
}

func snapshotForTest(t *testing.T, networkID, headNumber, headTime uint64) *rpcBootstrapStatus {
	t.Helper()

	chainConfig, genesis, terminalTotalDifficulty, err := bootstrapNetwork(networkID)
	require.NoError(t, err)

	head := &types.Header{
		Number: new(big.Int).SetUint64(headNumber),
		Time:   headTime,
	}
	forkID := forkid.NewID(chainConfig, genesis, headNumber, headTime)

	return &rpcBootstrapStatus{
		networkID:               networkID,
		genesis:                 genesis.Hash(),
		terminalTotalDifficulty: terminalTotalDifficulty,
		headNumber:              headNumber,
		forkID:                  forkID,
		forkFilter: forkid.NewFilter(&rpcBootstrapChain{
			config:  chainConfig,
			genesis: genesis,
			head:    head,
		}),
	}
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
