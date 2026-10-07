package v1

import (
	"encoding/json"

	eth2v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
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
			Epoch: phase0.Epoch(f.JustifiedCheckpoint.Epoch),
			Root:  justifiedRoot,
		},
		FinalizedCheckpoint: phase0.Checkpoint{
			Epoch: phase0.Epoch(f.FinalizedCheckpoint.Epoch),
			Root:  finalizedRoot,
		},
		ForkChoiceNodes: nodes,
	}, nil
}

// AsGoEth2ClientV1ForkChoice returns the fork choice in a go-eth2-client v1 fork choice format.
func (f *ForkChoiceV2) AsGoEth2ClientV1ForkChoice() (*eth2v1.ForkChoice, error) {
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

	var extraData map[string]any
	if f.GetExtraData() != "" {
		if err := json.Unmarshal([]byte(f.GetExtraData()), &extraData); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal extra_data")
		}
	}

	return &eth2v1.ForkChoice{
		JustifiedCheckpoint: phase0.Checkpoint{
			Epoch: phase0.Epoch(f.JustifiedCheckpoint.Epoch.GetValue()),
			Root:  justifiedRoot,
		},
		FinalizedCheckpoint: phase0.Checkpoint{
			Epoch: phase0.Epoch(f.FinalizedCheckpoint.Epoch.GetValue()),
			Root:  finalizedRoot,
		},
		ForkChoiceNodes: nodes,
		ExtraData:       extraData,
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

	extraData := make(map[string]any)

	err = json.Unmarshal([]byte(f.ExtraData), &extraData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal extra_data")
	}

	validity, err := eth2v1.ForkChoiceNodeValidityFromString(f.Validity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert validity")
	}

	return &eth2v1.ForkChoiceNode{
		Slot:               phase0.Slot(f.Slot),
		BlockRoot:          blockRoot,
		ParentRoot:         parentRoot,
		JustifiedEpoch:     phase0.Epoch(f.JustifiedEpoch),
		FinalizedEpoch:     phase0.Epoch(f.FinalizedEpoch),
		Weight:             f.Weight,
		Validity:           validity,
		ExecutionBlockHash: executionBlockHash,
		ExtraData:          extraData,
	}, nil
}

func (f *ForkChoiceNodeV2) AsGoEth2ClientV1ForkChoiceNode() (*eth2v1.ForkChoiceNode, error) {
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

	extraData := make(map[string]any)

	err = json.Unmarshal([]byte(f.ExtraData), &extraData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal extra_data")
	}

	// A node without extra data is encoded as JSON null, which decodes to a
	// nil map.
	if extraData == nil {
		extraData = make(map[string]any)
	}

	// Round-trip the EIP-7732 payload_status and the other v2 fork choice
	// fields into extra_data so downstream consumers that only inspect
	// extra_data still see them.
	if ps := f.GetPayloadStatus(); ps != nil {
		extraData["payload_status"] = ps.GetValue()
	}

	if ps := f.GetParentPayloadStatus(); ps != nil {
		extraData["parent_payload_status"] = ps.GetValue()
	}

	for key, value := range map[string]*wrapperspb.UInt64Value{
		"payload_attester_count":              f.GetPayloadAttesterCount(),
		"payload_availability_yes_count":      f.GetPayloadAvailabilityYesCount(),
		"payload_data_availability_yes_count": f.GetPayloadDataAvailabilityYesCount(),
	} {
		if _, exists := extraData[key]; !exists && value != nil {
			extraData[key] = value.GetValue()
		}
	}

	validity, err := eth2v1.ForkChoiceNodeValidityFromString(f.Validity)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert validity")
	}

	return &eth2v1.ForkChoiceNode{
		Slot:               phase0.Slot(f.Slot.GetValue()),
		BlockRoot:          blockRoot,
		ParentRoot:         parentRoot,
		JustifiedEpoch:     phase0.Epoch(f.JustifiedEpoch.GetValue()),
		FinalizedEpoch:     phase0.Epoch(f.FinalizedEpoch.GetValue()),
		Weight:             f.Weight.GetValue(),
		Validity:           validity,
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
		},
		JustifiedCheckpoint: &Checkpoint{
			Epoch: uint64(f.JustifiedCheckpoint.Epoch),
			Root:  RootAsString(f.JustifiedCheckpoint.Root),
		},
		ForkChoiceNodes: nodes,
	}, nil
}

