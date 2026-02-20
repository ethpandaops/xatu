package canonical

import (
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconValidatorsEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconValidatorsTableName,
		canonicalBeaconValidatorsEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconValidatorsBatch() },
	))
}

func (b *canonicalBeaconValidatorsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil || event.GetEthV1Validators() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	// Extract epoch info from additional data.
	var (
		epoch          uint32
		epochStartTime time.Time
	)

	if client := event.GetMeta().GetClient(); client != nil {
		if extra := client.GetEthV1Validators(); extra != nil {
			if epochData := extra.GetEpoch(); epochData != nil {
				if number := epochData.GetNumber(); number != nil {
					epoch = uint32(number.GetValue()) //nolint:gosec // bounded by uint32 column
				}

				if start := epochData.GetStartDateTime(); start != nil {
					epochStartTime = start.AsTime()
				}
			}
		}
	}

	now := time.Now()

	for _, validator := range event.GetEthV1Validators().GetValidators() {
		if validator == nil {
			continue
		}

		b.UpdatedDateTime.Append(now)
		b.Epoch.Append(epoch)
		b.EpochStartDateTime.Append(epochStartTime)

		if idx := validator.GetIndex(); idx != nil {
			b.Index.Append(uint32(idx.GetValue())) //nolint:gosec // bounded by uint32 column
		} else {
			b.Index.Append(0)
		}

		if balance := validator.GetBalance(); balance != nil && balance.GetValue() != 0 {
			b.Balance.Append(chProto.NewNullable[uint64](balance.GetValue()))
		} else {
			b.Balance.Append(chProto.Nullable[uint64]{})
		}

		if status := validator.GetStatus(); status != nil {
			b.Status.Append(status.GetValue())
		} else {
			b.Status.Append("")
		}

		data := validator.GetData()
		if data != nil {
			if data.GetEffectiveBalance() != nil && data.GetEffectiveBalance().GetValue() != 0 {
				b.EffectiveBalance.Append(chProto.NewNullable[uint64](data.GetEffectiveBalance().GetValue()))
			} else {
				b.EffectiveBalance.Append(chProto.Nullable[uint64]{})
			}

			if data.GetSlashed() != nil {
				b.Slashed.Append(data.GetSlashed().GetValue())
			} else {
				b.Slashed.Append(false)
			}

			if val, ok := setOptionalEpoch(data.GetActivationEpoch()); ok {
				b.ActivationEpoch.Append(chProto.NewNullable[uint64](val))
			} else {
				b.ActivationEpoch.Append(chProto.Nullable[uint64]{})
			}

			if val, ok := setOptionalEpoch(data.GetActivationEligibilityEpoch()); ok {
				b.ActivationEligibilityEpoch.Append(chProto.NewNullable[uint64](val))
			} else {
				b.ActivationEligibilityEpoch.Append(chProto.Nullable[uint64]{})
			}

			if val, ok := setOptionalEpoch(data.GetExitEpoch()); ok {
				b.ExitEpoch.Append(chProto.NewNullable[uint64](val))
			} else {
				b.ExitEpoch.Append(chProto.Nullable[uint64]{})
			}

			if val, ok := setOptionalEpoch(data.GetWithdrawableEpoch()); ok {
				b.WithdrawableEpoch.Append(chProto.NewNullable[uint64](val))
			} else {
				b.WithdrawableEpoch.Append(chProto.Nullable[uint64]{})
			}
		} else {
			b.EffectiveBalance.Append(chProto.Nullable[uint64]{})
			b.Slashed.Append(false)
			b.ActivationEpoch.Append(chProto.Nullable[uint64]{})
			b.ActivationEligibilityEpoch.Append(chProto.Nullable[uint64]{})
			b.ExitEpoch.Append(chProto.Nullable[uint64]{})
			b.WithdrawableEpoch.Append(chProto.Nullable[uint64]{})
		}

		b.appendMetadata(meta)
		b.rows++
	}

	return nil
}
