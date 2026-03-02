package canonical

import (
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconElaboratedAttestationEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconElaboratedAttestationTableName,
		canonicalBeaconElaboratedAttestationEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconElaboratedAttestationBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconElaboratedAttestationBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendRow(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconElaboratedAttestationBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

// appendRow appends all payload and additional data fields in one pass to avoid double-appending
// columns that exist in both payload and additional data (e.g. Slot).
//
//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconElaboratedAttestationBatch) appendRow(event *xatu.DecoratedEvent) {
	attestation := event.GetEthV2BeaconBlockElaboratedAttestation()

	// Payload fields.
	if attestation != nil {
		b.Validators.Append(wrappedUint64SliceToUint32(attestation.GetValidatorIndexes()))

		if data := attestation.GetData(); data != nil {
			b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

			if index := data.GetIndex(); index != nil {
				b.CommitteeIndex.Append(strconv.FormatUint(index.GetValue(), 10))
			} else {
				b.CommitteeIndex.Append("")
			}

			if source := data.GetSource(); source != nil {
				if epoch := source.GetEpoch(); epoch != nil {
					b.SourceEpoch.Append(uint32(epoch.GetValue()))
				} else {
					b.SourceEpoch.Append(0)
				}

				b.SourceRoot.Append([]byte(source.GetRoot()))
			} else {
				b.SourceEpoch.Append(0)
				b.SourceRoot.Append(nil)
			}

			if target := data.GetTarget(); target != nil {
				if epoch := target.GetEpoch(); epoch != nil {
					b.TargetEpoch.Append(uint32(epoch.GetValue()))
				} else {
					b.TargetEpoch.Append(0)
				}

				b.TargetRoot.Append([]byte(target.GetRoot()))
			} else {
				b.TargetEpoch.Append(0)
				b.TargetRoot.Append(nil)
			}
		} else {
			b.BeaconBlockRoot.Append(nil)
			b.CommitteeIndex.Append("")
			b.SourceEpoch.Append(0)
			b.SourceRoot.Append(nil)
			b.TargetEpoch.Append(0)
			b.TargetRoot.Append(nil)
		}
	} else {
		b.Validators.Append([]uint32{})
		b.BeaconBlockRoot.Append(nil)
		b.CommitteeIndex.Append("")
		b.SourceEpoch.Append(0)
		b.SourceRoot.Append(nil)
		b.TargetEpoch.Append(0)
		b.TargetRoot.Append(nil)
	}

	// Slot from payload (matching Vector: .slot = .data.data.slot).
	if attestation != nil && attestation.GetData() != nil {
		if slot := attestation.GetData().GetSlot(); slot != nil {
			b.Slot.Append(uint32(slot.GetValue()))
		} else {
			b.Slot.Append(0)
		}
	} else {
		b.Slot.Append(0)
	}

	// Additional data fields.
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockElaboratedAttestation()
	if additional == nil {
		b.BlockSlot.Append(0)
		b.BlockSlotStartDateTime.Append(time.Time{})
		b.BlockEpoch.Append(0)
		b.BlockEpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)
		b.PositionInBlock.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SourceEpochStartDateTime.Append(time.Time{})
		b.TargetEpochStartDateTime.Append(time.Time{})

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.BlockSlot, &b.BlockSlotStartDateTime, &b.BlockEpoch, &b.BlockEpochStartDateTime, nil, &b.BlockRoot)

	if positionInBlock := additional.GetPositionInBlock(); positionInBlock != nil {
		b.PositionInBlock.Append(uint32(positionInBlock.GetValue()))
	} else {
		b.PositionInBlock.Append(0)
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

	if source := additional.GetSource(); source != nil {
		if sourceEpoch := source.GetEpoch(); sourceEpoch != nil {
			if startDateTime := sourceEpoch.GetStartDateTime(); startDateTime != nil {
				b.SourceEpochStartDateTime.Append(startDateTime.AsTime())
			} else {
				b.SourceEpochStartDateTime.Append(time.Time{})
			}
		} else {
			b.SourceEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.SourceEpochStartDateTime.Append(time.Time{})
	}

	if target := additional.GetTarget(); target != nil {
		if targetEpoch := target.GetEpoch(); targetEpoch != nil {
			if startDateTime := targetEpoch.GetStartDateTime(); startDateTime != nil {
				b.TargetEpochStartDateTime.Append(startDateTime.AsTime())
			} else {
				b.TargetEpochStartDateTime.Append(time.Time{})
			}
		} else {
			b.TargetEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.TargetEpochStartDateTime.Append(time.Time{})
	}
}
