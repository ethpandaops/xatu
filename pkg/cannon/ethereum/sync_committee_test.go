package ethereum

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
)

func TestSyncCommitteeStateEpoch(t *testing.T) {
	const epochsPerPeriod = phase0.Epoch(256)

	tests := []struct {
		name   string
		altair phase0.Epoch
		epoch  phase0.Epoch
		want   phase0.Epoch
	}{
		// Sepolia forked to Altair at epoch 50, inside sync committee period 0.
		{name: "sepolia altair fork epoch", altair: 50, epoch: 50, want: 50},
		{name: "sepolia last epoch of period 0", altair: 50, epoch: 255, want: 50},
		{name: "sepolia period 1 start", altair: 50, epoch: 256, want: 256},
		{name: "sepolia inside period 1", altair: 50, epoch: 300, want: 256},
		// Mainnet forked to Altair exactly on the period 290 boundary.
		{name: "mainnet altair fork epoch", altair: 74240, epoch: 74240, want: 74240},
		{name: "mainnet inside period 290", altair: 74240, epoch: 74495, want: 74240},
		{name: "mainnet period 291 start", altair: 74240, epoch: 74496, want: 74496},
		// Hoodi starts at Altair or later from genesis.
		{name: "hoodi genesis", altair: 0, epoch: 0, want: 0},
		{name: "hoodi inside period 0", altair: 0, epoch: 255, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, syncCommitteeStateEpoch(tt.epoch, epochsPerPeriod, tt.altair))
		})
	}
}
