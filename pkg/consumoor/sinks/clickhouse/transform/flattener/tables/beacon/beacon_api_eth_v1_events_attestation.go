package beacon

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsAttestationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsAttestationTableName,
		beaconApiEthV1EventsAttestationEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsAttestationBatch() },
	))
}

func (b *beaconApiEthV1EventsAttestationBatch) FlattenTo(
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
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsAttestationBatch) appendRuntime(event *xatu.DecoratedEvent) {
	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsAttestationBatch) appendPayload(event *xatu.DecoratedEvent) {
	attestationV2 := event.GetEthV1EventsAttestationV2()
	if attestationV2 == nil {
		b.AggregationBits.Append("")
		b.Slot.Append(0)
		b.CommitteeIndex.Append("")
		b.BeaconBlockRoot.Append(nil)
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)

		return
	}

	b.AggregationBits.Append(attestationV2.GetAggregationBits())

	data := attestationV2.GetData()
	if data == nil {
		b.Slot.Append(0)
		b.CommitteeIndex.Append("")
		b.BeaconBlockRoot.Append(nil)
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)

		return
	}

	if slot := data.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	if committeeIndex := data.GetIndex(); committeeIndex != nil {
		b.CommitteeIndex.Append(fmt.Sprint(committeeIndex.GetValue()))
	} else {
		b.CommitteeIndex.Append("")
	}

	b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

	if source := data.GetSource(); source != nil {
		if sourceEpoch := source.GetEpoch(); sourceEpoch != nil {
			b.SourceEpoch.Append(uint32(sourceEpoch.GetValue())) //nolint:gosec // epoch fits uint32
		} else {
			b.SourceEpoch.Append(0)
		}

		b.SourceRoot.Append([]byte(source.GetRoot()))
	} else {
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
	}

	if target := data.GetTarget(); target != nil {
		if targetEpoch := target.GetEpoch(); targetEpoch != nil {
			b.TargetEpoch.Append(uint32(targetEpoch.GetValue())) //nolint:gosec // epoch fits uint32
		} else {
			b.TargetEpoch.Append(0)
		}

		b.TargetRoot.Append([]byte(target.GetRoot()))
	} else {
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)
	}
}

func (b *beaconApiEthV1EventsAttestationBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.PropagationSlotStartDiff.Append(0)
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
		b.AttestingValidatorCommitteeIndex.Append("")
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})

		return
	}

	client := event.GetMeta().GetClient()
	additionalV2 := client.GetEthV1EventsAttestationV2()
	additional := extractBeaconSlotEpochPropagation(additionalV2)

	b.SlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.PropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.Epoch.Append(uint32(additional.Epoch))                                       //nolint:gosec // epoch fits uint32
	b.EpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))

	if additionalV2 == nil {
		b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
		b.AttestingValidatorCommitteeIndex.Append("")
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})

		return
	}

	if av := additionalV2.GetAttestingValidator(); av != nil {
		if idx := av.GetIndex(); idx != nil {
			b.AttestingValidatorIndex.Append(proto.NewNullable[uint32](uint32(idx.GetValue()))) //nolint:gosec // validator index fits uint32
		} else {
			b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
		}

		if ci := av.GetCommitteeIndex(); ci != nil {
			b.AttestingValidatorCommitteeIndex.Append(fmt.Sprint(ci.GetValue()))
		} else {
			b.AttestingValidatorCommitteeIndex.Append("")
		}
	} else {
		b.AttestingValidatorIndex.Append(proto.Nullable[uint32]{})
		b.AttestingValidatorCommitteeIndex.Append("")
	}

	if source := additionalV2.GetSource(); source != nil {
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

	if target := additionalV2.GetTarget(); target != nil {
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
}
