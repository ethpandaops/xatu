package canonical

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// appendBlockIdentifier fills the common block-identifier columns (slot, epoch, block version,
// block root) from a *xatu.BlockIdentifier into the provided column pointers.
// Every pointer except blockVersion and blockRoot may be nil to skip that field.
//
//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func appendBlockIdentifier(
	block *xatu.BlockIdentifier,
	slot *proto.ColUInt32,
	slotStartDateTime *proto.ColDateTime,
	epoch *proto.ColUInt32,
	epochStartDateTime *proto.ColDateTime,
	blockVersion *proto.ColStr,
	blockRoot *flattener.SafeColFixedStr,
) {
	if block == nil {
		if slot != nil {
			slot.Append(0)
		}

		if slotStartDateTime != nil {
			slotStartDateTime.Append(time.Time{})
		}

		if epoch != nil {
			epoch.Append(0)
		}

		if epochStartDateTime != nil {
			epochStartDateTime.Append(time.Time{})
		}

		if blockVersion != nil {
			blockVersion.Append("")
		}

		if blockRoot != nil {
			blockRoot.Append(nil)
		}

		return
	}

	if blockVersion != nil {
		blockVersion.Append(block.GetVersion())
	}

	if blockRoot != nil {
		blockRoot.Append([]byte(block.GetRoot()))
	}

	if blockEpoch := block.GetEpoch(); blockEpoch != nil {
		if epoch != nil {
			if epochNumber := blockEpoch.GetNumber(); epochNumber != nil {
				epoch.Append(uint32(epochNumber.GetValue()))
			} else {
				epoch.Append(0)
			}
		}

		if epochStartDateTime != nil {
			if startDateTime := blockEpoch.GetStartDateTime(); startDateTime != nil {
				epochStartDateTime.Append(startDateTime.AsTime())
			} else {
				epochStartDateTime.Append(time.Time{})
			}
		}
	} else {
		if epoch != nil {
			epoch.Append(0)
		}

		if epochStartDateTime != nil {
			epochStartDateTime.Append(time.Time{})
		}
	}

	if blockSlot := block.GetSlot(); blockSlot != nil {
		if slot != nil {
			if slotNumber := blockSlot.GetNumber(); slotNumber != nil {
				slot.Append(uint32(slotNumber.GetValue()))
			} else {
				slot.Append(0)
			}
		}

		if slotStartDateTime != nil {
			if startDateTime := blockSlot.GetStartDateTime(); startDateTime != nil {
				slotStartDateTime.Append(startDateTime.AsTime())
			} else {
				slotStartDateTime.Append(time.Time{})
			}
		}
	} else {
		if slot != nil {
			slot.Append(0)
		}

		if slotStartDateTime != nil {
			slotStartDateTime.Append(time.Time{})
		}
	}
}
