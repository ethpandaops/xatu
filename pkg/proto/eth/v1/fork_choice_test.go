package v1

import (
	"encoding/json"
	"testing"

	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func payloadStatusPtr(s eth2v1.ForkChoicePayloadStatus) *eth2v1.ForkChoicePayloadStatus {
	return &s
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func epochPtr(v phase0.Epoch) *phase0.Epoch {
	return &v
}

func testForkChoiceV2() *eth2v1.ForkChoiceV2 {
	return &eth2v1.ForkChoiceV2{
		JustifiedCheckpoint: phase0.Checkpoint{Epoch: 3, Root: phase0.Root{0xa}},
		FinalizedCheckpoint: phase0.Checkpoint{Epoch: 2, Root: phase0.Root{0x9}},
		ForkChoiceNodes: []*eth2v1.ForkChoiceNodeV2{
			{
				// a client that implements the spec shape
				Slot:                            29,
				BlockRoot:                       phase0.Root{0xb},
				PayloadStatus:                   eth2v1.ForkChoicePayloadStatusPending,
				ParentRoot:                      phase0.Root{0xa},
				ParentPayloadStatus:             payloadStatusPtr(eth2v1.ForkChoicePayloadStatusFull),
				JustifiedEpoch:                  epochPtr(3),
				FinalizedEpoch:                  epochPtr(2),
				Weight:                          128,
				Validity:                        eth2v1.ForkChoiceNodeValidityValid,
				ExecutionBlockHash:              phase0.Hash32{0x1},
				PayloadAttesterCount:            uint64Ptr(512),
				PayloadAvailabilityYesCount:     uint64Ptr(498),
				PayloadDataAvailabilityYesCount: uint64Ptr(497),
				ExtraData:                       map[string]any{"state_root": "0xc0"},
			},
			{
				// a client that leaves the optional fields out
				Slot:               29,
				BlockRoot:          phase0.Root{0xb},
				PayloadStatus:      eth2v1.ForkChoicePayloadStatusEmpty,
				ParentRoot:         phase0.Root{0xa},
				Weight:             0,
				Validity:           eth2v1.ForkChoiceNodeValidityValid,
				ExecutionBlockHash: phase0.Hash32{0x1},
			},
		},
		ExtraData: map[string]any{"proposer_boost_root": "0xb0"},
	}
}

func TestNewForkChoiceV2FromGoEth2ClientV2(t *testing.T) {
	fc, err := NewForkChoiceV2FromGoEth2ClientV2(testForkChoiceV2())
	require.NoError(t, err)

	assert.Equal(t, uint64(3), fc.GetJustifiedCheckpoint().GetEpoch().GetValue())
	assert.Equal(t, uint64(2), fc.GetFinalizedCheckpoint().GetEpoch().GetValue())
	assert.JSONEq(t, `{"proposer_boost_root":"0xb0"}`, fc.GetExtraData())
	require.Len(t, fc.GetForkChoiceNodes(), 2)

	pending := fc.GetForkChoiceNodes()[0]
	assert.Equal(t, uint32(2), pending.GetPayloadStatus().GetValue(), "PAYLOAD_STATUS_PENDING")
	assert.Equal(t, uint32(1), pending.GetParentPayloadStatus().GetValue(), "PAYLOAD_STATUS_FULL")
	assert.Equal(t, uint64(3), pending.GetJustifiedEpoch().GetValue())
	assert.Equal(t, uint64(2), pending.GetFinalizedEpoch().GetValue())
	assert.Equal(t, uint64(512), pending.GetPayloadAttesterCount().GetValue())
	assert.Equal(t, uint64(498), pending.GetPayloadAvailabilityYesCount().GetValue())
	assert.Equal(t, uint64(497), pending.GetPayloadDataAvailabilityYesCount().GetValue())
	assert.Equal(t, RootAsString(phase0.Root{0x1}), pending.GetExecutionBlockHash())
	assert.JSONEq(t, `{"state_root":"0xc0"}`, pending.GetExtraData())

	empty := fc.GetForkChoiceNodes()[1]
	assert.NotNil(t, empty.GetPayloadStatus())
	assert.Equal(t, uint32(0), empty.GetPayloadStatus().GetValue(), "PAYLOAD_STATUS_EMPTY")
	assert.Nil(t, empty.GetParentPayloadStatus())
	assert.Nil(t, empty.GetJustifiedEpoch())
	assert.Nil(t, empty.GetFinalizedEpoch())
	assert.Nil(t, empty.GetPayloadAttesterCount())
}

// Consumers that convert to the go-eth2-client v1 shape still see the v2
// fields, in extra_data.
func TestForkChoiceV2RoundTripsV2FieldsIntoExtraData(t *testing.T) {
	fc, err := NewForkChoiceV2FromGoEth2ClientV2(testForkChoiceV2())
	require.NoError(t, err)

	v1, err := fc.AsGoEth2ClientV1ForkChoice()
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"proposer_boost_root": "0xb0"}, v1.ExtraData)
	require.Len(t, v1.ForkChoiceNodes, 2)

	extra, err := json.Marshal(v1.ForkChoiceNodes[0].ExtraData)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"state_root": "0xc0",
		"payload_status": 2,
		"parent_payload_status": 1,
		"payload_attester_count": 512,
		"payload_availability_yes_count": 498,
		"payload_data_availability_yes_count": 497
	}`, string(extra))

	assert.Equal(t, map[string]any{"payload_status": uint32(0)}, v1.ForkChoiceNodes[1].ExtraData)
}

func TestNewForkChoiceV2FromGoEth2ClientV1KeepsStoreExtraData(t *testing.T) {
	fc, err := NewForkChoiceV2FromGoEth2ClientV1(&eth2v1.ForkChoice{
		JustifiedCheckpoint: phase0.Checkpoint{Epoch: 3},
		FinalizedCheckpoint: phase0.Checkpoint{Epoch: 2},
		ForkChoiceNodes:     []*eth2v1.ForkChoiceNode{},
		ExtraData: map[string]any{
			"unrealized_justified_checkpoint": map[string]any{"epoch": "4", "root": "0xd0"},
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"unrealized_justified_checkpoint":{"epoch":"4","root":"0xd0"}}`, fc.GetExtraData())

	fc, err = NewForkChoiceV2FromGoEth2ClientV1(&eth2v1.ForkChoice{ForkChoiceNodes: []*eth2v1.ForkChoiceNode{}})
	require.NoError(t, err)
	assert.Empty(t, fc.GetExtraData())
}
