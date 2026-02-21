package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1ValidatorAttestationDataEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1ValidatorAttestationDataTableName,
		beaconApiEthV1ValidatorAttestationDataEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1ValidatorAttestationDataBatch() },
	))
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) FlattenTo(
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

func (b *beaconApiEthV1ValidatorAttestationDataBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) appendPayload(event *xatu.DecoratedEvent) {
	attestationData := event.GetEthV1ValidatorAttestationData()
	if attestationData == nil {
		b.Slot.Append(0)
		b.CommitteeIndex.Append("")
		b.BeaconBlockRoot.Append(nil)
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)

		return
	}

	if slot := attestationData.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	if committeeIndex := attestationData.GetIndex(); committeeIndex != nil {
		b.CommitteeIndex.Append(fmt.Sprint(committeeIndex.GetValue()))
	} else {
		b.CommitteeIndex.Append("")
	}

	b.BeaconBlockRoot.Append([]byte(attestationData.GetBeaconBlockRoot()))

	if source := attestationData.GetSource(); source != nil {
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

	if target := attestationData.GetTarget(); target != nil {
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

func (b *beaconApiEthV1ValidatorAttestationDataBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})
		b.RequestDateTime.Append(time.Time{})
		b.RequestDuration.Append(0)
		b.RequestSlotStartDiff.Append(0)

		return
	}

	additional := event.GetMeta().GetClient().GetEthV1ValidatorAttestationData()
	if additional == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})
		b.RequestDateTime.Append(time.Time{})
		b.RequestDuration.Append(0)
		b.RequestSlotStartDiff.Append(0)

		return
	}

	if epochNumber := additional.GetEpoch().GetNumber(); epochNumber != nil {
		b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.Epoch.Append(0)
	}

	if slot := additional.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}

	if source := additional.GetSource(); source != nil {
		if sourceEpoch := source.GetEpoch(); sourceEpoch != nil {
			if startDateTime := sourceEpoch.GetStartDateTime(); startDateTime != nil {
				b.SourceEpochStartDateTime.Append(startDateTime.AsTime())
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
		if targetEpoch := target.GetEpoch(); targetEpoch != nil {
			if startDateTime := targetEpoch.GetStartDateTime(); startDateTime != nil {
				b.TargetEpochStartDateTime.Append(startDateTime.AsTime())
			} else {
				b.TargetEpochStartDateTime.Append(time.Time{})
			}
		} else {
			b.TargetEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.TargetEpochStartDateTime.Append(time.Time{})
	}

	snapshot := additional.GetSnapshot()
	if snapshot == nil {
		b.RequestDateTime.Append(time.Time{})
		b.RequestDuration.Append(0)
		b.RequestSlotStartDiff.Append(0)

		return
	}

	if timestamp := snapshot.GetTimestamp(); timestamp != nil {
		b.RequestDateTime.Append(timestamp.AsTime())
	} else {
		b.RequestDateTime.Append(time.Time{})
	}

	if requestDuration := snapshot.GetRequestDurationMs(); requestDuration != nil {
		b.RequestDuration.Append(uint32(requestDuration.GetValue())) //nolint:gosec // duration ms fits uint32
	} else {
		b.RequestDuration.Append(0)
	}

	if requestSlotStartDiff := snapshot.GetRequestedAtSlotStartDiffMs(); requestSlotStartDiff != nil {
		b.RequestSlotStartDiff.Append(uint32(requestSlotStartDiff.GetValue())) //nolint:gosec // diff ms fits uint32
	} else {
		b.RequestSlotStartDiff.Append(0)
	}
}
