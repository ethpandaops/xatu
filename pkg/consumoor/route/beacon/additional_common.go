package beacon

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

type beaconSlotEpochPropagationV2 interface {
	GetSlot() *xatu.SlotV2
	GetPropagation() *xatu.PropagationV2
	GetEpoch() *xatu.EpochV2
}

type beaconSlotEpochPropagationAdditional struct {
	SlotStartDateTime        int64
	PropagationSlotStartDiff uint64
	Epoch                    uint64
	EpochStartDateTime       int64
}

func extractBeaconSlotEpochPropagation(
	additionalV2 beaconSlotEpochPropagationV2,
) beaconSlotEpochPropagationAdditional {
	out := beaconSlotEpochPropagationAdditional{}
	if additionalV2 == nil {
		return out
	}

	if slot := additionalV2.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			out.SlotStartDateTime = startDateTime.AsTime().Unix()
		}
	}

	if propagation := additionalV2.GetPropagation(); propagation != nil {
		if slotStartDiff := propagation.GetSlotStartDiff(); slotStartDiff != nil {
			out.PropagationSlotStartDiff = slotStartDiff.GetValue()
		}
	}

	if epoch := additionalV2.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			out.Epoch = epochNumber.GetValue()
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			out.EpochStartDateTime = startDateTime.AsTime().Unix()
		}
	}

	return out
}
