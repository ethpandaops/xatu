package event

import (
	"encoding/json"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/stretchr/testify/require"
)

func TestEventsAttestationGetDataPQ(t *testing.T) {
	input := `{"aggregation_bits":"0x0b","data":{"slot":"4070","index":"1","beacon_block_root":"0x1111111111111111111111111111111111111111111111111111111111111111","source":{"epoch":"125","root":"0x2222222222222222222222222222222222222222222222222222222222222222"},"target":{"epoch":"127","root":"0x3333333333333333333333333333333333333333333333333333333333333333"}},"committee_bits":"0x0200"}`

	var attestation spec.VersionedAttestation
	require.NoError(t, json.Unmarshal([]byte(input), &attestation))
	require.Equal(t, spec.DataVersionPQ, attestation.Version)

	data, err := (&EventsAttestation{event: &attestation}).getData()
	require.NoError(t, err)

	require.Equal(t, "0x0b", data.GetAggregationBits())
	require.Empty(t, data.GetSignature())
	require.Equal(t, uint64(4070), data.GetData().GetSlot().GetValue())
	require.Equal(t, uint64(1), data.GetData().GetIndex().GetValue())
	require.Equal(t, "0x1111111111111111111111111111111111111111111111111111111111111111", data.GetData().GetBeaconBlockRoot())
	require.Equal(t, uint64(127), data.GetData().GetTarget().GetEpoch().GetValue())
	require.Equal(t, "0x3333333333333333333333333333333333333333333333333333333333333333", data.GetData().GetTarget().GetRoot())
}
