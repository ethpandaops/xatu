package rpcbootstrap

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	eth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/params"
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

func TestBootstrapNetworkForkIDs(t *testing.T) {
	tests := []struct {
		name      string
		networkID uint64
		headTime  uint64
		want      forkid.ID
	}{
		{
			name:      "mainnet bpo2",
			networkID: 1,
			headTime:  2_000_000_000,
			want:      forkid.ID{Hash: [4]byte{0x07, 0xc9, 0x46, 0x2e}, Next: 0},
		},
		{
			name:      "sepolia before amsterdam",
			networkID: 11155111,
			headTime:  1_791_294_815,
			want:      forkid.ID{Hash: [4]byte{0x26, 0x89, 0x56, 0xb6}, Next: 1_791_294_816},
		},
		{
			name:      "sepolia amsterdam",
			networkID: 11155111,
			headTime:  1_791_294_816,
			want:      forkid.ID{Hash: [4]byte{0x6c, 0x1d, 0x94, 0x23}, Next: 0},
		},
		{
			name:      "hoodi before amsterdam",
			networkID: 560048,
			headTime:  1_793_036_567,
			want:      forkid.ID{Hash: [4]byte{0x23, 0xaa, 0x13, 0x51}, Next: 1_793_036_568},
		},
		{
			name:      "hoodi amsterdam",
			networkID: 560048,
			headTime:  1_793_036_568,
			want:      forkid.ID{Hash: [4]byte{0x3d, 0x06, 0x8b, 0x59}, Next: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := snapshotForTest(t, tt.networkID, 30_000_000, tt.headTime)

			require.Equal(t, tt.want, snapshot.forkID)
			require.NoError(t, validatePeerStatus(
				statusForTest(tt.networkID, snapshot.genesis, tt.want),
				snapshot,
			))
		})
	}
}

func TestHoodiChainConfigLeavesSharedConfigUntouched(t *testing.T) {
	before := params.HoodiChainConfig.AmsterdamTime

	chainConfig := hoodiChainConfig()

	require.True(t, before == params.HoodiChainConfig.AmsterdamTime)
	require.NotNil(t, chainConfig.AmsterdamTime)
	require.Equal(t, hoodiAmsterdamTime, *chainConfig.AmsterdamTime)
}

func snapshotForTest(t *testing.T, networkID, headNumber, headTime uint64) *statusSnapshot {
	t.Helper()

	chainConfig, genesis, terminalTotalDifficulty, err := bootstrapNetwork(networkID)
	require.NoError(t, err)

	head := &types.Header{
		Number: new(big.Int).SetUint64(headNumber),
		Time:   headTime,
	}
	forkID := forkid.NewID(chainConfig, genesis, headNumber, headTime)

	return &statusSnapshot{
		networkID:               networkID,
		genesis:                 genesis.Hash(),
		terminalTotalDifficulty: terminalTotalDifficulty,
		headNumber:              headNumber,
		forkID:                  forkID,
		forkFilter: forkid.NewFilter(&bootstrapChain{
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
