package beacon

import (
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1ProposerDutyEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
}

var beaconApiEthV1ProposerDutyPredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	stateID := event.GetMeta().GetClient().GetEthV1ProposerDuty().GetStateId()

	return strings.EqualFold(stateID, "head")
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1ProposerDutyTableName,
		beaconApiEthV1ProposerDutyEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1ProposerDutyBatch() },
		flattener.WithStaticRoutePredicate(beaconApiEthV1ProposerDutyPredicate),
	))
}

func (b *beaconApiEthV1ProposerDutyBatch) FlattenTo(
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

func (b *beaconApiEthV1ProposerDutyBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1ProposerDutyBatch) appendPayload(event *xatu.DecoratedEvent) {
	duty := event.GetEthV1ProposerDuty()
	if duty == nil {
		b.Slot.Append(0)
		b.ProposerValidatorIndex.Append(0)
		b.ProposerPubkey.Append("")

		return
	}

	if slot := duty.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	if validatorIndex := duty.GetValidatorIndex(); validatorIndex != nil {
		b.ProposerValidatorIndex.Append(uint32(validatorIndex.GetValue())) //nolint:gosec // validator index fits uint32
	} else {
		b.ProposerValidatorIndex.Append(0)
	}

	b.ProposerPubkey.Append(duty.GetPubkey())
}

func (b *beaconApiEthV1ProposerDutyBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additional := event.GetMeta().GetClient().GetEthV1ProposerDuty()
	if additional == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})

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
}
