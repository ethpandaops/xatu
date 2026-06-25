package beacon

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1ValidatorAttestationDataEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
}

func init() {
	r, err := route.NewStaticRoute(
		beaconApiEthV1ValidatorAttestationDataTableName,
		beaconApiEthV1ValidatorAttestationDataEventNames,
		func() route.ColumnarBatch { return newbeaconApiEthV1ValidatorAttestationDataBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) FlattenTo(
	event *xatu.DecoratedEvent,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1ValidatorAttestationData() == nil {
		return fmt.Errorf("nil eth_v1_validator_attestation_data payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) validate(
	event *xatu.DecoratedEvent,
) error {
	payload := event.GetEthV1ValidatorAttestationData()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil CommitteeIndex: %w", route.ErrInvalidEvent)
	}

	source := payload.GetSource()
	if source == nil || source.GetEpoch() == nil {
		return fmt.Errorf("nil SourceEpoch: %w", route.ErrInvalidEvent)
	}

	target := payload.GetTarget()
	if target == nil || target.GetEpoch() == nil {
		return fmt.Errorf("nil TargetEpoch: %w", route.ErrInvalidEvent)
	}

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
	b.Slot.Append(uint32(attestationData.GetSlot().GetValue())) //nolint:gosec // slot fits uint32
	b.CommitteeIndex.Append(strconv.FormatUint(attestationData.GetIndex().GetValue(), 10))
	b.BeaconBlockRoot.Append([]byte(attestationData.GetBeaconBlockRoot()))

	source := attestationData.GetSource()
	b.SourceEpoch.Append(uint32(source.GetEpoch().GetValue())) //nolint:gosec // epoch fits uint32
	b.SourceRoot.Append([]byte(source.GetRoot()))

	target := attestationData.GetTarget()
	b.TargetEpoch.Append(uint32(target.GetEpoch().GetValue())) //nolint:gosec // epoch fits uint32
	b.TargetRoot.Append([]byte(target.GetRoot()))
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
