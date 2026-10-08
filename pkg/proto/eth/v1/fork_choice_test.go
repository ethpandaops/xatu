package v1

import (
	"encoding/json"
	"testing"

	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func payloadStatusPtr(s eth2v1.ForkChoicePayloadStatus) *eth2v1.ForkChoicePayloadStatus {
	return &s
}

func testForkChoiceV2() *eth2v1.ForkChoiceV2 {
	justified := phase0.Checkpoint{Epoch: 3, Root: phase0.Root{0xa}}
	finalized := phase0.Checkpoint{Epoch: 2, Root: phase0.Root{0x9}}

	return &eth2v1.ForkChoiceV2{
		JustifiedCheckpoint: justified,
		FinalizedCheckpoint: finalized,
		ForkChoiceNodes: []*eth2v1.ForkChoiceNodeV2{
			{
				// a pending node whose parent is not retained in the tree
				Slot:                            29,
				BlockRoot:                       phase0.Root{0xb},
				PayloadStatus:                   eth2v1.ForkChoicePayloadStatusPending,
				ParentRoot:                      phase0.Root{0xa},
				JustifiedCheckpoint:             justified,
				FinalizedCheckpoint:             finalized,
				Weight:                          128,
				Validity:                        eth2v1.ForkChoiceNodeValidityValid,
				ExecutionBlockHash:              phase0.Hash32{0x1},
				PayloadAttesterCount:            512,
				PayloadAvailabilityYesCount:     498,
				PayloadDataAvailabilityYesCount: 497,
				ExtraData:                       map[string]any{"state_root": "0xc0"},
			},
			{
				// the block's empty node, pointing at its pending node
				Slot:                            29,
				BlockRoot:                       phase0.Root{0xb},
				PayloadStatus:                   eth2v1.ForkChoicePayloadStatusEmpty,
				ParentRoot:                      phase0.Root{0xb},
				ParentPayloadStatus:             payloadStatusPtr(eth2v1.ForkChoicePayloadStatusPending),
				JustifiedCheckpoint:             justified,
				FinalizedCheckpoint:             finalized,
				Weight:                          0,
				Validity:                        eth2v1.ForkChoiceNodeValidityOptimistic,
				ExecutionBlockHash:              phase0.Hash32{0x1},
				PayloadAttesterCount:            512,
				PayloadAvailabilityYesCount:     498,
				PayloadDataAvailabilityYesCount: 497,
				ExtraData:                       map[string]any{},
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
	assert.Nil(t, pending.GetParentPayloadStatus(), "parent not retained")
	assert.Equal(t, uint64(3), pending.GetJustifiedEpoch().GetValue())
	assert.Equal(t, uint64(2), pending.GetFinalizedEpoch().GetValue())
	assert.Equal(t, uint64(3), pending.GetJustifiedCheckpoint().GetEpoch().GetValue())
	assert.Equal(t, RootAsString(phase0.Root{0xa}), pending.GetJustifiedCheckpoint().GetRoot())
	assert.Equal(t, uint64(2), pending.GetFinalizedCheckpoint().GetEpoch().GetValue())
	assert.Equal(t, RootAsString(phase0.Root{0x9}), pending.GetFinalizedCheckpoint().GetRoot())
	assert.Equal(t, uint64(128), pending.GetWeight().GetValue())
	assert.Equal(t, uint64(512), pending.GetPayloadAttesterCount().GetValue())
	assert.Equal(t, uint64(498), pending.GetPayloadAvailabilityYesCount().GetValue())
	assert.Equal(t, uint64(497), pending.GetPayloadDataAvailabilityYesCount().GetValue())
	assert.Equal(t, RootAsString(phase0.Root{0x1}), pending.GetExecutionBlockHash())
	assert.JSONEq(t, `{"state_root":"0xc0"}`, pending.GetExtraData())

	empty := fc.GetForkChoiceNodes()[1]
	assert.Equal(t, uint32(0), empty.GetPayloadStatus().GetValue(), "PAYLOAD_STATUS_EMPTY")
	assert.Equal(t, uint32(2), empty.GetParentPayloadStatus().GetValue(), "PAYLOAD_STATUS_PENDING")
	assert.Equal(t, RootAsString(phase0.Root{0xb}), empty.GetParentRoot())
	assert.Equal(t, "optimistic", empty.GetValidity())
	require.NotNil(t, empty.GetWeight())
	assert.Equal(t, uint64(0), empty.GetWeight().GetValue())
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
	assert.Equal(t, phase0.Epoch(3), v1.ForkChoiceNodes[0].JustifiedEpoch)
	assert.Equal(t, phase0.Epoch(2), v1.ForkChoiceNodes[0].FinalizedEpoch)

	extra, err := json.Marshal(v1.ForkChoiceNodes[0].ExtraData)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"state_root": "0xc0",
		"payload_status": 2,
		"payload_attester_count": 512,
		"payload_availability_yes_count": 498,
		"payload_data_availability_yes_count": 497,
		"justified_checkpoint": {"epoch": "3", "root": "`+RootAsString(phase0.Root{0xa})+`"},
		"finalized_checkpoint": {"epoch": "2", "root": "`+RootAsString(phase0.Root{0x9})+`"}
	}`, string(extra))

	assert.Equal(t, uint32(2), v1.ForkChoiceNodes[1].ExtraData["parent_payload_status"])
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

// Decoded v2 values take precedence over a client's extra_data under the same keys.
func TestForkChoiceV2RoundTripDecodedValuesWin(t *testing.T) {
	forkChoice := testForkChoiceV2()
	forkChoice.ForkChoiceNodes[0].ExtraData = map[string]any{
		"payload_status":         "FULL",
		"payload_attester_count": "7",
		"justified_checkpoint":   "client value",
	}

	fc, err := NewForkChoiceV2FromGoEth2ClientV2(forkChoice)
	require.NoError(t, err)

	v1, err := fc.AsGoEth2ClientV1ForkChoice()
	require.NoError(t, err)

	extra := v1.ForkChoiceNodes[0].ExtraData
	assert.Equal(t, uint32(2), extra["payload_status"])
	assert.Equal(t, uint64(512), extra["payload_attester_count"])
	assert.Equal(t, map[string]any{checkpointEpochKey: "3", checkpointRootKey: RootAsString(phase0.Root{0xa})}, extra["justified_checkpoint"])
}

// A v1 node without extra data is encoded as JSON null; converting it back must not panic.
func TestForkChoiceNodeV2NullExtraData(t *testing.T) {
	fc, err := NewForkChoiceV2FromGoEth2ClientV1(&eth2v1.ForkChoice{
		ForkChoiceNodes: []*eth2v1.ForkChoiceNode{{Slot: 1, Validity: eth2v1.ForkChoiceNodeValidityValid}},
	})
	require.NoError(t, err)
	require.Equal(t, "null", fc.GetForkChoiceNodes()[0].GetExtraData())

	fc.GetForkChoiceNodes()[0].PayloadStatus = &wrapperspb.UInt32Value{Value: 1}

	v1, err := fc.AsGoEth2ClientV1ForkChoice()
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"payload_status": uint32(1)}, v1.ForkChoiceNodes[0].ExtraData)
}
