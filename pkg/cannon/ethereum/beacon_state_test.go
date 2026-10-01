package ethereum

import (
	"bytes"
	"math"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	dynssz "github.com/pk910/dynamic-ssz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSlotsPerEpoch = 32

// newTestBeaconState returns a structurally complete Gloas state at slot whose
// ptc_window cell (window index, position) holds index*1000+position.
func newTestBeaconState(slot phase0.Slot) *gloas.BeaconState {
	window := make([][]phase0.ValidatorIndex, 96)
	for i := range window {
		window[i] = make([]phase0.ValidatorIndex, 512)

		for j := range window[i] {
			window[i][j] = phase0.ValidatorIndex(i*1000 + j)
		}
	}

	payments := make([]*gloas.BuilderPendingPayment, 64)
	for i := range payments {
		payments[i] = &gloas.BuilderPendingPayment{Withdrawal: &gloas.BuilderPendingWithdrawal{}}
	}

	return &gloas.BeaconState{
		Slot:                         slot,
		Fork:                         &phase0.Fork{},
		LatestBlockHeader:            &phase0.BeaconBlockHeader{Slot: slot},
		BlockRoots:                   make([]phase0.Root, 8192),
		StateRoots:                   make([]phase0.Root, 8192),
		ETH1Data:                     &phase0.ETH1Data{},
		RANDAOMixes:                  make([]phase0.Root, 65536),
		Slashings:                    make([]phase0.Gwei, 8192),
		PreviousJustifiedCheckpoint:  &phase0.Checkpoint{},
		CurrentJustifiedCheckpoint:   &phase0.Checkpoint{},
		FinalizedCheckpoint:          &phase0.Checkpoint{Epoch: 10},
		CurrentSyncCommittee:         &altair.SyncCommittee{Pubkeys: make([]phase0.BLSPubKey, 512)},
		NextSyncCommittee:            &altair.SyncCommittee{Pubkeys: make([]phase0.BLSPubKey, 512)},
		ProposerLookahead:            make([]phase0.ValidatorIndex, 64),
		ExecutionPayloadAvailability: make([]byte, 1024),
		BuilderPendingPayments:       payments,
		LatestExecutionPayloadBid:    &gloas.ExecutionPayloadBid{},
		PTCWindow:                    window,
		Validators:                   []*phase0.Validator{{}},
		Balances:                     []phase0.Gwei{32_000_000_000},
		Builders: []*gloas.Builder{
			{Balance: 5, DepositEpoch: 7, WithdrawableEpoch: phase0.Epoch(math.MaxUint64)},
		},
		BuilderPendingWithdrawals: []*gloas.BuilderPendingWithdrawal{{Amount: 9, BuilderIndex: 3}},
	}
}

func TestDecodeGloasEpochState(t *testing.T) {
	state := newTestBeaconState(3200)
	state.FinalizedCheckpoint.Epoch = 98
	state.LatestBlockHeader.Slot = 3199
	state.ExecutionPayloadAvailability[3199%8192/8] |= 1 << (3199 % 8)
	state.BuilderPendingPayments[35] = &gloas.BuilderPendingPayment{
		Weight:        11,
		Withdrawal:    &gloas.BuilderPendingWithdrawal{Amount: 22, BuilderIndex: 4},
		ProposerIndex: 5,
	}

	ds := dynssz.GetGlobalDynSsz()

	raw, err := ds.MarshalSSZ(state)
	require.NoError(t, err)

	for name, size := range map[string]int{"known length": len(raw), "unknown length": -1} {
		t.Run(name, func(t *testing.T) {
			got, err := decodeGloasEpochState(ds, bytes.NewReader(raw), size)
			require.NoError(t, err)

			assert.Equal(t, phase0.Slot(3200), got.Slot)
			assert.Equal(t, phase0.Slot(3199), got.LatestBlockHeaderSlot)
			assert.Equal(t, phase0.Epoch(98), got.FinalizedEpoch)
			require.Len(t, got.Builders, 1)
			assert.Equal(t, phase0.Gwei(5), got.Builders[0].Balance)
			require.Len(t, got.BuilderPendingWithdrawals, 1)
			assert.Equal(t, phase0.Gwei(9), got.BuilderPendingWithdrawals[0].Amount)
			require.Len(t, got.BuilderPendingPayments, 64)
			assert.Equal(t, phase0.Gwei(22), got.BuilderPendingPayments[35].Withdrawal.Amount)
			assert.Equal(t, phase0.ValidatorIndex(5), got.BuilderPendingPayments[35].ProposerIndex)
			require.Len(t, got.PTCWindow, 96)
			assert.Equal(t, phase0.ValidatorIndex(40_007), got.PTCWindow[40][7])

			available, err := got.PayloadAvailable(3199)
			require.NoError(t, err)
			assert.True(t, available)
		})
	}
}

func TestDecodeGloasEpochStateRejectsTruncatedInput(t *testing.T) {
	ds := dynssz.GetGlobalDynSsz()

	raw, err := ds.MarshalSSZ(newTestBeaconState(3200))
	require.NoError(t, err)

	_, err = decodeGloasEpochState(ds, bytes.NewReader(raw[:len(raw)/2]), len(raw))
	require.Error(t, err)
}

func TestGloasEpochStatePTC(t *testing.T) {
	// The state is at the first slot of epoch 100.
	state, err := newGloasEpochState(newTestBeaconState(100 * testSlotsPerEpoch))
	require.NoError(t, err)

	tests := []struct {
		name       string
		slot       phase0.Slot
		wantWindow int
		wantErr    bool
	}{
		{name: "previous epoch first slot", slot: 99 * testSlotsPerEpoch, wantWindow: 0},
		{name: "previous epoch last slot", slot: 100*testSlotsPerEpoch - 1, wantWindow: 31},
		{name: "current epoch first slot", slot: 100 * testSlotsPerEpoch, wantWindow: 32},
		{name: "current epoch last slot", slot: 101*testSlotsPerEpoch - 1, wantWindow: 63},
		{name: "next epoch first slot", slot: 101 * testSlotsPerEpoch, wantWindow: 64},
		{name: "next epoch last slot", slot: 102*testSlotsPerEpoch - 1, wantWindow: 95},
		{name: "two epochs behind", slot: 98 * testSlotsPerEpoch, wantErr: true},
		{name: "two epochs ahead", slot: 102 * testSlotsPerEpoch, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptc, err := state.PTC(tt.slot, testSlotsPerEpoch)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Len(t, ptc, 512)
			// Cells are index*1000+position, so the first and last members
			// identify the window entry and prove the order is preserved.
			assert.Equal(t, phase0.ValidatorIndex(tt.wantWindow*1000), ptc[0])
			assert.Equal(t, phase0.ValidatorIndex(tt.wantWindow*1000+511), ptc[511])
		})
	}
}

