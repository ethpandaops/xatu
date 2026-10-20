package v1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Lighthouse rewards for mainnet epoch 200759 (May 2023 inactivity leak), trimmed to the first 64 validators.
func TestAttestationRewardDeriver_InactivityLeakEpoch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "mainnet-attestation-rewards-200759.json"))
	require.NoError(t, err)

	var rsp struct {
		Data json.RawMessage `json:"data"`
	}

	require.NoError(t, json.Unmarshal(raw, &rsp))

	var rewards apiv1.AttestationRewards

	require.NoError(t, json.Unmarshal(rsp.Data, &rewards))

	var fields map[string]json.RawMessage

	require.NoError(t, json.Unmarshal(rsp.Data, &fields))

	var wantTotal []map[string]string

	require.NoError(t, json.Unmarshal(fields["total_rewards"], &wantTotal))
	require.Len(t, rewards.TotalRewards, 64)
	require.Len(t, wantTotal, 64)

	deriver := &AttestationRewardDeriver{clientMeta: &xatu.ClientMeta{Name: "cannon-test"}}
	additional := &xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData{}

	penalised := 0

	for i, reward := range rewards.TotalRewards {
		event, err := deriver.createEventFromAttestationReward(reward, additional)
		require.NoError(t, err)

		data := event.GetEthV1BeaconAttestationReward()
		require.NotNil(t, data)

		expected := wantTotal[i]
		assert.Equal(t, expected["validator_index"], strconv.FormatUint(data.GetValidatorIndex().GetValue(), 10))
		assert.Equal(t, expected["head"], strconv.FormatInt(data.GetHead().GetValue(), 10))
		assert.Equal(t, expected["target"], strconv.FormatInt(data.GetTarget().GetValue(), 10))
		assert.Equal(t, expected["source"], strconv.FormatInt(data.GetSource().GetValue(), 10))
		assert.Equal(t, expected["inactivity"], strconv.FormatInt(data.GetInactivity().GetValue(), 10))

		if data.GetInactivity().GetValue() < 0 {
			penalised++
		}
	}

	assert.Equal(t, 2, penalised)

	validator9 := rewards.TotalRewards[9]
	assert.Equal(t, int64(-5722), validator9.Inactivity)
	assert.Equal(t, int64(-6162), validator9.Target)
	assert.Equal(t, int64(-3318), validator9.Source)
}
