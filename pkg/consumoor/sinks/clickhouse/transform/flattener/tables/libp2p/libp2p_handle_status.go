package libp2p

import (
	"fmt"
	"math"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	libp2ppb "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pHandleStatusTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_STATUS},
		func() flattener.ColumnarBatch {
			return newlibp2pHandleStatusBatch()
		},
	))
}

func (b *libp2pHandleStatusBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceHandleStatus()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceHandleStatus payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceHandleStatus()

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

	// Request status fields.
	b.appendStatus(payload.GetRequest(), true)

	// Response status fields.
	b.appendStatus(payload.GetResponse(), false)

	// LatencyMilliseconds: convert float seconds to scaled Decimal(10,3) milliseconds.
	latencyMs := float64(payload.GetLatency().GetValue()) * 1000.0
	b.LatencyMilliseconds.Append(proto.Decimal64(int64(math.Round(latencyMs * 1000))))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pHandleStatusBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceHandleStatus()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceHandleStatus additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pHandleStatusBatch) appendStatus(status *libp2ppb.Status, isRequest bool) {
	if isRequest {
		if status == nil {
			b.RequestFinalizedEpoch.Append(proto.Nullable[uint32]{})
			b.RequestFinalizedRoot.Append(proto.Nullable[string]{})
			b.RequestForkDigest.Append("")
			b.RequestHeadRoot.Append(proto.Nullable[[]byte]{})
			b.RequestHeadSlot.Append(proto.Nullable[uint32]{})
			b.RequestEarliestAvailableSlot.Append(proto.Nullable[uint32]{})

			return
		}

		if fe := status.GetFinalizedEpoch(); fe != nil {
			b.RequestFinalizedEpoch.Append(proto.NewNullable(uint32(fe.GetValue()))) //nolint:gosec // G115: finalized epoch fits uint32.
		} else {
			b.RequestFinalizedEpoch.Append(proto.Nullable[uint32]{})
		}

		if fr := status.GetFinalizedRoot().GetValue(); fr != "" {
			b.RequestFinalizedRoot.Append(proto.NewNullable(fr))
		} else {
			b.RequestFinalizedRoot.Append(proto.Nullable[string]{})
		}

		b.RequestForkDigest.Append(status.GetForkDigest().GetValue())

		if hr := status.GetHeadRoot().GetValue(); hr != "" {
			b.RequestHeadRoot.Append(proto.NewNullable([]byte(hr)))
		} else {
			b.RequestHeadRoot.Append(proto.Nullable[[]byte]{})
		}

		if hs := status.GetHeadSlot(); hs != nil {
			b.RequestHeadSlot.Append(proto.NewNullable(uint32(hs.GetValue()))) //nolint:gosec // G115: head slot fits uint32.
		} else {
			b.RequestHeadSlot.Append(proto.Nullable[uint32]{})
		}

		if eas := status.GetEarliestAvailableSlot(); eas != nil {
			b.RequestEarliestAvailableSlot.Append(proto.NewNullable(uint32(eas.GetValue()))) //nolint:gosec // G115: earliest available slot fits uint32.
		} else {
			b.RequestEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
		}
	} else {
		if status == nil {
			b.ResponseFinalizedEpoch.Append(proto.Nullable[uint32]{})
			b.ResponseFinalizedRoot.Append(proto.Nullable[[]byte]{})
			b.ResponseForkDigest.Append("")
			b.ResponseHeadRoot.Append(proto.Nullable[[]byte]{})
			b.ResponseHeadSlot.Append(proto.Nullable[uint32]{})
			b.ResponseEarliestAvailableSlot.Append(proto.Nullable[uint32]{})

			return
		}

		if fe := status.GetFinalizedEpoch(); fe != nil {
			b.ResponseFinalizedEpoch.Append(proto.NewNullable(uint32(fe.GetValue()))) //nolint:gosec // G115: finalized epoch fits uint32.
		} else {
			b.ResponseFinalizedEpoch.Append(proto.Nullable[uint32]{})
		}

		if fr := status.GetFinalizedRoot().GetValue(); fr != "" {
			b.ResponseFinalizedRoot.Append(proto.NewNullable([]byte(fr)))
		} else {
			b.ResponseFinalizedRoot.Append(proto.Nullable[[]byte]{})
		}

		b.ResponseForkDigest.Append(status.GetForkDigest().GetValue())

		if hr := status.GetHeadRoot().GetValue(); hr != "" {
			b.ResponseHeadRoot.Append(proto.NewNullable([]byte(hr)))
		} else {
			b.ResponseHeadRoot.Append(proto.Nullable[[]byte]{})
		}

		if hs := status.GetHeadSlot(); hs != nil {
			b.ResponseHeadSlot.Append(proto.NewNullable(uint32(hs.GetValue()))) //nolint:gosec // G115: head slot fits uint32.
		} else {
			b.ResponseHeadSlot.Append(proto.Nullable[uint32]{})
		}

		if eas := status.GetEarliestAvailableSlot(); eas != nil {
			b.ResponseEarliestAvailableSlot.Append(proto.NewNullable(uint32(eas.GetValue()))) //nolint:gosec // G115: earliest available slot fits uint32.
		} else {
			b.ResponseEarliestAvailableSlot.Append(proto.Nullable[uint32]{})
		}
	}
}
