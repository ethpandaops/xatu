package beacon

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1BeaconCommitteeEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
}

var beaconApiEthV1BeaconCommitteePredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	stateID := event.GetMeta().GetClient().GetEthV1BeaconCommittee().GetStateId()

	return !strings.EqualFold(stateID, "finalized")
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1BeaconCommitteeTableName,
		beaconApiEthV1BeaconCommitteeEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1BeaconCommitteeBatch() },
		flattener.WithStaticRoutePredicate(beaconApiEthV1BeaconCommitteePredicate),
	))
}

func (b *beaconApiEthV1BeaconCommitteeBatch) FlattenTo(
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

func (b *beaconApiEthV1BeaconCommitteeBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1BeaconCommitteeBatch) appendPayload(event *xatu.DecoratedEvent) {
	committee := event.GetEthV1BeaconCommittee()
	if committee == nil {
		b.Slot.Append(0)
		b.CommitteeIndex.Append("")
		b.Validators.Append(nil)

		return
	}

	if slot := committee.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.Slot.Append(0)
	}

	if committeeIndex := committee.GetIndex(); committeeIndex != nil {
		b.CommitteeIndex.Append(fmt.Sprint(committeeIndex.GetValue()))
	} else {
		b.CommitteeIndex.Append("")
	}

	validators := committee.GetValidators()
	if len(validators) > 0 {
		result := make([]uint32, 0, len(validators))

		for _, v := range validators {
			if v != nil {
				result = append(result, uint32(v.GetValue())) //nolint:gosec // validator index fits uint32
			}
		}

		b.Validators.Append(result)
	} else {
		b.Validators.Append(nil)
	}
}

func (b *beaconApiEthV1BeaconCommitteeBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.Epoch.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	additional := event.GetMeta().GetClient().GetEthV1BeaconCommittee()
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
