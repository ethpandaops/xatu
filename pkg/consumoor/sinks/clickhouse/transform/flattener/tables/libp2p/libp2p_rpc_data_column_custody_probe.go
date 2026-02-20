package libp2p

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pRpcDataColumnCustodyProbeEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		libp2pRpcDataColumnCustodyProbeTableName,
		libp2pRpcDataColumnCustodyProbeEventNames,
		func() flattener.ColumnarBatch { return newlibp2pRpcDataColumnCustodyProbeBatch() },
	))
}

func (b *libp2pRpcDataColumnCustodyProbeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
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

func (b *libp2pRpcDataColumnCustodyProbeBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

// slotEpochFields holds resolved slot/epoch timing values for the custody probe event.
type slotEpochFields struct {
	slot, epoch, wallSlot, wallEpoch                             uint32
	slotStartDT, epochStartDT, wallSlotStartDT, wallEpochStartDT int64
}

// resolveSlotEpochFields extracts slot/epoch fields from the payload,
// then overrides them with client additional data when available.
//
//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func resolveSlotEpochFields(
	payload *libp2p.DataColumnCustodyProbe,
	event *xatu.DecoratedEvent,
) slotEpochFields {
	var f slotEpochFields

	if s := payload.GetSlot(); s != nil {
		f.slot = s.GetValue()
	}

	if dt := payload.GetSlotStartDateTime(); dt != nil {
		f.slotStartDT = int64(dt.GetValue())
	}

	if e := payload.GetEpoch(); e != nil {
		f.epoch = e.GetValue()
	}

	if dt := payload.GetEpochStartDateTime(); dt != nil {
		f.epochStartDT = int64(dt.GetValue())
	}

	if ws := payload.GetWallclockRequestSlot(); ws != nil {
		f.wallSlot = ws.GetValue()
	}

	if dt := payload.GetWallclockRequestSlotStartDateTime(); dt != nil {
		f.wallSlotStartDT = int64(dt.GetValue())
	}

	if we := payload.GetWallclockRequestEpoch(); we != nil {
		f.wallEpoch = we.GetValue()
	}

	if dt := payload.GetWallclockRequestEpochStartDateTime(); dt != nil {
		f.wallEpochStartDT = int64(dt.GetValue())
	}

	// Override from client additional data if available.
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return f
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceRpcDataColumnCustodyProbe()
	if additional == nil {
		return f
	}

	if s := additional.GetSlot(); s != nil {
		if num := s.GetNumber(); num != nil {
			f.slot = uint32(num.GetValue())
		}

		if dt := s.GetStartDateTime(); dt != nil {
			f.slotStartDT = dt.AsTime().Unix()
		}
	}

	if e := additional.GetEpoch(); e != nil {
		if num := e.GetNumber(); num != nil {
			f.epoch = uint32(num.GetValue())
		}

		if dt := e.GetStartDateTime(); dt != nil {
			f.epochStartDT = dt.AsTime().Unix()
		}
	}

	if ws := additional.GetWallclockSlot(); ws != nil {
		if num := ws.GetNumber(); num != nil {
			f.wallSlot = uint32(num.GetValue())
		}

		if dt := ws.GetStartDateTime(); dt != nil {
			f.wallSlotStartDT = dt.AsTime().Unix()
		}
	}

	if we := additional.GetWallclockEpoch(); we != nil {
		if num := we.GetNumber(); num != nil {
			f.wallEpoch = uint32(num.GetValue())
		}

		if dt := we.GetStartDateTime(); dt != nil {
			f.wallEpochStartDT = dt.AsTime().Unix()
		}
	}

	return f
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pRpcDataColumnCustodyProbeBatch) appendPayload(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	payload := event.GetLibp2PTraceRpcDataColumnCustodyProbe()
	if payload == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockRequestSlot.Append(0)
		b.WallclockRequestSlotStartDateTime.Append(time.Time{})
		b.WallclockRequestEpoch.Append(0)
		b.WallclockRequestEpochStartDateTime.Append(time.Time{})
		b.ColumnIndex.Append(0)
		b.ColumnRowsCount.Append(0)
		b.BeaconBlockRoot.Append(nil)
		b.PeerIDUniqueKey.Append(0)
		b.Result.Append("")
		b.ResponseTimeMs.Append(0)
		b.Error.Append(proto.Nullable[string]{})

		return
	}

	f := resolveSlotEpochFields(payload, event)

	b.Slot.Append(f.slot)
	b.SlotStartDateTime.Append(time.Unix(f.slotStartDT, 0))
	b.Epoch.Append(f.epoch)
	b.EpochStartDateTime.Append(time.Unix(f.epochStartDT, 0))
	b.WallclockRequestSlot.Append(f.wallSlot)
	b.WallclockRequestSlotStartDateTime.Append(time.Unix(f.wallSlotStartDT, 0))
	b.WallclockRequestEpoch.Append(f.wallEpoch)
	b.WallclockRequestEpochStartDateTime.Append(time.Unix(f.wallEpochStartDT, 0))

	if colIdx := payload.GetColumnIndex(); colIdx != nil {
		b.ColumnIndex.Append(uint64(colIdx.GetValue()))
	} else {
		b.ColumnIndex.Append(0)
	}

	if colRows := payload.GetColumnRowsCount(); colRows != nil {
		b.ColumnRowsCount.Append(uint16(colRows.GetValue()))
	} else {
		b.ColumnRowsCount.Append(0)
	}

	if root := wrappedStringValue(payload.GetBeaconBlockRoot()); root != "" {
		b.BeaconBlockRoot.Append([]byte(root))
	} else {
		b.BeaconBlockRoot.Append(nil)
	}

	b.Result.Append(wrappedStringValue(payload.GetResult()))

	if responseTime := payload.GetResponseTimeMs(); responseTime != nil {
		b.ResponseTimeMs.Append(int32(responseTime.GetValue()))
	} else {
		b.ResponseTimeMs.Append(0)
	}

	// Error (nullable string).
	if errVal := wrappedStringValue(payload.GetError()); errVal != "" {
		b.Error.Append(proto.NewNullable[string](errVal))
	} else {
		b.Error.Append(proto.Nullable[string]{})
	}

	// Compute peer_id_unique_key, preferring client metadata over payload.
	peerID := ""

	if event.GetMeta() != nil && event.GetMeta().GetClient() != nil {
		if probeMeta := event.GetMeta().GetClient().GetLibp2PTraceRpcDataColumnCustodyProbe(); probeMeta != nil {
			if traceMeta := probeMeta.GetMetadata(); traceMeta != nil && traceMeta.GetPeerId() != nil {
				peerID = traceMeta.GetPeerId().GetValue()
			}
		}
	}

	if peerID == "" {
		peerID = wrappedStringValue(payload.GetPeerId())
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
