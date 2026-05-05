package canonical

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
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
	r, err := route.NewStaticRoute(
		canonicalBeaconCommitteeTableName,
		canonicalBeaconCommitteeEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconCommitteeBatch() },
		route.WithStaticRoutePredicate(canonicalBeaconCommitteePredicate),
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconCommitteeBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconCommittee() == nil {
		return fmt.Errorf("nil eth_v1_beacon_committee payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
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

	if slot := committee.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // G115
	} else {
		b.Slot.Append(0)
	}

	if index := committee.GetIndex(); index != nil {
		b.CommitteeIndex.Append(strconv.FormatUint(index.GetValue(), 10))
	} else {
		b.CommitteeIndex.Append("")
	}

	b.Validators.Append(wrappedUint64SliceToUint32(committee.GetValidators()))
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconCommitteeBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconCommittee()
	if additional == nil {
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

		return
	}

	if slotData := additional.GetSlot(); slotData != nil {
		if startDateTime := slotData.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
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
}
