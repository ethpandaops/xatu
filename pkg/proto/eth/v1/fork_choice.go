package v1

import (
	"encoding/json"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// AsGoEth2ClientV1ForkChoice returns the fork choice in a go-eth2-client v1 fork choice format.
func (f *ForkChoice) AsGoEth2ClientV1ForkChoice() (*eth2v1.ForkChoice, error) {
	justifiedRoot, err := StringToRoot(f.JustifiedCheckpoint.Root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert justified_checkpoint.root")
	}

	finalizedRoot, err := StringToRoot(f.FinalizedCheckpoint.Root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert finalized_checkpoint.root")
	}

	nodes := []*eth2v1.ForkChoiceNode{}

	for _, node := range f.ForkChoiceNodes {
		node, err := node.AsGoEth2ClientV1ForkChoiceNode()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert node")
		}

		nodes = append(nodes, node)
	}

	return &eth2v1.ForkChoice{
		JustifiedCheckpoint: phase0.Checkpoint{
			Epoch: phase0.Epoch(f.JustifiedCheckpoint.EpochV2.Value),
			Root:  justifiedRoot,
		},
		FinalizedCheckpoint: phase0.Checkpoint{
			Epoch: phase0.Epoch(f.FinalizedCheckpoint.EpochV2.Value),
			Root:  finalizedRoot,
		},
		ForkChoiceNodes: nodes,
	}, nil
}

func (f *ForkChoiceNode) AsGoEth2ClientV1ForkChoiceNode() (*eth2v1.ForkChoiceNode, error) {
	blockRoot, err := StringToRoot(f.BlockRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert block_root")
	}

	parentRoot, err := StringToRoot(f.ParentRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert parent_root")
	}

	executionBlockHash, err := StringToRoot(f.ExecutionBlockHash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert execution_block_hash")
	}

	extraData := make(map[string]interface{})

	err = json.Unmarshal([]byte(f.ExtraData), &extraData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal extra_data")
	}

	return &eth2v1.ForkChoiceNode{
		Slot:               phase0.Slot(f.SlotV2.Value),
		BlockRoot:          blockRoot,
		ParentRoot:         parentRoot,
		JustifiedEpoch:     phase0.Epoch(f.JustifiedEpochV2.Value),
		FinalizedEpoch:     phase0.Epoch(f.FinalizedEpochV2.Value),
		Weight:             f.WeightV2.Value,
		Validity:           eth2v1.ForkChoiceNodeValidity(f.Validity),
		ExecutionBlockHash: executionBlockHash,
		ExtraData:          extraData,
	}, nil
}

func NewForkChoiceFromGoEth2ClientV1(f *eth2v1.ForkChoice) (*ForkChoice, error) {
	nodes := []*ForkChoiceNode{}

	for _, node := range f.ForkChoiceNodes {
		n, err := NewForkChoiceNodeFromGoEth2ClientV1(node)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert node")
		}

		nodes = append(nodes, n)
	}

	return &ForkChoice{
		FinalizedCheckpoint: &Checkpoint{
			Epoch: uint64(f.FinalizedCheckpoint.Epoch),
			Root:  RootAsString(f.FinalizedCheckpoint.Root),
			EpochV2: &wrapperspb.UInt64Value{
				Value: uint64(f.FinalizedCheckpoint.Epoch),
			},
		},
		JustifiedCheckpoint: &Checkpoint{
			Epoch: uint64(f.JustifiedCheckpoint.Epoch),
			Root:  RootAsString(f.JustifiedCheckpoint.Root),
			EpochV2: &wrapperspb.UInt64Value{
				Value: uint64(f.JustifiedCheckpoint.Epoch),
			},
		},
		ForkChoiceNodes: nodes,
	}, nil
}

func NewForkChoiceNodeFromGoEth2ClientV1(node *eth2v1.ForkChoiceNode) (*ForkChoiceNode, error) {
	extraData, err := json.Marshal(node.ExtraData)
	if err != nil {
		return nil, err
	}

	return &ForkChoiceNode{
		Slot:               uint64(node.Slot),
		SlotV2:             &wrapperspb.UInt64Value{Value: uint64(node.Slot)},
		BlockRoot:          RootAsString(node.BlockRoot),
		ParentRoot:         RootAsString(node.ParentRoot),
		JustifiedEpoch:     uint64(node.JustifiedEpoch),
		JustifiedEpochV2:   &wrapperspb.UInt64Value{Value: uint64(node.JustifiedEpoch)},
		FinalizedEpoch:     uint64(node.FinalizedEpoch),
		FinalizedEpochV2:   &wrapperspb.UInt64Value{Value: uint64(node.FinalizedEpoch)},
		Weight:             node.Weight,
		WeightV2:           &wrapperspb.UInt64Value{Value: node.Weight},
		Validity:           string(node.Validity),
		ExecutionBlockHash: RootAsString(node.ExecutionBlockHash),
		ExtraData:          string(extraData),
	}, nil
}
