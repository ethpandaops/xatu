package relaymonitor

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/encoding/protojson"
)

type Location struct {
	// LocationID is the location id.
	LocationID any `json:"locationId" db:"location_id"`
	// CreateTime is the timestamp of when the record was created.
	CreateTime time.Time `json:"createTime" db:"create_time" fieldopt:"omitempty"`
	// UpdateTime is the timestamp of when the record was updated.
	UpdateTime time.Time `json:"updateTime" db:"update_time" fieldopt:"omitempty"`
	// MetaNetworkName is the network name of the location.
	MetaNetworkName string `json:"metaNetworkName" db:"meta_network_name"`
	// MetaClientName is the client instance name.
	MetaClientName string `json:"metaClientName" db:"meta_client_name"`
	// Type is the type of the location.
	Type string `json:"type" db:"type"`
	// RelayName is the name of the relay.
	RelayName string `json:"relayName" db:"relay_name"`
	// Value is the value of the location.
	Value string `json:"value" db:"value"`
}

var (
	ErrFailedToMarshal   = errors.New("failed to marshal location")
	ErrFailedToUnmarshal = errors.New("failed to unmarshal location")
)

// Marshal marshals a proto message into the Location fields.
func (l *Location) Marshal(msg *xatu.RelayMonitorLocation) error {
	l.MetaNetworkName = msg.MetaNetworkName
	l.MetaClientName = msg.MetaClientName
	l.RelayName = msg.RelayName

	switch msg.Type {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		l.Type = "RELAY_MONITOR_BID_TRACE"

		data := msg.GetBidTrace()
		if data != nil {
			b, err := protojson.Marshal(data)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
			}

			l.Value = string(b)
		}

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		l.Type = "RELAY_MONITOR_PAYLOAD_DELIVERED"

		data := msg.GetPayloadDelivered()
		if data != nil {
			b, err := protojson.Marshal(data)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
			}

			l.Value = string(b)
		}

	default:
		return fmt.Errorf("unknown type: %s", msg.Type)
	}

	return nil
}

// Unmarshal unmarshals the Location into a proto message.
func (l *Location) Unmarshal() (*xatu.RelayMonitorLocation, error) {
	msg := &xatu.RelayMonitorLocation{
		MetaNetworkName: l.MetaNetworkName,
		MetaClientName:  l.MetaClientName,
		RelayName:       l.RelayName,
	}

	switch l.Type {
	case "RELAY_MONITOR_BID_TRACE":
		msg.Type = xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE

		data := &xatu.RelayMonitorLocationBidTrace{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.RelayMonitorLocation_BidTrace{
			BidTrace: data,
		}

	case "RELAY_MONITOR_PAYLOAD_DELIVERED":
		msg.Type = xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED

		data := &xatu.RelayMonitorLocationPayloadDelivered{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.RelayMonitorLocation_PayloadDelivered{
			PayloadDelivered: data,
		}

	default:
		return nil, fmt.Errorf("unknown type: %s", l.Type)
	}

	return msg, nil
}
