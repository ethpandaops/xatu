package libp2p

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pRpcDataColumnCustodyProbeTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE},
		func() flattener.ColumnarBatch {
			return newlibp2pRpcDataColumnCustodyProbeBatch()
		},
	))
}

func (b *libp2pRpcDataColumnCustodyProbeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceRpcDataColumnCustodyProbe()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceRpcDataColumnCustodyProbe payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcDataColumnCustodyProbe()

	peerID := payload.GetPeerId().GetValue()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())

	// Slot/epoch from addl data (enriched with start datetimes).
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	// Wallclock from addl data.
	b.WallclockRequestSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockRequestSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockRequestEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockRequestEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())

	b.ColumnIndex.Append(uint64(payload.GetColumnIndex().GetValue()))
	b.ColumnRowsCount.Append(uint16(payload.GetColumnRowsCount().GetValue())) //nolint:gosec // G115: rows count fits uint16.
	b.BeaconBlockRoot.Append([]byte(payload.GetBeaconBlockRoot().GetValue()))
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))
	b.Result.Append(payload.GetResult().GetValue())
	b.ResponseTimeMs.Append(int32(payload.GetResponseTimeMs().GetValue())) //nolint:gosec // G115: response time fits int32.

	// Error is nullable.
	if errVal := payload.GetError().GetValue(); errVal != "" {
		b.Error.Append(proto.NewNullable(errVal))
	} else {
		b.Error.Append(proto.Nullable[string]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pRpcDataColumnCustodyProbeBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceRpcDataColumnCustodyProbe()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceRpcDataColumnCustodyProbe additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