func NewForkChoiceV2FromGoEth2ClientV1(f *eth2v1.ForkChoice) (*ForkChoiceV2, error) {
	extraData, err := marshalExtraData(f.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal extra_data")
	}

	nodes := []*ForkChoiceNodeV2{}

	for _, node := range f.ForkChoiceNodes {
		n, err := NewForkChoiceNodeV2FromGoEth2ClientV1(node)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert node")
		}

		nodes = append(nodes, n)
	}

	return &ForkChoiceV2{
		FinalizedCheckpoint: &CheckpointV2{
			Root: RootAsString(f.FinalizedCheckpoint.Root),
			Epoch: &wrapperspb.UInt64Value{
				Value: uint64(f.FinalizedCheckpoint.Epoch),
			},
		},
		JustifiedCheckpoint: &CheckpointV2{
			Root: RootAsString(f.JustifiedCheckpoint.Root),
			Epoch: &wrapperspb.UInt64Value{
				Value: uint64(f.JustifiedCheckpoint.Epoch),
			},
		},
		ForkChoiceNodes: nodes,
		ExtraData:       extraData,
	}, nil
}

func NewForkChoiceNodeFromGoEth2ClientV1(node *eth2v1.ForkChoiceNode) (*ForkChoiceNode, error) {
	extraData, err := json.Marshal(node.ExtraData)
	if err != nil {
		return nil, err
	}

	return &ForkChoiceNode{
		Slot:               uint64(node.Slot),
		BlockRoot:          RootAsString(node.BlockRoot),
		ParentRoot:         RootAsString(node.ParentRoot),
		JustifiedEpoch:     uint64(node.JustifiedEpoch),
		FinalizedEpoch:     uint64(node.FinalizedEpoch),
		Weight:             node.Weight,
		Validity:           node.Validity.String(),
		ExecutionBlockHash: RootAsString(node.ExecutionBlockHash),
		ExtraData:          string(extraData),
	}, nil
}

func NewForkChoiceNodeV2FromGoEth2ClientV1(node *eth2v1.ForkChoiceNode) (*ForkChoiceNodeV2, error) {
	extraData, err := json.Marshal(node.ExtraData)
	if err != nil {
		return nil, err
	}

	out := &ForkChoiceNodeV2{
		Slot:               &wrapperspb.UInt64Value{Value: uint64(node.Slot)},
		BlockRoot:          RootAsString(node.BlockRoot),
		ParentRoot:         RootAsString(node.ParentRoot),
		JustifiedEpoch:     &wrapperspb.UInt64Value{Value: uint64(node.JustifiedEpoch)},
		FinalizedEpoch:     &wrapperspb.UInt64Value{Value: uint64(node.FinalizedEpoch)},
		Weight:             &wrapperspb.UInt64Value{Value: node.Weight},
		Validity:           node.Validity.String(),
		ExecutionBlockHash: RootAsString(node.ExecutionBlockHash),
		ExtraData:          string(extraData),
	}

	if v, ok := payloadStatusFromExtraData(node.ExtraData); ok {
		out.PayloadStatus = &wrapperspb.UInt32Value{Value: v}
	}

	return out, nil
}

// payloadStatusFromExtraData reads the EIP-7732 per-node payload_status enum
// from the beacon API's fork_choice extra_data map. Gloas+ beacon nodes report
// it as a string ("EMPTY", "FULL", "PENDING") or an integer (0/1/2); upstream
// hasn't standardised the shape yet so we tolerate both. Returns false when
// the key isn't present (pre-Gloas nodes), or when the value can't be coerced.
func payloadStatusFromExtraData(extraData map[string]any) (uint32, bool) {
	raw, ok := extraData["payload_status"]
	if !ok {
		return 0, false
	}

	switch v := raw.(type) {
	case string:
		switch v {
		case "EMPTY", "empty", "0":
			return 0, true
		case "FULL", "full", "1":
			return 1, true
		case "PENDING", "pending", "2":
			return 2, true
		}
	case float64:
		if v >= 0 && v <= 2 {
			return uint32(v), true
		}
	case int:
		if v >= 0 && v <= 2 {
			return uint32(v), true
		}
	case uint64:
		if v <= 2 {
			return uint32(v), true
		}
	}

	return 0, false
}