func TestGloasEpochStatePTCMidEpochState(t *testing.T) {
	// A state a few slots into epoch 100 still reads the same window.
	state, err := newGloasEpochState(newTestBeaconState(100*testSlotsPerEpoch + 9))
	require.NoError(t, err)

	ptc, err := state.PTC(100*testSlotsPerEpoch+9, testSlotsPerEpoch)
	require.NoError(t, err)
	assert.Equal(t, phase0.ValidatorIndex(41_000), ptc[0])
}

func TestGloasEpochStatePayloadAvailable(t *testing.T) {
	base := newTestBeaconState(8200)
	base.LatestBlockHeader.Slot = 8199
	// Bit for slot 8193 wraps to index 1.
	base.ExecutionPayloadAvailability[0] = 0b0000_0010

	state, err := newGloasEpochState(base)
	require.NoError(t, err)

	tests := []struct {
		name    string
		slot    phase0.Slot
		want    bool
		wantErr bool
	}{
		{name: "set bit wrapped around the vector", slot: 8193, want: true},
		{name: "unset bit", slot: 8194, want: false},
		{name: "slot ahead of the state", slot: 8201, wantErr: true},
		{name: "slot a full vector behind the state", slot: 8, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := state.PayloadAvailable(tt.slot)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	assert.True(t, state.PayloadAvailabilitySettled(8198), "a later block exists")
	assert.False(t, state.PayloadAvailabilitySettled(8199), "the latest block's own slot is still open")
	assert.False(t, state.PayloadAvailabilitySettled(8200), "no block yet at the state slot")
}

func TestNewGloasEpochStateDetachesPTCWindow(t *testing.T) {
	src := newTestBeaconState(3200)

	state, err := newGloasEpochState(src)
	require.NoError(t, err)

	src.PTCWindow[0][0] = 999_999
	src.ExecutionPayloadAvailability[0] = 0xff

	assert.Equal(t, phase0.ValidatorIndex(0), state.PTCWindow[0][0])
	assert.Equal(t, byte(0), state.ExecutionPayloadAvailability[0])
}
