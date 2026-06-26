package canonical

import (
	"fmt"
	"time"

	chProto "github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconValidatorsEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconValidatorsTableName,
		canonicalBeaconValidatorsEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconValidatorsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconValidatorsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1Validators() == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
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
		// Canonical validator rows are gap-sensitive: halt rather than skip or
		// zero-fill a validator whose required fields are absent.
		if validator == nil {
			return fmt.Errorf("nil validator entry: %w", route.ErrInvalidEvent)
		}

		if validator.GetIndex() == nil {
			return fmt.Errorf("nil validator Index: %w", route.ErrInvalidEvent)
		}

		if validator.GetStatus() == nil {
			return fmt.Errorf("nil validator Status: %w", route.ErrInvalidEvent)
		}

		data := validator.GetData()
		if data == nil {
			return fmt.Errorf("nil validator Data: %w", route.ErrInvalidEvent)
		}

		if data.GetSlashed() == nil {
			return fmt.Errorf("nil validator Slashed: %w", route.ErrInvalidEvent)
		}

		b.UpdatedDateTime.Append(now)
		b.Epoch.Append(epoch)
		b.EpochStartDateTime.Append(epochStartTime)

		//nolint:gosec // G115: validator index bounded by uint32 column
		b.Index.Append(uint32(validator.GetIndex().GetValue()))

		// Store the exact value the beacon API returns, always as a set value
		// (never NULL). Balance 0 (an exited validator), a genesis epoch 0, and
		// the FAR_FUTURE_EPOCH sentinel (2^64-1) carried by a validator that is
		// not yet activated / not exiting / not withdrawable are all meaningful
		// values the API reports verbatim — the old setOptionalEpoch logic
		// discarded them to NULL, conflating real data with absence.
		b.Balance.Append(chProto.NewNullable[uint64](validator.GetBalance().GetValue()))
		b.Status.Append(validator.GetStatus().GetValue())
		b.EffectiveBalance.Append(chProto.NewNullable[uint64](data.GetEffectiveBalance().GetValue()))
		b.Slashed.Append(data.GetSlashed().GetValue())
		b.ActivationEpoch.Append(chProto.NewNullable[uint64](data.GetActivationEpoch().GetValue()))
		b.ActivationEligibilityEpoch.Append(chProto.NewNullable[uint64](data.GetActivationEligibilityEpoch().GetValue()))
		b.ExitEpoch.Append(chProto.NewNullable[uint64](data.GetExitEpoch().GetValue()))
		b.WithdrawableEpoch.Append(chProto.NewNullable[uint64](data.GetWithdrawableEpoch().GetValue()))

		b.appendMetadata(event)
		b.rows++
	}

	return nil
}

func (b *canonicalBeaconValidatorsBatch) validate(event *xatu.DecoratedEvent) error {
	extra := event.GetMeta().GetClient().GetEthV1Validators()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}
