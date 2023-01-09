package node

import "time"

type Activity struct {
	// ActivityID is the activity id.
	ActivityID interface{} `json:"activity_id" db:"activity_id"`
	// Enr is the enr of the node record.
	Enr string `json:"enr" db:"enr" fieldopt:"omitempty"`
	// Client is the name of the coordinated client.
	ClientID string `json:"client_id" db:"client_id" fieldopt:"omitempty"`
	// CreateTime is the timestamp of when the activity record was created.
	CreateTime time.Time `json:"create_time" db:"create_time" fieldopt:"omitempty"`
	// UpdateTime is the timestamp of when the activity record was updated.
	UpdateTime time.Time `json:"update_time" db:"update_time" fieldopt:"omitempty"`
	// Connected is the connected status of the node.
	Connected bool `json:"connected" db:"connected" fieldopt:"omitempty"`
}
