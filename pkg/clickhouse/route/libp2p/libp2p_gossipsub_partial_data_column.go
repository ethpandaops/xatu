package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubPartialDataColumnEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubPartialDataColumnTableName,
		libp2pGossipsubPartialDataColumnEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubPartialDataColumnBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubPartialDataColumnBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubPartialDataColumn() == nil {
		return fmt.Errorf("nil libp2p_trace_gossipsub_partial_data_column payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendClientAdditionalData(event)
	b.rows++

	return nil
}

func (b *libp2pGossipsubPartialDataColumnBatch) validate(event *xatu.DecoratedEvent) error {
	if event.GetLibp2PTraceGossipsubPartialDataColumn().GetColumnIndex() == nil {
		return fmt.Errorf("nil ColumnIndex: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumn()
	if additional == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if additional.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockSlot() == nil {
		return fmt.Errorf("nil WallclockSlot: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockEpoch() == nil {
		return fmt.Errorf("nil WallclockEpoch: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pGossipsubPartialDataColumnBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}

	b.UniqueKey.Append(route.SeaHashInt64(event.GetEvent().GetId()))
}

//nolint:gosec // G115: proto values are bounded by ClickHouse column schema
func (b *libp2pGossipsubPartialDataColumnBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubPartialDataColumn()

	b.Direction.Append(wrappedStringValue(payload.GetDirection()))
	b.CapturePoint.Append(wrappedStringValue(payload.GetCapturePoint()))
	b.GroupID.Append(wrappedStringValue(payload.GetGroupId()))
	b.GroupVersion.Append(uint8(payload.GetGroupVersion().GetValue()))
	b.BeaconBlockRoot.Append([]byte(wrappedStringValue(payload.GetBlockRoot())))
	b.ColumnIndex.Append(payload.GetColumnIndex().GetValue())
	b.HasMessage.Append(payload.GetHasMessage().GetValue())
	b.CellsCount.Append(payload.GetCellsCount().GetValue())
	b.CellsBitmap.Append(wrappedStringValue(payload.GetCellsBitmap()))
	b.BitmapLength.Append(payload.GetBitmapLength().GetValue())
	b.HasHeader.Append(payload.GetHasHeader().GetValue())
	b.KzgCommitmentsCount.Append(payload.GetKzgCommitmentsCount().GetValue())
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue()))
	b.HasMetadata.Append(payload.GetHasMetadata().GetValue())
	b.MetadataSize.Append(payload.GetMetadataSize().GetValue())
	b.MetadataAvailableCount.Append(payload.GetMetadataAvailableCount().GetValue())
	b.MetadataRequestsCount.Append(payload.GetMetadataRequestsCount().GetValue())
	b.MetadataAvailable.Append(wrappedStringValue(payload.GetMetadataAvailable()))
	b.MetadataRequests.Append(wrappedStringValue(payload.GetMetadataRequests()))
}

func (b *libp2pGossipsubPartialDataColumnBatch) appendClientAdditionalData(
	event *xatu.DecoratedEvent,
) {
	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumn()

	setGossipsubSlotEpochFields(additional, func(f gossipsubSlotEpochResult) {
		b.Slot.Append(f.Slot)
		b.SlotStartDateTime.Append(time.Unix(f.SlotStartDateTime, 0))
		b.Epoch.Append(f.Epoch)
		b.EpochStartDateTime.Append(time.Unix(f.EpochStartDateTime, 0))
		b.WallclockSlot.Append(f.WallclockSlot)
		b.WallclockSlotStartDateTime.Append(time.Unix(f.WallclockSlotStartDateTime, 0))
		b.WallclockEpoch.Append(f.WallclockEpoch)
		b.WallclockEpochStartDateTime.Append(time.Unix(f.WallclockEpochStartDateTime, 0))
		b.PropagationSlotStartDiff.Append(f.PropagationSlotStartDiff)
	})

	b.MessageSize.Append(additional.GetMessageSize().GetValue())

	topic := parseTopicFields(wrappedStringValue(additional.GetTopic()))
	b.TopicLayer.Append(topic.Layer)
	b.TopicForkDigestValue.Append(topic.ForkDigestValue)
	b.TopicName.Append(topic.Name)
	b.TopicEncoding.Append(topic.Encoding)

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(additional.GetMetadata().GetPeerId().GetValue(), networkName))
}
