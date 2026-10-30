package iterator

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillTargetEpoch(t *testing.T) {
	tests := []struct {
		name            string
		activationEpoch phase0.Epoch
		toEpoch         phase0.Epoch
		want            phase0.Epoch
	}{
		{name: "phase0 type without toEpoch backfills to genesis", activationEpoch: 0, toEpoch: 0, want: 0},
		{name: "phase0 type honours toEpoch", activationEpoch: 0, toEpoch: 437872, want: 437872},
		{name: "forked type without toEpoch stops at its fork", activationEpoch: 269568, toEpoch: 0, want: 269568},
		{name: "forked type honours a later toEpoch", activationEpoch: 269568, toEpoch: 437872, want: 437872},
		{name: "toEpoch before the fork is clamped to the fork", activationEpoch: 269568, toEpoch: 100, want: 269568},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, backfillTargetEpoch(tt.activationEpoch, tt.toEpoch))
		})
	}
}

func TestGetEarliestPossibleBackfillEpoch_EpochsBeforeFork(t *testing.T) {
	tests := []struct {
		name      string
		forkEpoch phase0.Epoch
		toEpoch   phase0.Epoch
		want      phase0.Epoch
	}{
		{name: "reaches the epoch before the fork", forkEpoch: 74240, want: 74239},
		{name: "toEpoch at the epoch before the fork", forkEpoch: 74240, toEpoch: 74239, want: 74239},
		{name: "later toEpoch wins", forkEpoch: 74240, toEpoch: 80000, want: 80000},
		{name: "fork at genesis stays at genesis", forkEpoch: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &BackfillingCheckpointConfig{}
			config.Backfill.Enabled = true
			config.Backfill.ToEpoch = tt.toEpoch

			c := &BackfillingCheckpoint{
				config:           config,
				activationFork:   spec.DataVersionAltair,
				epochsBeforeFork: 1,
				chain:            &staticChain{forkEpochs: map[spec.DataVersion]phase0.Epoch{spec.DataVersionAltair: tt.forkEpoch}},
			}

			got, err := c.getEarliestPossibleBackfillEpoch()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEarliestPossibleBackfillEpoch_Phase0HonoursToEpoch(t *testing.T) {
	config := &BackfillingCheckpointConfig{}
	config.Backfill.Enabled = true
	config.Backfill.ToEpoch = 437872

	c := &BackfillingCheckpoint{config: config, activationFork: spec.DataVersionPhase0}

	got, err := c.getEarliestPossibleBackfillEpoch()
	require.NoError(t, err)
	assert.Equal(t, phase0.Epoch(437872), got)
}
