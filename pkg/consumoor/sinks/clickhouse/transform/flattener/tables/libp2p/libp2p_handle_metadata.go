package libp2p

import (
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pHandleMetadataTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_METADATA},
		func() flattener.ColumnarBatch {
			return newlibp2pHandleMetadataBatch()
		},
	))
}

func (b *libp2pHandleMetadataBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceHandleMetadata()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceHandleMetadata payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceHandleMetadata()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))

	// Error is nullable.
	if errStr := payload.GetError().GetValue(); errStr != "" {
		b.Error.Append(proto.NewNullable(errStr))
	} else {
		b.Error.Append(proto.Nullable[string]{})
	}

	b.Protocol.Append(payload.GetProtocolId().GetValue())

	// Direction is nullable.
	if dir := payload.GetDirection().GetValue(); dir != "" {
		b.Direction.Append(proto.NewNullable(dir))
	} else {
		b.Direction.Append(proto.Nullable[string]{})
	}

	md := payload.GetMetadata()
	b.Attnets.Append(md.GetAttnets().GetValue())
	b.SeqNumber.Append(md.GetSeqNumber().GetValue())
	b.Syncnets.Append(md.GetSyncnets().GetValue())

	// CustodyGroupCount is nullable uint8.
	if cgc := md.GetCustodyGroupCount(); cgc != nil {
		b.CustodyGroupCount.Append(proto.NewNullable(uint8(cgc.GetValue()))) //nolint:gosec // G115: custody group count fits uint8.
	} else {
		b.CustodyGroupCount.Append(proto.Nullable[uint8]{})
	}

	// LatencyMilliseconds: convert float seconds to scaled Decimal(10,3) milliseconds.
	latencyMs := float64(payload.GetLatency().GetValue()) * 1000.0
	b.LatencyMilliseconds.Append(proto.Decimal64(int64(math.Round(latencyMs * 1000))))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pHandleMetadataBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceHandleMetadata()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceHandleMetadata additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
