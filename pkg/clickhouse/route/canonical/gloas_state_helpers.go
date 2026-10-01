package canonical

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func validateEpochData(epoch *xatu.EpochV2) error {
	if epoch == nil || epoch.GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}

func validateSlotData(slot *xatu.SlotV2) error {
	if slot == nil || slot.GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	return nil
}

func timeOrZero(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}

	return ts.AsTime()
}
