package horizon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Location represents a Horizon location record in the database.
type Location struct {
	// LocationID is the location id.
	LocationID any `json:"locationId" db:"location_id"`
	// CreateTime is the timestamp of when the record was created.
	CreateTime time.Time `json:"createTime" db:"create_time" fieldopt:"omitempty"`
	// UpdateTime is the timestamp of when the record was updated.
	UpdateTime time.Time `json:"updateTime" db:"update_time" fieldopt:"omitempty"`
	// NetworkID is the network id of the location.
	NetworkID string `json:"networkId" db:"network_id"`
	// Type is the type of the location.
	Type string `json:"type" db:"type"`
	// HeadSlot is the current head slot position for real-time tracking.
	HeadSlot uint64 `json:"headSlot" db:"head_slot"`
	// FillSlot is the fill slot position for catch-up processing.
	FillSlot uint64 `json:"fillSlot" db:"fill_slot"`
}

// Marshal marshals a proto HorizonLocation message into the Location fields.
func (l *Location) Marshal(msg *xatu.HorizonLocation) error {
	if msg == nil {
		return fmt.Errorf("horizon location message is nil")
	}

	l.NetworkID = msg.NetworkId
	l.Type = msg.Type.String()
	l.HeadSlot = msg.HeadSlot
	l.FillSlot = msg.FillSlot

	return nil
}

// Unmarshal unmarshals the Location into a proto HorizonLocation message.
func (l *Location) Unmarshal() (*xatu.HorizonLocation, error) {
	horizonType, ok := xatu.HorizonType_value[l.Type]
	if !ok {
		return nil, fmt.Errorf("unknown horizon type: %s", l.Type)
	}

	return &xatu.HorizonLocation{
		NetworkId: l.NetworkID,
		Type:      xatu.HorizonType(horizonType),
		HeadSlot:  l.HeadSlot,
		FillSlot:  l.FillSlot,
	}, nil
}
