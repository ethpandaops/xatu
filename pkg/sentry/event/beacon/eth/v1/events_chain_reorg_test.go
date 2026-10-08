package event

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethpandaops/ethwallclock"
	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReorgEventV2Epoch(t *testing.T) {
	wallclock := ethwallclock.NewEthereumBeaconChain(time.Unix(1_700_000_000, 0), 12*time.Second, 32)

	tests := []struct {
		name  string
		input string
		epoch uint64
	}{
		{
			// Nimbus payload: no epoch.
			name:  "NimbusNoEpoch",
			input: `{"slot":"343871","depth":"2","old_head_block":"0x9af8248c506f8635c5eb734d45508854ef8a1668fcece9813c0a8e2a1ace4b73","new_head_block":"0x0fec5a8fb2e9a13b5393db10bd9df1caaef2774e0b3083a19e5784402c8baa25","old_head_state":"0x405653c3ebba8072ef70a6ab7588f64f1b6a32495810139928c0ac1ecb6ecbc3","new_head_state":"0xc7b8f3b19ba14bdbae92a322c695583ee19dba5cb3ff466114c9c5ca8011ab56","execution_optimistic":false}`,
			epoch: 343871 / 32,
		},
		{
			// Every other CL sends epoch; it is kept as reported.
			name:  "EpochPresent",
			input: `{"slot":"343871","depth":"2","old_head_block":"0x9af8248c506f8635c5eb734d45508854ef8a1668fcece9813c0a8e2a1ace4b73","new_head_block":"0x0fec5a8fb2e9a13b5393db10bd9df1caaef2774e0b3083a19e5784402c8baa25","old_head_state":"0x405653c3ebba8072ef70a6ab7588f64f1b6a32495810139928c0ac1ecb6ecbc3","new_head_state":"0xc7b8f3b19ba14bdbae92a322c695583ee19dba5cb3ff466114c9c5ca8011ab56","epoch":"10745","execution_optimistic":false}`,
			epoch: 10745,
		},
		{
			name:  "GenesisEpoch",
			input: `{"slot":"5","depth":"1","old_head_block":"0x9af8248c506f8635c5eb734d45508854ef8a1668fcece9813c0a8e2a1ace4b73","new_head_block":"0x0fec5a8fb2e9a13b5393db10bd9df1caaef2774e0b3083a19e5784402c8baa25","old_head_state":"0x405653c3ebba8072ef70a6ab7588f64f1b6a32495810139928c0ac1ecb6ecbc3","new_head_state":"0xc7b8f3b19ba14bdbae92a322c695583ee19dba5cb3ff466114c9c5ca8011ab56"}`,
			epoch: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var event eth2v1.ChainReorgEvent
			require.NoError(t, json.Unmarshal([]byte(test.input), &event))

			reorg := newReorgEventV2(&event, wallclock)
			require.NotNil(t, reorg.GetEpoch())
			assert.Equal(t, test.epoch, reorg.GetEpoch().GetValue())
			assert.Equal(t, uint64(event.Slot), reorg.GetSlot().GetValue())
		})
	}
}
