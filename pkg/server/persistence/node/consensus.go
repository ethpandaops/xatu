package node

import "time"

type Consensus struct {
	// ConsensusID is the consensus id.
	ConsensusID int64 `json:"consensusId" db:"consensus_id"`
	// Enr is the enr of the node record.
	Enr string `json:"enr" db:"enr" fieldopt:"omitempty"`
	// CreateTime is the timestamp of when the consensus record was created.
	CreateTime time.Time `json:"createTime" db:"create_time" fieldopt:"omitempty"`
	// Name is the name of the node.
	Name string `json:"name" db:"name" fieldopt:"omitempty"`
	// ForkDigest is the fork digest of the node.
	ForkDigest []byte `json:"forkDigest" db:"fork_digest" fieldopt:"omitempty"`
	// FinalizedRoot is the finalized root of the node.
	FinalizedRoot []byte `json:"finalizedRoot" db:"finalized_root" fieldopt:"omitempty"`
	// FinalizedEpoch is the finalized epoch of the node.
	FinalizedEpoch []byte `json:"finalizedEpoch" db:"finalized_epoch" fieldopt:"omitempty"`
	// HeadRoot is the head root of the node.
	HeadRoot []byte `json:"headRoot" db:"head_root" fieldopt:"omitempty"`
	// HeadSlot is the head slot of the node.
	HeadSlot []byte `json:"headSlot" db:"head_slot" fieldopt:"omitempty"`
	// CSC is the custody subnet count of the node.
	CSC []byte `json:"csc" db:"csc" fieldopt:"omitempty"`
	// NetworkID is the network id of the node.
	NetworkID string `json:"networkId" db:"network_id" fieldopt:"omitempty"`
}
