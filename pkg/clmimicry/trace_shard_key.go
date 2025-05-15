package clmimicry

import (
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
		return event.PeerID.String()
	default:
		// Default to MsgID.
		return getMsgID(event.Payload)
	}
}
