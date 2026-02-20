package libp2p

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	xatuProto "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pHandleMetadataEventNames = []xatuProto.Event_Name{
	xatuProto.Event_LIBP2P_TRACE_HANDLE_METADATA,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pHandleMetadataTableName,
		libp2pHandleMetadataEventNames,
		func() flattener.ColumnarBatch { return newlibp2pHandleMetadataBatch() },
	))
}

func (b *libp2pHandleMetadataBatch) FlattenTo(
	event *xatuProto.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event, meta)
	b.rows++

	return nil
}

func (b *libp2pHandleMetadataBatch) appendRuntime(event *xatuProto.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pHandleMetadataBatch) appendPayload(
	event *xatuProto.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceHandleMetadata()
	if payload == nil {
		b.PeerIDUniqueKey.Append(0)
		b.Error.Append(proto.Nullable[string]{})
		b.Protocol.Append("")
		b.Direction.Append(proto.Nullable[string]{})
		b.Attnets.Append("")
		b.SeqNumber.Append(0)
		b.Syncnets.Append("")
		b.CustodyGroupCount.Append(proto.Nullable[uint8]{})
		b.LatencyMilliseconds.Append(0)

		return
	}

	// Error (nullable string).
	if errVal := wrappedStringValue(payload.GetError()); errVal != "" {
		b.Error.Append(proto.NewNullable[string](errVal))
	} else {
		b.Error.Append(proto.Nullable[string]{})
	}

	b.Protocol.Append(wrappedStringValue(payload.GetProtocolId()))

	// Direction (nullable string).
	if dir := wrappedStringValue(payload.GetDirection()); dir != "" {
		b.Direction.Append(proto.NewNullable[string](dir))
	} else {
		b.Direction.Append(proto.Nullable[string]{})
	}

	// Latency: proto stores seconds as float64; convert to Decimal(10,3) ms.
	// Truncate (not round) to match Vector's VRL behaviour.
	if latency := payload.GetLatency(); latency != nil {
		b.LatencyMilliseconds.Append(proto.Decimal64(int64(latency.GetValue() * 1_000_000)))
	} else {
		b.LatencyMilliseconds.Append(0)
	}

	// Extract metadata sub-message fields.
	if md := payload.GetMetadata(); md != nil {
		if seqNum := md.GetSeqNumber(); seqNum != nil {
			b.SeqNumber.Append(seqNum.GetValue())
		} else {
			b.SeqNumber.Append(0)
		}

		b.Attnets.Append(wrappedStringValue(md.GetAttnets()))
		b.Syncnets.Append(wrappedStringValue(md.GetSyncnets()))

		if cgc := md.GetCustodyGroupCount(); cgc != nil {
			b.CustodyGroupCount.Append(proto.NewNullable[uint8](uint8(cgc.GetValue())))
		} else {
			b.CustodyGroupCount.Append(proto.Nullable[uint8]{})
		}
	} else {
		b.SeqNumber.Append(0)
		b.Attnets.Append("")
		b.Syncnets.Append("")
		b.CustodyGroupCount.Append(proto.Nullable[uint8]{})
	}

	// Compute peer_id_unique_key: prefer payload peer ID, fall back to metadata.
	peerID := wrappedStringValue(payload.GetPeerId())
	if peerID == "" {
		peerID = peerIDFromMetadata(event, func(c *xatuProto.ClientMeta) peerIDMetadataProvider {
			return c.GetLibp2PTraceHandleMetadata()
		})
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
