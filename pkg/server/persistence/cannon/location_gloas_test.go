package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestLocation_GloasCheckpointTypes_RoundTrip(t *testing.T) {
	marker := &xatu.BackfillingCheckpointMarker{FinalizedEpoch: 300, BackfillEpoch: 200}

	tests := []struct {
		name     string
		wantType string
		msg      *xatu.CannonLocation
		marker   func(*xatu.CannonLocation) *xatu.BackfillingCheckpointMarker
	}{
		{
			name:     "builder deposit",
			wantType: "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_DEPOSIT,
				Data: &xatu.CannonLocation_EthV2BeaconBlockExecutionRequestBuilderDeposit{
					EthV2BeaconBlockExecutionRequestBuilderDeposit: &xatu.CannonLocationEthV2BeaconBlockExecutionRequestBuilderDeposit{
						BackfillingCheckpointMarker: marker,
					},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV2BeaconBlockExecutionRequestBuilderDeposit().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "builder exit",
			wantType: "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_BUILDER_EXIT,
				Data: &xatu.CannonLocation_EthV2BeaconBlockExecutionRequestBuilderExit{
					EthV2BeaconBlockExecutionRequestBuilderExit: &xatu.CannonLocationEthV2BeaconBlockExecutionRequestBuilderExit{
						BackfillingCheckpointMarker: marker,
					},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV2BeaconBlockExecutionRequestBuilderExit().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "block access list summary",
			wantType: "BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST_SUMMARY,
				Data: &xatu.CannonLocation_EthV2BeaconBlockAccessListSummary{
					EthV2BeaconBlockAccessListSummary: &xatu.CannonLocationEthV2BeaconBlockAccessListSummary{
						BackfillingCheckpointMarker: marker,
					},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV2BeaconBlockAccessListSummary().GetBackfillingCheckpointMarker()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.msg.NetworkId = "11155111"

			l := &Location{}
			require.NoError(t, l.Marshal(tt.msg))

			assert.Equal(t, tt.wantType, l.Type)
			assert.Equal(t, "11155111", l.NetworkID)

			out, err := l.Unmarshal()
			require.NoError(t, err)

			assert.Equal(t, tt.msg.GetType(), out.GetType())
			assert.Equal(t, uint64(300), tt.marker(out).GetFinalizedEpoch())
			assert.Equal(t, int64(200), tt.marker(out).GetBackfillEpoch())
		})
	}
}
