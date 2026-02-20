package canonical

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconCommitteeEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
}

var canonicalBeaconCommitteePredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	stateID := event.GetMeta().GetClient().GetEthV1BeaconCommittee().GetStateId()

	return strings.EqualFold(stateID, "finalized")
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconCommitteeTableName,
		canonicalBeaconCommitteeEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconCommitteeBatch() },
		flattener.WithStaticRoutePredicate(canonicalBeaconCommitteePredicate),
	))
}

func (b *canonicalBeaconCommitteeBatch) FlattenTo(
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

func (b *canonicalBeaconCommitteeBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconCommitteeBatch) appendPayload(event *xatu.DecoratedEvent) {
	committee := event.GetEthV1BeaconCommittee()
	if committee == nil {
		b.CommitteeIndex.Append("")
		b.Validators.Append([]uint32{})

		return
	}

	if index := committee.GetIndex(); index != nil {
		b.CommitteeIndex.Append(fmt.Sprintf("%d", index.GetValue()))
	} else {
		b.CommitteeIndex.Append("")
	}

	b.Validators.Append(wrappedUint64SliceToUint32(committee.GetValidators()))
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconCommitteeBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconCommittee()
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
