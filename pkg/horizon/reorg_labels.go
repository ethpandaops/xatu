package horizon

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const reorgDetectedLabelKey = "reorg_detected"

func (h *Horizon) markReorgMetadata(events []*xatu.DecoratedEvent) {
	if h.reorgTracker == nil {
		return
	}

	for _, event := range events {
		slot, ok := slotFromDecoratedEvent(event)
		if !ok {
			continue
		}

		if !h.reorgTracker.IsReorgSlot(slot) {
			continue
		}

		if event.Meta == nil || event.Meta.Client == nil {
			continue
		}

		if event.Meta.Client.Labels == nil {
			event.Meta.Client.Labels = make(map[string]string)
		}

		event.Meta.Client.Labels[reorgDetectedLabelKey] = "true"
	}
}

func slotFromDecoratedEvent(event *xatu.DecoratedEvent) (uint64, bool) {
	if event == nil || event.Meta == nil || event.Meta.Client == nil {
		return 0, false
	}

	switch data := event.Meta.Client.AdditionalData.(type) {
	case *xatu.ClientMeta_EthV2BeaconBlockV2:
		return slotFromSlotV2(data.EthV2BeaconBlockV2.GetSlot())
	case *xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation:
		return slotFromSlotV2(data.EthV2BeaconBlockElaboratedAttestation.GetSlot())
	case *xatu.ClientMeta_EthV2BeaconBlockDeposit:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockDeposit.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockWithdrawal:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockWithdrawal.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockVoluntaryExit.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockProposerSlashing:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockProposerSlashing.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockAttesterSlashing.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockBlsToExecutionChange.GetBlock())
	case *xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction:
		return slotFromBlockIdentifier(data.EthV2BeaconBlockExecutionTransaction.GetBlock())
	default:
		return 0, false
	}
}

func slotFromBlockIdentifier(block *xatu.BlockIdentifier) (uint64, bool) {
	if block == nil {
		return 0, false
	}

	return slotFromSlotV2(block.GetSlot())
}

func slotFromSlotV2(slot *xatu.SlotV2) (uint64, bool) {
	if slot == nil || slot.Number == nil {
		return 0, false
	}

	return slot.Number.Value, true
}

// slotFromIdentifierString parses a slot identifier if it is numeric.
