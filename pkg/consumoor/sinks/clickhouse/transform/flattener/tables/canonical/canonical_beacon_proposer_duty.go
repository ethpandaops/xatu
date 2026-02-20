package canonical

import (
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconProposerDutyEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
}

var canonicalBeaconProposerDutyPredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	stateID := event.GetMeta().GetClient().GetEthV1ProposerDuty().GetStateId()

	return strings.EqualFold(stateID, "finalized")
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconProposerDutyTableName,
		canonicalBeaconProposerDutyEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconProposerDutyBatch() },
		flattener.WithStaticRoutePredicate(canonicalBeaconProposerDutyPredicate),
	))
}

func (b *canonicalBeaconProposerDutyBatch) FlattenTo(
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

func (b *canonicalBeaconProposerDutyBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconProposerDutyBatch) appendPayload(event *xatu.DecoratedEvent) {
	duty := event.GetEthV1ProposerDuty()
	if duty == nil {
		b.ProposerPubkey.Append("")
		b.ProposerValidatorIndex.Append(0)

		return
	}

	b.ProposerPubkey.Append(duty.GetPubkey())

	if validatorIndex := duty.GetValidatorIndex(); validatorIndex != nil {
		b.ProposerValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ProposerValidatorIndex.Append(0)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconProposerDutyBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1ProposerDuty()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	if epochData := additional.GetEpoch(); epochData != nil {
		if epochNumber := epochData.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue()))
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epochData.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if slotData := additional.GetSlot(); slotData != nil {
		if slotNumber := slotData.GetNumber(); slotNumber != nil {
			b.Slot.Append(uint32(slotNumber.GetValue()))
		} else {
			b.Slot.Append(0)
		}

		if startDateTime := slotData.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
	}
}
