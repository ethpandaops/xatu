package libp2p

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pGossipsubBeaconAttestationTableName,
		[]xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION},
		func() flattener.ColumnarBatch {
			return newlibp2pGossipsubBeaconAttestationBatch()
		},
	))
}

func (b *libp2pGossipsubBeaconAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetLibp2PTraceGossipsubBeaconAttestation()
	if payload == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubBeaconAttestation payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconAttestation()

	peerID := addl.GetMetadata().GetPeerId().GetValue()

	b.UpdatedDateTime.Append(time.Now())
	b.Version.Append(gossipsubSchemaVersion)
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())

	// Slot/epoch from addl data.
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	// CommitteeIndex from payload (AttestationData.Index stored as string).
	b.CommitteeIndex.Append(strconv.FormatUint(payload.GetData().GetIndex(), 10))

	// AttestingValidator is nullable.
	if av := addl.GetAttestingValidator(); av != nil && av.GetIndex() != nil {
		b.AttestingValidatorIndex.Append(proto.NewNullable(uint32(av.GetIndex().GetValue()))) //nolint:gosec // G115: validator index fits uint32.
	} else {
		b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
	}

	// AttestingValidatorCommitteeIndex from addl attesting validator.
	if av := addl.GetAttestingValidator(); av != nil && av.GetCommitteeIndex() != nil {
		b.AttestingValidatorCommitteeIndex.Append(strconv.FormatUint(av.GetCommitteeIndex().GetValue(), 10))
	} else {
		b.AttestingValidatorCommitteeIndex.Append("")
	}

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

	b.AggregationBits.Append(payload.GetAggregationBits())
	b.BeaconBlockRoot.Append([]byte(payload.GetData().GetBeaconBlockRoot()))

	// Source checkpoint.
	b.SourceEpoch.Append(uint32(addl.GetSource().GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: source epoch fits uint32.
	b.SourceEpochStartDateTime.Append(addl.GetSource().GetEpoch().GetStartDateTime().AsTime())
	b.SourceRoot.Append([]byte(payload.GetData().GetSource().GetRoot()))

	// Target checkpoint.
	b.TargetEpoch.Append(uint32(addl.GetTarget().GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: target epoch fits uint32.
	b.TargetEpochStartDateTime.Append(addl.GetTarget().GetEpoch().GetStartDateTime().AsTime())
	b.TargetRoot.Append([]byte(payload.GetData().GetTarget().GetRoot()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *libp2pGossipsubBeaconAttestationBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconAttestation()
	if addl == nil {
		return fmt.Errorf("nil LibP2PTraceGossipsubBeaconAttestation additional data: %w", flattener.ErrInvalidEvent)
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

	if addl.GetSource() == nil || addl.GetSource().GetEpoch() == nil || addl.GetSource().GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil SourceEpoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetTarget() == nil || addl.GetTarget().GetEpoch() == nil || addl.GetTarget().GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil TargetEpoch: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
