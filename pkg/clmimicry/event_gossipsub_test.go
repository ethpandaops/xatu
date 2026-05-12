package clmimicry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestMapGossipSubEventToXatuEvent(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		expected xatu.Event_Name
		wantErr  bool
	}{
		{
			name:     "beacon_block exact",
			topic:    "/eth2/abcd1234/beacon_block/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		},
		{
			name:     "beacon_aggregate_and_proof exact",
			topic:    "/eth2/abcd1234/beacon_aggregate_and_proof/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF,
		},
		{
			name:     "beacon_attestation subnet (prefix match)",
			topic:    "/eth2/abcd1234/beacon_attestation_5/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
		},
		{
			name:     "blob_sidecar subnet (prefix match)",
			topic:    "/eth2/abcd1234/blob_sidecar_0/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR,
		},
		{
			name:     "data_column_sidecar subnet (prefix match)",
			topic:    "/eth2/abcd1234/data_column_sidecar_127/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR,
		},
		{
			// Regression test for the substring-collision bug introduced by ePBS:
			// `execution_payload` and `execution_payload_bid` are both valid Gloas topics,
			// and the old strings.Contains-based resolver would non-deterministically
			// route bid messages into the envelope branch. Exact-match must keep these distinct.
			name:     "execution_payload (envelope) exact",
			topic:    "/eth2/abcd1234/execution_payload/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_ENVELOPE,
		},
		{
			name:     "execution_payload_bid exact (collision regression)",
			topic:    "/eth2/abcd1234/execution_payload_bid/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID,
		},
		{
			name:     "payload_attestation_message exact",
			topic:    "/eth2/abcd1234/payload_attestation_message/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE,
		},
		{
			name:     "proposer_preferences exact",
			topic:    "/eth2/abcd1234/proposer_preferences/ssz_snappy",
			expected: xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES,
		},
		{
			name:    "malformed topic (too few segments)",
			topic:   "missing/slashes",
			wantErr: true,
		},
		{
			name:    "unknown topic name",
			topic:   "/eth2/abcd1234/not_a_real_topic/ssz_snappy",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapGossipSubEventToXatuEvent(tt.topic)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.String(), got)
		})
	}
}
