package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubDataColumnSidecarEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		libp2pGossipsubDataColumnSidecarTableName,
		libp2pGossipsubDataColumnSidecarEventNames,
		func() flattener.ColumnarBatch { return newlibp2pGossipsubDataColumnSidecarBatch() },
	))
}

func (b *libp2pGossipsubDataColumnSidecarBatch) FlattenTo(
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
	b.appendPayload(event)
	b.appendClientAdditionalData(event, meta)
	b.rows++

	return nil
}

func (b *libp2pGossipsubDataColumnSidecarBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubDataColumnSidecarBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubDataColumnSidecar()
	if payload == nil {
		b.ColumnIndex.Append(0)
		b.ProposerIndex.Append(0)
		b.StateRoot.Append(nil)
		b.ParentRoot.Append(nil)
		b.BeaconBlockRoot.Append(nil)
		b.KzgCommitmentsCount.Append(0)

		return
	}

	if idx := payload.GetIndex(); idx != nil {
		b.ColumnIndex.Append(idx.GetValue())
	} else {
		b.ColumnIndex.Append(0)
	}

	if proposerIndex := payload.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
	} else {
		b.ProposerIndex.Append(0)
	}

	b.StateRoot.Append([]byte(wrappedStringValue(payload.GetStateRoot())))
	b.ParentRoot.Append([]byte(wrappedStringValue(payload.GetParentRoot())))
	b.BeaconBlockRoot.Append([]byte(wrappedStringValue(payload.GetBlockRoot())))

	if kzg := payload.GetKzgCommitmentsCount(); kzg != nil {
		b.KzgCommitmentsCount.Append(kzg.GetValue())
	} else {
		b.KzgCommitmentsCount.Append(0)
	}
}

func (b *libp2pGossipsubDataColumnSidecarBatch) appendClientAdditionalData(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Version.Append(4294967295)
		b.MessageID.Append("")
		b.MessageSize.Append(0)
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubDataColumnSidecar()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.WallclockSlot.Append(0)
		b.WallclockSlotStartDateTime.Append(time.Time{})
		b.WallclockEpoch.Append(0)
		b.WallclockEpochStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Version.Append(4294967295)
		b.MessageID.Append("")
		b.MessageSize.Append(0)
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

	// Extract slot/epoch/wallclock/propagation fields.
	var propagationSlotStartDiff uint32

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
		propagationSlotStartDiff = f.PropagationSlotStartDiff
	})

	// Compute version for ReplacingMergeTree dedup.
	b.Version.Append(4294967295 - propagationSlotStartDiff)

	// Extract message fields.
	b.MessageID.Append(wrappedStringValue(additional.GetMessageId()))

	if msgSize := additional.GetMessageSize(); msgSize != nil {
		b.MessageSize.Append(msgSize.GetValue())
	} else {
		b.MessageSize.Append(0)
	}

	// Parse topic fields.
	if topic := wrappedStringValue(additional.GetTopic()); topic != "" {
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

	// Extract peer ID from metadata.
	peerID := ""
	if traceMeta := additional.GetMetadata(); traceMeta != nil && traceMeta.GetPeerId() != nil {
		peerID = traceMeta.GetPeerId().GetValue()
	}

	networkName := meta.MetaNetworkName
	b.PeerIDUniqueKey.Append(computePeerIDUniqueKey(peerID, networkName))
}
