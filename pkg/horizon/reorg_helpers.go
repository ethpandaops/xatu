package horizon

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/horizon/subscription"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var blockBasedHorizonTypes = []xatu.HorizonType{
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
	xatu.HorizonType_HORIZON_TYPE_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
}

func reorgSlotRange(event subscription.ReorgEvent) (uint64, uint64) {
	end := uint64(event.Slot)
	start := end

	if event.Depth > 0 {
		depth := uint64(event.Depth)
		if depth > end+1 {
			start = 0
		} else {
			start = end - (depth - 1)
		}
	}

	return start, end
}

func rollbackSlot(start uint64) uint64 {
	if start == 0 {
		return 0
	}

	return start - 1
}

func (h *Horizon) rollbackReorgLocations(ctx context.Context, start uint64) {
	if h.coordinatorClient == nil {
		return
	}

	rollback := rollbackSlot(start)

	for _, horizonType := range blockBasedHorizonTypes {
		location, err := h.coordinatorClient.GetHorizonLocation(ctx, horizonType, h.networkID())
		if err != nil {
			h.log.WithError(err).WithField("horizon_type", horizonType.String()).
				Debug("Failed to fetch horizon location for reorg rollback")
			continue
		}

		if location == nil {
			continue
		}

		updated := false

		if location.HeadSlot > rollback {
			location.HeadSlot = rollback
			updated = true
		}

		if location.FillSlot > rollback {
			location.FillSlot = rollback
			updated = true
		}

		if !updated {
			continue
		}

		if err := h.coordinatorClient.UpsertHorizonLocation(ctx, location); err != nil {
			h.log.WithError(err).WithField("horizon_type", horizonType.String()).
				Warn("Failed to rollback horizon location after reorg")
		} else {
			h.log.WithFields(logrus.Fields{
				"horizon_type": horizonType.String(),
				"head_slot":    location.HeadSlot,
				"fill_slot":    location.FillSlot,
			}).Info("Rolled back horizon location after reorg")
		}
	}
}

func (h *Horizon) networkID() string {
	if h.beaconPool == nil || h.beaconPool.Metadata() == nil {
		return ""
	}

	return fmt.Sprintf("%d", h.beaconPool.Metadata().Network.ID)
}
