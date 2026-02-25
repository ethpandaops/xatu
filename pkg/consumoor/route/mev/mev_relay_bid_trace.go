package mev

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mevRelayBidTraceEventNames = []xatu.Event_Name{
	xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
}

func init() {
	r, err := route.NewStaticRoute(
		mevRelayBidTraceTableName,
		mevRelayBidTraceEventNames,
		func() route.ColumnarBatch { return newmevRelayBidTraceBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *mevRelayBidTraceBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetMevRelayBidTraceBuilderBlockSubmission() == nil {
		return fmt.Errorf("nil mev_relay_bid_trace_builder_block_submission payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)

	if err := b.appendPayload(event); err != nil {
		return fmt.Errorf("appending payload: %w", err)
	}

	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *mevRelayBidTraceBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayBidTraceBuilderBlockSubmission()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", route.ErrInvalidEvent)
	}

	if payload.GetBlockNumber() == nil {
		return fmt.Errorf("nil BlockNumber: %w", route.ErrInvalidEvent)
	}

	if payload.GetGasLimit() == nil {
		return fmt.Errorf("nil GasLimit: %w", route.ErrInvalidEvent)
	}

	if payload.GetGasUsed() == nil {
		return fmt.Errorf("nil GasUsed: %w", route.ErrInvalidEvent)
	}

	if payload.GetNumTx() == nil {
		return fmt.Errorf("nil NumTx: %w", route.ErrInvalidEvent)
	}

	if payload.GetTimestamp() == nil {
		return fmt.Errorf("nil Timestamp: %w", route.ErrInvalidEvent)
	}

	if payload.GetTimestampMs() == nil {
		return fmt.Errorf("nil TimestampMs: %w", route.ErrInvalidEvent)
	}

	if payload.GetOptimisticSubmission() == nil {
		return fmt.Errorf("nil OptimisticSubmission: %w", route.ErrInvalidEvent)
	}

	if client := event.GetMeta().GetClient(); client != nil {
		if additional := client.GetMevRelayBidTraceBuilderBlockSubmission(); additional != nil {
			if epoch := additional.GetEpoch(); epoch != nil {
				if epoch.GetNumber() == nil {
					return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
				}
			}

			if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
				if wallclockSlot.GetNumber() == nil {
					return fmt.Errorf("nil WallclockSlot: %w", route.ErrInvalidEvent)
				}
			}

			if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
				if wallclockEpoch.GetNumber() == nil {
					return fmt.Errorf("nil WallclockEpoch: %w", route.ErrInvalidEvent)
				}
			}
		}
	}

	return nil
}

func (b *mevRelayBidTraceBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *mevRelayBidTraceBatch) appendPayload(event *xatu.DecoratedEvent) error {
	payload := event.GetMevRelayBidTraceBuilderBlockSubmission()
	if slot := payload.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.Slot.Append(0)
	}

	if parentHash := payload.GetParentHash(); parentHash != nil {
		b.ParentHash.Append([]byte(parentHash.GetValue()))
	} else {
		b.ParentHash.Append(nil)
	}

	if blockNumber := payload.GetBlockNumber(); blockNumber != nil {
		b.BlockNumber.Append(blockNumber.GetValue())
	} else {
		b.BlockNumber.Append(0)
	}

	if blockHash := payload.GetBlockHash(); blockHash != nil {
		b.BlockHash.Append([]byte(blockHash.GetValue()))
	} else {
		b.BlockHash.Append(nil)
	}

	if builderPubkey := payload.GetBuilderPubkey(); builderPubkey != nil {
		b.BuilderPubkey.Append(builderPubkey.GetValue())
	} else {
		b.BuilderPubkey.Append("")
	}

	if proposerPubkey := payload.GetProposerPubkey(); proposerPubkey != nil {
		b.ProposerPubkey.Append(proposerPubkey.GetValue())
	} else {
		b.ProposerPubkey.Append("")
	}

	if proposerFeeRecipient := payload.GetProposerFeeRecipient(); proposerFeeRecipient != nil {
		b.ProposerFeeRecipient.Append([]byte(proposerFeeRecipient.GetValue()))
	} else {
		b.ProposerFeeRecipient.Append(nil)
	}

	if gasLimit := payload.GetGasLimit(); gasLimit != nil {
		b.GasLimit.Append(gasLimit.GetValue())
	} else {
		b.GasLimit.Append(0)
	}

	if gasUsed := payload.GetGasUsed(); gasUsed != nil {
		b.GasUsed.Append(gasUsed.GetValue())
	} else {
		b.GasUsed.Append(0)
	}

	if value := payload.GetValue(); value != nil {
		parsedValue, err := route.ParseUInt256(value.GetValue())
		if err != nil {
			return fmt.Errorf("parsing value: %w", err)
		}

		b.Value.Append(parsedValue)
	} else {
		b.Value.Append(proto.UInt256{})
	}

	if numTx := payload.GetNumTx(); numTx != nil {
		b.NumTx.Append(uint32(numTx.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.NumTx.Append(0)
	}

	if timestamp := payload.GetTimestamp(); timestamp != nil {
		b.Timestamp.Append(timestamp.GetValue())
	} else {
		b.Timestamp.Append(0)
	}

	if timestampMs := payload.GetTimestampMs(); timestampMs != nil {
		b.TimestampMs.Append(timestampMs.GetValue())
	} else {
		b.TimestampMs.Append(0)
	}

	if optimistic := payload.GetOptimisticSubmission(); optimistic != nil {
		b.OptimisticSubmission.Append(optimistic.GetValue())
	} else {
		b.OptimisticSubmission.Append(false)
	}

	return nil
}

func (b *mevRelayBidTraceBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.appendZeroAdditionalData()

		return
	}

	client := event.GetMeta().GetClient()
	additional := client.GetMevRelayBidTraceBuilderBlockSubmission()

	if additional == nil {
		b.appendZeroAdditionalData()

		return
	}

	if relay := additional.GetRelay(); relay != nil {
		if name := relay.GetName(); name != nil {
			b.RelayName.Append(name.GetValue())
		} else {
			b.RelayName.Append("")
		}
	} else {
		b.RelayName.Append("")
	}

	b.appendMevSlotEpoch(additional)

	if requestedAt := additional.GetRequestedAtSlotTime(); requestedAt != nil {
		b.RequestedAtSlotTime.Append(uint32(requestedAt.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.RequestedAtSlotTime.Append(0)
	}

	if responseAt := additional.GetResponseAtSlotTime(); responseAt != nil {
		b.ResponseAtSlotTime.Append(uint32(responseAt.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
	} else {
		b.ResponseAtSlotTime.Append(0)
	}
}

func (b *mevRelayBidTraceBatch) appendMevSlotEpoch(
	additional *xatu.ClientMeta_AdditionalMevRelayBidTraceBuilderBlockSubmissionData,
) {
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
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if wallclockSlot := additional.GetWallclockSlot(); wallclockSlot != nil {
		if slotNumber := wallclockSlot.GetNumber(); slotNumber != nil {
			b.WallclockRequestSlot.Append(uint32(slotNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.WallclockRequestSlot.Append(0)
		}

		if startDateTime := wallclockSlot.GetStartDateTime(); startDateTime != nil {
			b.WallclockRequestSlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockRequestSlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockRequestSlot.Append(0)
		b.WallclockRequestSlotStartDateTime.Append(time.Time{})
	}

	if wallclockEpoch := additional.GetWallclockEpoch(); wallclockEpoch != nil {
		if epochNumber := wallclockEpoch.GetNumber(); epochNumber != nil {
			b.WallclockRequestEpoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // proto uint64 narrowed to uint32 target field
		} else {
			b.WallclockRequestEpoch.Append(0)
		}

		if startDateTime := wallclockEpoch.GetStartDateTime(); startDateTime != nil {
			b.WallclockRequestEpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.WallclockRequestEpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.WallclockRequestEpoch.Append(0)
		b.WallclockRequestEpochStartDateTime.Append(time.Time{})
	}
}

func (b *mevRelayBidTraceBatch) appendZeroAdditionalData() {
	b.RelayName.Append("")
	b.SlotStartDateTime.Append(time.Time{})
	b.Epoch.Append(0)
	b.EpochStartDateTime.Append(time.Time{})
	b.WallclockRequestSlot.Append(0)
	b.WallclockRequestSlotStartDateTime.Append(time.Time{})
	b.WallclockRequestEpoch.Append(0)
	b.WallclockRequestEpochStartDateTime.Append(time.Time{})
	b.RequestedAtSlotTime.Append(0)
	b.ResponseAtSlotTime.Append(0)
}