// NewForkChoiceV2FromGoEth2ClientV2 converts a GET /eth/v2/debug/fork_choice
// response, with one node per (block root, payload status) pair, keeping the
// Gloas payload fields the v1 endpoint lacks.
func NewForkChoiceV2FromGoEth2ClientV2(f *eth2v1.ForkChoiceV2) (*ForkChoiceV2, error) {
	extraData, err := marshalExtraData(f.ExtraData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal extra_data")
	}

	nodes := make([]*ForkChoiceNodeV2, 0, len(f.ForkChoiceNodes))

	for _, node := range f.ForkChoiceNodes {
		n, err := NewForkChoiceNodeV2FromGoEth2ClientV2(node)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert node")
		}

		nodes = append(nodes, n)
	}

	return &ForkChoiceV2{
		FinalizedCheckpoint: &CheckpointV2{
			Root:  RootAsString(f.FinalizedCheckpoint.Root),
			Epoch: &wrapperspb.UInt64Value{Value: uint64(f.FinalizedCheckpoint.Epoch)},
		},
		JustifiedCheckpoint: &CheckpointV2{
			Root:  RootAsString(f.JustifiedCheckpoint.Root),
			Epoch: &wrapperspb.UInt64Value{Value: uint64(f.JustifiedCheckpoint.Epoch)},
		},
		ForkChoiceNodes: nodes,
		ExtraData:       extraData,
	}, nil
}

// NewForkChoiceNodeV2FromGoEth2ClientV2 converts a v2 debug fork choice node.
// Values the client does not provide are left unset.
func NewForkChoiceNodeV2FromGoEth2ClientV2(node *eth2v1.ForkChoiceNodeV2) (*ForkChoiceNodeV2, error) {
	extraData, err := json.Marshal(node.ExtraData)
	if err != nil {
		return nil, err
	}

	out := &ForkChoiceNodeV2{
		Slot:                            &wrapperspb.UInt64Value{Value: uint64(node.Slot)},
		BlockRoot:                       RootAsString(node.BlockRoot),
		ParentRoot:                      RootAsString(node.ParentRoot),
		Weight:                          &wrapperspb.UInt64Value{Value: node.Weight},
		Validity:                        node.Validity.String(),
		ExecutionBlockHash:              RootAsString(phase0.Root(node.ExecutionBlockHash)),
		ExtraData:                       string(extraData),
		PayloadAttesterCount:            optionalUint64Value(node.PayloadAttesterCount),
		PayloadAvailabilityYesCount:     optionalUint64Value(node.PayloadAvailabilityYesCount),
		PayloadDataAvailabilityYesCount: optionalUint64Value(node.PayloadDataAvailabilityYesCount),
	}

	if node.JustifiedEpoch != nil {
		out.JustifiedEpoch = &wrapperspb.UInt64Value{Value: uint64(*node.JustifiedEpoch)}
	}

	if node.FinalizedEpoch != nil {
		out.FinalizedEpoch = &wrapperspb.UInt64Value{Value: uint64(*node.FinalizedEpoch)}
	}

	if v, ok := payloadStatusValue(node.PayloadStatus); ok {
		out.PayloadStatus = &wrapperspb.UInt32Value{Value: v}
	}

	if node.ParentPayloadStatus != nil {
		if v, ok := payloadStatusValue(*node.ParentPayloadStatus); ok {
			out.ParentPayloadStatus = &wrapperspb.UInt32Value{Value: v}
		}
	}

	return out, nil
}

// payloadStatusValue maps a beacon API payload status onto the spec's
// PayloadStatus enum (PAYLOAD_STATUS_EMPTY = 0, FULL = 1, PENDING = 2).
func payloadStatusValue(status eth2v1.ForkChoicePayloadStatus) (uint32, bool) {
	switch status {
	case eth2v1.ForkChoicePayloadStatusEmpty:
		return 0, true
	case eth2v1.ForkChoicePayloadStatusFull:
		return 1, true
	case eth2v1.ForkChoicePayloadStatusPending:
		return 2, true
	default:
		return 0, false
	}
}

func optionalUint64Value(value *uint64) *wrapperspb.UInt64Value {
	if value == nil {
		return nil
	}

	return &wrapperspb.UInt64Value{Value: *value}
}

// marshalExtraData encodes a fork choice store's extra data, leaving it empty
// when the client provides none.
func marshalExtraData(extraData map[string]any) (string, error) {
	if len(extraData) == 0 {
		return "", nil
	}

	encoded, err := json.Marshal(extraData)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
