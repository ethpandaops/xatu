package v1

import (
	"context"
	"encoding/json"
	"testing"

	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSyncCommitteeRewardDeriver_CreateEvents(t *testing.T) {
	tests := []struct {
		name string
		// rsp is the data of a POST /eth/v1/beacon/rewards/sync_committee/{block_id} response.
		rsp        string
		wantReward map[uint64]int64
		wantTotal  int64
	}{
		{
			// Nimbus returns one entry per sync committee seat.
			name: "per seat",
			rsp: `[
				{"validator_index":"103","reward":"951"},
				{"validator_index":"7","reward":"-951"},
				{"validator_index":"103","reward":"900"},
				{"validator_index":"42","reward":"951"},
				{"validator_index":"103","reward":"10"},
				{"validator_index":"7","reward":"-100"}
			]`,
			wantReward: map[uint64]int64{103: 1861, 7: -1051, 42: 951},
			wantTotal:  1761,
		},
		{
			// Grandine, Lighthouse and Prysm return one entry per validator.
			name: "per validator",
			rsp: `[
				{"validator_index":"103","reward":"1861"},
				{"validator_index":"7","reward":"-1051"},
				{"validator_index":"42","reward":"951"}
			]`,
			wantReward: map[uint64]int64{103: 1861, 7: -1051, 42: 951},
			wantTotal:  1761,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rewards []*apiv1.SyncCommitteeReward

			require.NoError(t, json.Unmarshal([]byte(tt.rsp), &rewards))

			deriver := &SyncCommitteeRewardDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
			block := &xatu.BlockIdentifier{Root: "0xabc"}

			events, err := deriver.createEvents(context.Background(), rewards, block)
			require.NoError(t, err)
			require.Len(t, events, len(tt.wantReward))

			gotReward := make(map[uint64]int64, len(events))

			var total int64

			for _, event := range events {
				data := event.GetEthV1BeaconSyncCommitteeReward()
				require.NotNil(t, data)

				index := data.GetValidatorIndex().GetValue()
				require.NotContains(t, gotReward, index, "validator %d has more than one event", index)

				gotReward[index] = data.GetReward().GetValue()
				total += data.GetReward().GetValue()

				assert.Equal(t, "0xabc", event.GetMeta().GetClient().GetEthV1BeaconSyncCommitteeReward().GetBlock().GetRoot())
			}

			assert.Equal(t, tt.wantReward, gotReward)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}
