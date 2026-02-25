package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconValidatorsTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconValidatorsBatch()
		},
	))
}

func (b *canonicalBeaconValidatorsBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1Validators()
	if payload == nil {
		return fmt.Errorf("nil EthV1Validators payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1Validators()

	for _, validator := range payload.GetValidators() {
		if validator.GetIndex() == nil || validator.GetData() == nil ||
			validator.GetData().GetSlashed() == nil {
			continue
		}

		b.UpdatedDateTime.Append(time.Now())
		b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
		b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
		b.Index.Append(uint32(validator.GetIndex().GetValue())) //nolint:gosec // G115: validator index fits uint32.
		b.Status.Append(validator.GetStatus().GetValue())
		b.Slashed.Append(validator.GetData().GetSlashed().GetValue())

		if validator.GetBalance() != nil {
			b.Balance.Append(proto.NewNullable[uint64](validator.GetBalance().GetValue()))
		} else {
			b.Balance.Append(proto.Nullable[uint64]{})
		}

		if validator.GetData().GetEffectiveBalance() != nil {
			b.EffectiveBalance.Append(proto.NewNullable[uint64](validator.GetData().GetEffectiveBalance().GetValue()))
		} else {
			b.EffectiveBalance.Append(proto.Nullable[uint64]{})
		}

		if validator.GetData().GetActivationEpoch() != nil {
			b.ActivationEpoch.Append(proto.NewNullable[uint64](validator.GetData().GetActivationEpoch().GetValue()))
		} else {
			b.ActivationEpoch.Append(proto.Nullable[uint64]{})
		}

		if validator.GetData().GetActivationEligibilityEpoch() != nil {
			b.ActivationEligibilityEpoch.Append(proto.NewNullable[uint64](validator.GetData().GetActivationEligibilityEpoch().GetValue()))
		} else {
			b.ActivationEligibilityEpoch.Append(proto.Nullable[uint64]{})
		}

		if validator.GetData().GetExitEpoch() != nil {
			b.ExitEpoch.Append(proto.NewNullable[uint64](validator.GetData().GetExitEpoch().GetValue()))
		} else {
			b.ExitEpoch.Append(proto.Nullable[uint64]{})
		}

		if validator.GetData().GetWithdrawableEpoch() != nil {
			b.WithdrawableEpoch.Append(proto.NewNullable[uint64](validator.GetData().GetWithdrawableEpoch().GetValue()))
		} else {
			b.WithdrawableEpoch.Append(proto.Nullable[uint64]{})
		}

		b.appendMetadata(meta)
		b.rows++
	}

	return nil
}

func (b *canonicalBeaconValidatorsBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV1Validators()
	if addl == nil {
		return fmt.Errorf("nil EthV1Validators additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
