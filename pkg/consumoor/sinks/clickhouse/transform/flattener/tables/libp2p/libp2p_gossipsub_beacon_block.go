package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubBeaconBlockEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pGossipsubBeaconBlockTableName,
		libp2pGossipsubBeaconBlockEventNames,
		func() flattener.ColumnarBatch { return newlibp2pGossipsubBeaconBlockBatch() },
	))
}

func (b *libp2pGossipsubBeaconBlockBatch) FlattenTo(
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

func (b *libp2pGossipsubBeaconBlockBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubBeaconBlockBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubBeaconBlock()
	if payload == nil {
		b.Block.Append(nil)
		b.ProposerIndex.Append(0)

		return
	}

	b.Block.Append([]byte(wrappedStringValue(payload.GetBlock())))

	if proposerIndex := payload.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
	} else {
		b.ProposerIndex.Append(0)
	}
}

func (b *libp2pGossipsubBeaconBlockBatch) appendClientAdditionalData(
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

	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconBlock()
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

// gossipsubSlotEpochProvider is implemented by gossipsub client additional data
// types that contain slot/epoch/wallclock/propagation fields.
type gossipsubSlotEpochProvider interface {
	GetSlot() *xatu.SlotV2
	GetEpoch() *xatu.EpochV2
	GetWallclockSlot() *xatu.SlotV2
	GetWallclockEpoch() *xatu.EpochV2
	GetPropagation() *xatu.PropagationV2
}

// gossipsubSlotEpochResult holds the extracted slot/epoch/wallclock/propagation
// fields for gossipsub tables.
type gossipsubSlotEpochResult struct {
	Slot                        uint32
	SlotStartDateTime           int64
	Epoch                       uint32
	EpochStartDateTime          int64
	WallclockSlot               uint32
	WallclockSlotStartDateTime  int64
	WallclockEpoch              uint32
	WallclockEpochStartDateTime int64
	PropagationSlotStartDiff    uint32
}

// setGossipsubSlotEpochFields extracts slot/epoch/wallclock/propagation fields
// from a gossipsub additional data provider and applies them via a callback.
//
//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func setGossipsubSlotEpochFields(
	additional gossipsubSlotEpochProvider,
	apply func(gossipsubSlotEpochResult),
) {
	if additional == nil {
		return
	}

	result := gossipsubSlotEpochResult{}

	if slot := additional.GetSlot(); slot != nil {
		if num := slot.GetNumber(); num != nil {
			result.Slot = uint32(num.GetValue())
		}

		if startDT := slot.GetStartDateTime(); startDT != nil {
			result.SlotStartDateTime = startDT.AsTime().Unix()
		}
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if num := epoch.GetNumber(); num != nil {
			result.Epoch = uint32(num.GetValue())
		}

		if startDT := epoch.GetStartDateTime(); startDT != nil {
			result.EpochStartDateTime = startDT.AsTime().Unix()
		}
	}

	if wallSlot := additional.GetWallclockSlot(); wallSlot != nil {
		if num := wallSlot.GetNumber(); num != nil {
			result.WallclockSlot = uint32(num.GetValue())
		}

		if startDT := wallSlot.GetStartDateTime(); startDT != nil {
			result.WallclockSlotStartDateTime = startDT.AsTime().Unix()
		}
	}

	if wallEpoch := additional.GetWallclockEpoch(); wallEpoch != nil {
		if num := wallEpoch.GetNumber(); num != nil {
			result.WallclockEpoch = uint32(num.GetValue())
		}

		if startDT := wallEpoch.GetStartDateTime(); startDT != nil {
			result.WallclockEpochStartDateTime = startDT.AsTime().Unix()
		}
	}

	if propagation := additional.GetPropagation(); propagation != nil {
		if diff := propagation.GetSlotStartDiff(); diff != nil {
			result.PropagationSlotStartDiff = uint32(diff.GetValue())
		}
	}

	apply(result)
}
