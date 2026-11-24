package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pDataColumnCompletedEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_DATA_COLUMN_COMPLETED,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pDataColumnCompletedTableName,
		libp2pDataColumnCompletedEventNames,
		func() route.ColumnarBatch { return newlibp2pDataColumnCompletedBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pDataColumnCompletedBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceDataColumnCompleted() == nil {
		return fmt.Errorf("nil libp2p_trace_data_column_completed payload: %w", route.ErrInvalidEvent)
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

func (b *libp2pDataColumnCompletedBatch) validate(event *xatu.DecoratedEvent) error {
	if event.GetLibp2PTraceDataColumnCompleted().GetColumnIndex() == nil {
		return fmt.Errorf("nil ColumnIndex: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceDataColumnCompleted()
	if additional == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockSlot() == nil {
		return fmt.Errorf("nil WallclockSlot: %w", route.ErrInvalidEvent)
	}

	if additional.GetWallclockEpoch() == nil {
		return fmt.Errorf("nil WallclockEpoch: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *libp2pDataColumnCompletedBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *libp2pDataColumnCompletedBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceDataColumnCompleted()

	b.BeaconBlockRoot.Append([]byte(wrappedStringValue(payload.GetBlockRoot())))
	b.ColumnIndex.Append(payload.GetColumnIndex().GetValue())
	b.CompletedVia.Append(wrappedStringValue(payload.GetCompletedVia()))
	b.CellsTotal.Append(payload.GetCellsTotal().GetValue())
	b.CellsFromGossip.Append(payload.GetCellsFromGossip().GetValue())
	b.CellsFromEl.Append(payload.GetCellsFromEl().GetValue())
	b.ContributingPeers.Append(payload.GetContributingPeers().GetValue())
	b.KzgCommitmentsCount.Append(payload.GetKzgCommitmentsCount().GetValue())
}

func (b *libp2pDataColumnCompletedBatch) appendClientAdditionalData(
	event *xatu.DecoratedEvent,
) {
	additional := event.GetMeta().GetClient().GetLibp2PTraceDataColumnCompleted()

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

	topic := parseTopicFields(wrappedStringValue(additional.GetTopic()))
	b.TopicLayer.Append(topic.Layer)
	b.TopicForkDigestValue.Append(topic.ForkDigestValue)
	b.TopicName.Append(topic.Name)
	b.TopicEncoding.Append(topic.Encoding)

	// A column completed from the EL or by reconstruction has no completing peer.
	peerID := additional.GetMetadata().GetPeerId().GetValue()
	if peerID == "" {
		b.PeerIDUniqueKey.Append(0)

		return
	}

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
