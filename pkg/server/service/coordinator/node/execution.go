package node

import "time"

type Execution struct {
	// ExecutionID is the execution id.
	ExecutionID int64 `json:"execution_id" db:"execution_id"`
	// Enr is the enr of the node record.
	Enr string `json:"enr" db:"enr" fieldopt:"omitempty"`
	// CreateTime is the timestamp of when the execution record was created.
	CreateTime time.Time `json:"create_time" db:"create_time" fieldopt:"omitempty"`
	// Name is the name of the node.
	Name string `json:"name" db:"name" fieldopt:"omitempty"`
	// Capabilities is the capabilities of the node.
	Capabilities string `json:"capabilities" db:"capabilities" fieldopt:"omitempty"`
	// ProtocolVersion is the protocol version of the node.
	ProtocolVersion string `json:"protocol_version" db:"protocol_version" fieldopt:"omitempty"`
	// NetworkId is the network id of the node.
	NetworkID string `json:"network_id" db:"network_id" fieldopt:"omitempty"`
	// TD is the total difficulty of the node.
	TotalDifficulty string `json:"total_difficulty" db:"total_difficulty" fieldopt:"omitempty"`
	// Head is the head of the node.
	Head []byte `json:"head" db:"head" fieldopt:"omitempty"`
	// Genesis is the genesis of the node.
	Genesis []byte `json:"genesis" db:"genesis" fieldopt:"omitempty"`
	// ForkIdHash is the fork id hash of the node.
	ForkIDHash []byte `json:"fork_id_hash" db:"fork_id_hash" fieldopt:"omitempty"`
	// ForkIdNext is the fork id next of the node.
	ForkIDNext string `json:"fork_id_next" db:"fork_id_next" fieldopt:"omitempty"`
}
