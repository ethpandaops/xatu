package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestLocation_BeaconPhase0Reward_RoundTrip(t *testing.T) {
	msg := &xatu.CannonLocation{
		NetworkId: "1",
		Type:      xatu.CannonType_BEACON_API_ETH_V1_BEACON_PHASE0_REWARD,
		Data: &xatu.CannonLocation_EthV1BeaconPhase0Reward{
			EthV1BeaconPhase0Reward: &xatu.CannonLocationEthV1BeaconPhase0Reward{
				BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{FinalizedEpoch: 74239, BackfillEpoch: 70000},
			},
		},
	}

	l := &Location{}
	require.NoError(t, l.Marshal(msg))

	assert.Equal(t, "BEACON_API_ETH_V1_BEACON_PHASE0_REWARD", l.Type)
	assert.Equal(t, "1", l.NetworkID)

	out, err := l.Unmarshal()
	require.NoError(t, err)

	marker := out.GetEthV1BeaconPhase0Reward().GetBackfillingCheckpointMarker()
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V1_BEACON_PHASE0_REWARD, out.GetType())
	assert.Equal(t, uint64(74239), marker.GetFinalizedEpoch())
	assert.Equal(t, int64(70000), marker.GetBackfillEpoch())
}
