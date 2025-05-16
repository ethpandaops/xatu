package clmimicry

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/probe-lab/hermes/host"
)

// ShardingKeyType represents the different types of sharding keys.
type ShardingKeyType string

const (
	// ShardingKeyTypeMsgID uses the message ID for sharding.
	ShardingKeyTypeMsgID ShardingKeyType = "MsgID"
	// ShardingKeyTypePeerID uses the peer ID for sharding.
	ShardingKeyTypePeerID ShardingKeyType = "PeerID"
)

// GetShardingKey extracts the appropriate sharding key based on the configured type.
func GetShardingKey(
	event *host.TraceEvent,
	clientMeta *xatu.ClientMeta,
	shardingKeyType string,
) string {
	switch ShardingKeyType(shardingKeyType) {
	case ShardingKeyTypePeerID:
		// The PeerID is the remote peer ID from the payload, where as the PeerID on the root event
		// is the host peer ID.  We always want to use the remote peer ID for sharding.
		if payload, ok := event.Payload.(map[string]any); ok {
			if remotePeerID, found := payload["PeerID"]; found {
				return fmt.Sprintf("%v", remotePeerID)
			}
		}

		// We'll just default to the host peer ID if we can't extract the remote peer ID.
		// This will mean the event won't be sharded correctly, but will parse fine and
		// show in metrics.
		return event.PeerID.String()
	default:
		// Default to MsgID.
		return getMsgID(event.Payload)
	}
}
