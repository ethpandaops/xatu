package libp2p

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	xatuProto "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pHandleStatusEventNames = []xatuProto.Event_Name{
	xatuProto.Event_LIBP2P_TRACE_HANDLE_STATUS,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pHandleStatusTableName,
		libp2pHandleStatusEventNames,
		func() flattener.ColumnarBatch { return newlibp2pHandleStatusBatch() },
	))
}

func (b *libp2pHandleStatusBatch) FlattenTo(
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

func (b *libp2pHandleStatusBatch) appendRuntime(event *xatuProto.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pHandleStatusBatch) appendPayload(
	event *xatuProto.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceHandleStatus()
	if payload == nil {
		b.PeerIDUniqueKey.Append(0)
		b.Error.Append(proto.Nullable[string]{})
		b.Protocol.Append("")
		b.Direction.Append(proto.Nullable[string]{})
		b.RequestForkDigest.Append("")
		b.RequestFinalizedRoot.Append(proto.Nullable[string]{})
		b.RequestHeadRoot.Append(proto.Nullable[[]byte]{})
		b.RequestFinalizedEpoch.Append(proto.Nullable[uint32]{})
		b.RequestHeadSlot.Append(proto.Nullable[uint32]{})
		b.RequestEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
		b.ResponseForkDigest.Append("")
		b.ResponseFinalizedRoot.Append(proto.Nullable[[]byte]{})
		b.ResponseHeadRoot.Append(proto.Nullable[[]byte]{})
		b.ResponseFinalizedEpoch.Append(proto.Nullable[uint32]{})
		b.ResponseHeadSlot.Append(proto.Nullable[uint32]{})
		b.ResponseEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
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

	// Extract request status fields.
	if req := payload.GetRequest(); req != nil {
		b.RequestForkDigest.Append(wrappedStringValue(req.GetForkDigest()))

		if root := wrappedStringValue(req.GetFinalizedRoot()); root != "" {
			b.RequestFinalizedRoot.Append(proto.NewNullable[string](root))
		} else {
			b.RequestFinalizedRoot.Append(proto.Nullable[string]{})
		}

		if headRoot := wrappedStringValue(req.GetHeadRoot()); headRoot != "" {
			b.RequestHeadRoot.Append(proto.NewNullable[[]byte]([]byte(headRoot)))
		} else {
			b.RequestHeadRoot.Append(proto.Nullable[[]byte]{})
		}

		if epoch := req.GetFinalizedEpoch(); epoch != nil {
			b.RequestFinalizedEpoch.Append(proto.NewNullable[uint32](uint32(epoch.GetValue())))
		} else {
			b.RequestFinalizedEpoch.Append(proto.Nullable[uint32]{})
		}

		if slot := req.GetHeadSlot(); slot != nil {
			b.RequestHeadSlot.Append(proto.NewNullable[uint32](uint32(slot.GetValue())))
		} else {
			b.RequestHeadSlot.Append(proto.Nullable[uint32]{})
		}

		if earliest := req.GetEarliestAvailableSlot(); earliest != nil {
			b.RequestEarliestAvailableSlot.Append(proto.NewNullable[uint32](uint32(earliest.GetValue())))
		} else {
			b.RequestEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
		}
	} else {
		b.RequestForkDigest.Append("")
		b.RequestFinalizedRoot.Append(proto.Nullable[string]{})
		b.RequestHeadRoot.Append(proto.Nullable[[]byte]{})
		b.RequestFinalizedEpoch.Append(proto.Nullable[uint32]{})
		b.RequestHeadSlot.Append(proto.Nullable[uint32]{})
		b.RequestEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
	}

	// Extract response status fields.
	if resp := payload.GetResponse(); resp != nil {
		b.ResponseForkDigest.Append(wrappedStringValue(resp.GetForkDigest()))

		if root := wrappedStringValue(resp.GetFinalizedRoot()); root != "" {
			b.ResponseFinalizedRoot.Append(proto.NewNullable[[]byte]([]byte(root)))
		} else {
			b.ResponseFinalizedRoot.Append(proto.Nullable[[]byte]{})
		}

		if headRoot := wrappedStringValue(resp.GetHeadRoot()); headRoot != "" {
			b.ResponseHeadRoot.Append(proto.NewNullable[[]byte]([]byte(headRoot)))
		} else {
			b.ResponseHeadRoot.Append(proto.Nullable[[]byte]{})
		}

		if epoch := resp.GetFinalizedEpoch(); epoch != nil {
			b.ResponseFinalizedEpoch.Append(proto.NewNullable[uint32](uint32(epoch.GetValue())))
		} else {
			b.ResponseFinalizedEpoch.Append(proto.Nullable[uint32]{})
		}

		if slot := resp.GetHeadSlot(); slot != nil {
			b.ResponseHeadSlot.Append(proto.NewNullable[uint32](uint32(slot.GetValue())))
		} else {
			b.ResponseHeadSlot.Append(proto.Nullable[uint32]{})
		}

		if earliest := resp.GetEarliestAvailableSlot(); earliest != nil {
			b.ResponseEarliestAvailableSlot.Append(proto.NewNullable[uint32](uint32(earliest.GetValue())))
		} else {
			b.ResponseEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
		}
	} else {
		b.ResponseForkDigest.Append("")
		b.ResponseFinalizedRoot.Append(proto.Nullable[[]byte]{})
		b.ResponseHeadRoot.Append(proto.Nullable[[]byte]{})
		b.ResponseFinalizedEpoch.Append(proto.Nullable[uint32]{})
		b.ResponseHeadSlot.Append(proto.Nullable[uint32]{})
		b.ResponseEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
	}

	// Compute peer_id_unique_key: prefer client metadata peer ID, fall back to payload.
	peerID := peerIDFromMetadata(event, func(c *xatuProto.ClientMeta) peerIDMetadataProvider {
		return c.GetLibp2PTraceHandleStatus()
	})

	if peerID == "" {
		peerID = wrappedStringValue(payload.GetPeerId())
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
