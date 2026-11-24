package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestLocation_BeaconBlockGraffiti_RoundTrip(t *testing.T) {
	msg := &xatu.CannonLocation{
		NetworkId: "1",
		Type:      xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI,
		Data: &xatu.CannonLocation_EthV2BeaconBlockGraffiti{
			EthV2BeaconBlockGraffiti: &xatu.CannonLocationEthV2BeaconBlockGraffiti{
				BackfillingCheckpointMarker: &xatu.BackfillingCheckpointMarker{FinalizedEpoch: 300, BackfillEpoch: 200},
			},
		},
	}

	l := &Location{}
	require.NoError(t, l.Marshal(msg))

	assert.Equal(t, "BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI", l.Type)
	assert.Equal(t, "1", l.NetworkID)

	out, err := l.Unmarshal()
	require.NoError(t, err)

	marker := out.GetEthV2BeaconBlockGraffiti().GetBackfillingCheckpointMarker()
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_GRAFFITI, out.GetType())
	assert.Equal(t, uint64(300), marker.GetFinalizedEpoch())
	assert.Equal(t, int64(200), marker.GetBackfillEpoch())
}
