package libp2p

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcMetaControlPruneEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pRpcMetaControlPruneTableName,
		libp2pRpcMetaControlPruneEventNames,
		func() route.ColumnarBatch { return newlibp2pRpcMetaControlPruneBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pRpcMetaControlPruneBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceRpcMetaControlPrune() == nil {
		return fmt.Errorf("nil libp2p_trace_rpc_meta_control_prune payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.rows++

	return nil
}

func (b *libp2pRpcMetaControlPruneBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint32 values are bounded by ClickHouse column schema
func (b *libp2pRpcMetaControlPruneBatch) appendPayload(
	event *xatu.DecoratedEvent,
) {
	payload := event.GetLibp2PTraceRpcMetaControlPrune()
	// Extract values needed for key computations.
	rootEventID := wrappedStringValue(payload.GetRootEventId())

	var controlIdxVal, peerIdxVal uint32
	if idx := payload.GetControlIndex(); idx != nil {
		controlIdxVal = idx.GetValue()
	}

	if idx := payload.GetPeerIndex(); idx != nil {
		peerIdxVal = idx.GetValue()
	}

	// Compute unique_key as composite key matching Vector's VRL format.
	if rootEventID != "" {
		b.UniqueKey.Append(computeRPCMetaChildUniqueKey(
			rootEventID, "rpc_meta_control_prune",
			strconv.FormatUint(uint64(controlIdxVal), 10),
			strconv.FormatUint(uint64(peerIdxVal), 10)))
	} else {
		b.UniqueKey.Append(0)
	}

	// Compute rpc_meta_unique_key from root_event_id.
	if rootEventID != "" {
		b.RPCMetaUniqueKey.Append(computeRPCMetaUniqueKey(rootEventID))
	} else {
		b.RPCMetaUniqueKey.Append(0)
	}

	b.ControlIndex.Append(int32(controlIdxVal))
	b.PeerIDIndex.Append(int32(peerIdxVal))

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()

	peerID := wrappedStringValue(payload.GetPeerId())
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))

	// Compute graft_peer_id_unique_key (nullable).
	graftPeerID := wrappedStringValue(payload.GetGraftPeerId())
	if graftPeerID != "" {
		b.GraftPeerIDUniqueKey.Append(proto.NewNullable[int64](computePeerIDUniqueKey(graftPeerID, networkName)))
	} else {
		b.GraftPeerIDUniqueKey.Append(proto.NewNullable[int64](route.SeaHashInt64(networkName)))
	}

	// Parse topic fields.
	if topic := wrappedStringValue(payload.GetTopic()); topic != "" {
		parsed := parseTopicFields(topic)
		b.TopicLayer.Append(parsed.Layer)
		b.TopicForkDigestValue.Append(parsed.ForkDigestValue)
		b.TopicName.Append(parsed.Name)
		b.TopicEncoding.Append(parsed.Encoding)
	} else {
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
	}
}
