package libp2p

import (
	"strconv"
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var libp2pGossipsubBeaconAttestationEventNames = []xatu.Event_Name{
	xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		libp2pGossipsubBeaconAttestationTableName,
		libp2pGossipsubBeaconAttestationEventNames,
		func() flattener.ColumnarBatch { return newlibp2pGossipsubBeaconAttestationBatch() },
	))
}

func (b *libp2pGossipsubBeaconAttestationBatch) FlattenTo(
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

func (b *libp2pGossipsubBeaconAttestationBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubBeaconAttestationBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetLibp2PTraceGossipsubBeaconAttestation()
	if payload == nil {
		b.AggregationBits.Append("")
		b.CommitteeIndex.Append("")
		b.BeaconBlockRoot.Append(nil)
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)

		return
	}

	b.AggregationBits.Append(payload.GetAggregationBits())

	if data := payload.GetData(); data != nil {
		if idx := data.GetIndex(); idx != 0 {
			b.CommitteeIndex.Append(strconv.FormatUint(idx, 10))
		} else {
			b.CommitteeIndex.Append("")
		}

		b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

		if source := data.GetSource(); source != nil {
			b.SourceEpoch.Append(uint32(source.GetEpoch()))
			b.SourceRoot.Append([]byte(source.GetRoot()))
		} else {
			b.SourceEpoch.Append(0)
			b.SourceRoot.Append(nil)
		}

		if target := data.GetTarget(); target != nil {
			b.TargetEpoch.Append(uint32(target.GetEpoch()))
			b.TargetRoot.Append([]byte(target.GetRoot()))
		} else {
			b.TargetEpoch.Append(0)
			b.TargetRoot.Append(nil)
		}
	} else {
		b.CommitteeIndex.Append("")
		b.BeaconBlockRoot.Append(nil)
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *libp2pGossipsubBeaconAttestationBatch) appendClientAdditionalData(
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
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})
		b.AttestingValidatorCommitteeIndex.Append("")
		b.AttestingValidatorIndex.Append(chProto.Nullable[uint32]{})
		b.MessageID.Append("")
		b.MessageSize.Append(0)
		b.TopicLayer.Append("")
		b.TopicForkDigestValue.Append("")
		b.TopicName.Append("")
		b.TopicEncoding.Append("")
		b.PeerIDUniqueKey.Append(0)

		return
	}

	additional := event.GetMeta().GetClient().GetLibp2PTraceGossipsubBeaconAttestation()
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
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})
		b.AttestingValidatorCommitteeIndex.Append("")
		b.AttestingValidatorIndex.Append(chProto.Nullable[uint32]{})
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

	// Extract source/target epoch datetime from additional data.
	if source := additional.GetSource(); source != nil {
		if epoch := source.GetEpoch(); epoch != nil {
			if startDT := epoch.GetStartDateTime(); startDT != nil {
				b.SourceEpochStartDateTime.Append(startDT.AsTime())
			} else {
				b.SourceEpochStartDateTime.Append(time.Time{})
			}
		} else {
			b.SourceEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.SourceEpochStartDateTime.Append(time.Time{})
	}

	if target := additional.GetTarget(); target != nil {
		if epoch := target.GetEpoch(); epoch != nil {
			if startDT := epoch.GetStartDateTime(); startDT != nil {
				b.TargetEpochStartDateTime.Append(startDT.AsTime())
			} else {
				b.TargetEpochStartDateTime.Append(time.Time{})
			}
		} else {
			b.TargetEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.TargetEpochStartDateTime.Append(time.Time{})
	}

	// Attesting validator fields.
	if av := additional.GetAttestingValidator(); av != nil {
		if idx := av.GetCommitteeIndex(); idx != nil {
			b.AttestingValidatorCommitteeIndex.Append(strconv.FormatUint(idx.GetValue(), 10))
		} else {
			b.AttestingValidatorCommitteeIndex.Append("")
		}

		if valIdx := av.GetIndex(); valIdx != nil {
			b.AttestingValidatorIndex.Append(chProto.NewNullable[uint32](uint32(valIdx.GetValue())))
		} else {
			b.AttestingValidatorIndex.Append(chProto.Nullable[uint32]{})
		}
	} else {
		b.AttestingValidatorCommitteeIndex.Append("")
		b.AttestingValidatorIndex.Append(chProto.Nullable[uint32]{})
	}

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
