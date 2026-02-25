package libp2p

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pGossipsubAggregateAndProofTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF},
		func() flattener.ColumnarBatch {
			return newlibp2pGossipsubAggregateAndProofBatch()
		},
	))
}

func (b *libp2pGossipsubAggregateAndProofBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceGossipsubAggregateAndProof()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubAggregateAndProof payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubAggregateAndProof()
	msg := payload.GetMessage()
	agg := msg.GetAggregate()
	data := agg.GetData()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UpdatedDateTime.Append(time.Now())
	b.Version.Append(gossipsubSchemaVersion)
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())

	// Slot/epoch from addl data.
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	// Wallclock from addl data.
	b.WallclockSlot.Append(uint32(addl.GetWallclockSlot().GetNumber().GetValue())) //nolint:gosec // G115: wallclock slot fits uint32.
	b.WallclockSlotStartDateTime.Append(addl.GetWallclockSlot().GetStartDateTime().AsTime())
	b.WallclockEpoch.Append(uint32(addl.GetWallclockEpoch().GetNumber().GetValue())) //nolint:gosec // G115: wallclock epoch fits uint32.
	b.WallclockEpochStartDateTime.Append(addl.GetWallclockEpoch().GetStartDateTime().AsTime())

	b.PropagationSlotStartDiff.Append(uint32(addl.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.PeerIDUniqueKey.Append(flattener.SeaHashInt64(peerID + meta.MetaNetworkName))
	b.MessageID.Append(addl.GetMessageId().GetValue())
	b.MessageSize.Append(addl.GetMessageSize().GetValue())

	topic := addl.GetTopic().GetValue()
	layer, forkDigest, name, encoding := parseTopic(topic)
	b.TopicLayer.Append(layer)
	b.TopicForkDigestValue.Append(forkDigest)
	b.TopicName.Append(name)
	b.TopicEncoding.Append(encoding)

	b.AggregatorIndex.Append(uint32(msg.GetAggregatorIndex().GetValue())) //nolint:gosec // G115: aggregator index fits uint32.
	b.CommitteeIndex.Append(strconv.FormatUint(data.GetIndex().GetValue(), 10))
	b.AggregationBits.Append(agg.GetAggregationBits())
	b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

	// Source checkpoint.
	b.SourceEpoch.Append(uint32(data.GetSource().GetEpoch().GetValue())) //nolint:gosec // G115: source epoch fits uint32.
	b.SourceRoot.Append([]byte(data.GetSource().GetRoot()))

	// Target checkpoint.
	b.TargetEpoch.Append(uint32(data.GetTarget().GetEpoch().GetValue())) //nolint:gosec // G115: target epoch fits uint32.
	b.TargetRoot.Append([]byte(data.GetTarget().GetRoot()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pGossipsubAggregateAndProofBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubAggregateAndProof()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubAggregateAndProof additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetMetadata() == nil || addl.GetMetadata().GetPeerId() == nil {
		return fmt.Errorf("nil PeerId in metadata: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetWallclockSlot() == nil || addl.GetWallclockSlot().GetNumber() == nil {
		return fmt.Errorf("nil WallclockSlot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetWallclockEpoch() == nil || addl.GetWallclockEpoch().GetNumber() == nil {
		return fmt.Errorf("nil WallclockEpoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetPropagation() == nil || addl.GetPropagation().GetSlotStartDiff() == nil {
		return fmt.Errorf("nil PropagationSlotStartDiff: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
