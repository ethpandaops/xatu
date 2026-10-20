package libp2p

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubPartialDataColumnOutcomeEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_PARTIAL_DATA_COLUMN_OUTCOME,
}

func init() {
	r, err := route.NewStaticRoute(
		libp2pGossipsubPartialDataColumnOutcomeTableName,
		libp2pGossipsubPartialDataColumnOutcomeEventNames,
		func() route.ColumnarBatch { return newlibp2pGossipsubPartialDataColumnOutcomeBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *libp2pGossipsubPartialDataColumnOutcomeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetLibp2PTraceGossipsubPartialDataColumnOutcome() == nil {
		return fmt.Errorf("nil libp2p_trace_gossipsub_partial_data_column_outcome payload: %w", route.ErrInvalidEvent)
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

func (b *libp2pGossipsubPartialDataColumnOutcomeBatch) validate(event *xatu.DecoratedEvent) error {
	if event.GetLibp2PTraceGossipsubPartialDataColumnOutcome().GetColumnIndex() == nil {
		return fmt.Errorf("nil ColumnIndex: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumnOutcome()
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

func (b *libp2pGossipsubPartialDataColumnOutcomeBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}

	b.UniqueKey.Append(route.SeaHashInt64(event.GetEvent().GetId()))
}

func (b *libp2pGossipsubPartialDataColumnOutcomeBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubPartialDataColumnOutcome()

	b.GroupID.Append(wrappedStringValue(payload.GetGroupId()))
	b.BeaconBlockRoot.Append([]byte(wrappedStringValue(payload.GetBlockRoot())))
	b.ColumnIndex.Append(payload.GetColumnIndex().GetValue())
	b.Stage.Append(wrappedStringValue(payload.GetStage()))
	b.Outcome.Append(wrappedStringValue(payload.GetOutcome()))
	b.Reason.Append(wrappedStringValue(payload.GetReason()))
	b.CellsCount.Append(payload.GetCellsCount().GetValue())
	b.NewCellsCount.Append(payload.GetNewCellsCount().GetValue())
}

func (b *libp2pGossipsubPartialDataColumnOutcomeBatch) appendClientAdditionalData(
	event *xatu.DecoratedEvent,
) {
	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubPartialDataColumnOutcome()

	setGossipsubSlotEpochFields(partialOutcomeSlotEpoch{additional}, func(f gossipsubSlotEpochResult) {
		b.Slot.Append(f.Slot)
		b.SlotStartDateTime.Append(time.Unix(f.SlotStartDateTime, 0))
		b.Epoch.Append(f.Epoch)
		b.EpochStartDateTime.Append(time.Unix(f.EpochStartDateTime, 0))
		b.WallclockSlot.Append(f.WallclockSlot)
		b.WallclockSlotStartDateTime.Append(time.Unix(f.WallclockSlotStartDateTime, 0))
		b.WallclockEpoch.Append(f.WallclockEpoch)
		b.WallclockEpochStartDateTime.Append(time.Unix(f.WallclockEpochStartDateTime, 0))
	})

	topic := parseTopicFields(wrappedStringValue(additional.GetTopic()))
	b.TopicLayer.Append(topic.Layer)
	b.TopicForkDigestValue.Append(topic.ForkDigestValue)
	b.TopicName.Append(topic.Name)
	b.TopicEncoding.Append(topic.Encoding)

	networkName := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(additional.GetMetadata().GetPeerId().GetValue(), networkName))
}

// partialOutcomeSlotEpoch adapts outcome additional data, which has no propagation, to gossipsubSlotEpochProvider.
type partialOutcomeSlotEpoch struct {
	*xatu.ClientMeta_AdditionalLibP2PTraceGossipSubPartialDataColumnOutcomeData
}

func (partialOutcomeSlotEpoch) GetPropagation() *xatu.PropagationV2 {
	return nil
}
