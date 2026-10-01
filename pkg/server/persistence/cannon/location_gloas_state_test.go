package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestLocation_GloasBeaconState_RoundTrip(t *testing.T) {
	marker := func() *xatu.BackfillingCheckpointMarker {
		return &xatu.BackfillingCheckpointMarker{FinalizedEpoch: 1700, BackfillEpoch: 1600}
	}

	tests := []struct {
		name     string
		typeName string
		msg      *xatu.CannonLocation
		marker   func(*xatu.CannonLocation) *xatu.BackfillingCheckpointMarker
	}{
		{
			name:     "ptc member",
			typeName: "BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER,
				Data: &xatu.CannonLocation_EthV1BeaconStatePtcMember{
					EthV1BeaconStatePtcMember: &xatu.CannonLocationEthV1BeaconStatePtcMember{BackfillingCheckpointMarker: marker()},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV1BeaconStatePtcMember().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "builder",
			typeName: "BEACON_API_ETH_V1_BEACON_STATE_BUILDER",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER,
				Data: &xatu.CannonLocation_EthV1BeaconStateBuilder{
					EthV1BeaconStateBuilder: &xatu.CannonLocationEthV1BeaconStateBuilder{BackfillingCheckpointMarker: marker()},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV1BeaconStateBuilder().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "builder pending payment",
			typeName: "BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT,
				Data: &xatu.CannonLocation_EthV1BeaconStateBuilderPendingPayment{
					EthV1BeaconStateBuilderPendingPayment: &xatu.CannonLocationEthV1BeaconStateBuilderPendingPayment{BackfillingCheckpointMarker: marker()},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV1BeaconStateBuilderPendingPayment().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "builder pending withdrawal",
			typeName: "BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL,
				Data: &xatu.CannonLocation_EthV1BeaconStateBuilderPendingWithdrawal{
					EthV1BeaconStateBuilderPendingWithdrawal: &xatu.CannonLocationEthV1BeaconStateBuilderPendingWithdrawal{BackfillingCheckpointMarker: marker()},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV1BeaconStateBuilderPendingWithdrawal().GetBackfillingCheckpointMarker()
			},
		},
		{
			name:     "execution payload availability",
			typeName: "BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY",
			msg: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY,
				Data: &xatu.CannonLocation_EthV1BeaconStateExecutionPayloadAvailability{
					EthV1BeaconStateExecutionPayloadAvailability: &xatu.CannonLocationEthV1BeaconStateExecutionPayloadAvailability{BackfillingCheckpointMarker: marker()},
				},
			},
			marker: func(l *xatu.CannonLocation) *xatu.BackfillingCheckpointMarker {
				return l.GetEthV1BeaconStateExecutionPayloadAvailability().GetBackfillingCheckpointMarker()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.msg.NetworkId = "11155111"

			l := &Location{}
			require.NoError(t, l.Marshal(tt.msg))
			assert.Equal(t, tt.typeName, l.Type)
			assert.Equal(t, "11155111", l.NetworkID)

			out, err := l.Unmarshal()
			require.NoError(t, err)
			assert.Equal(t, tt.msg.GetType(), out.GetType())

			got := tt.marker(out)
			assert.Equal(t, uint64(1700), got.GetFinalizedEpoch())
			assert.Equal(t, int64(1600), got.GetBackfillEpoch())
		})
	}
}
