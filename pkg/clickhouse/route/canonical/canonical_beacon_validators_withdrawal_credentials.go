package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconValidatorsWithdrawalCredentialsEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconValidatorsWithdrawalCredentialsTableName,
		canonicalBeaconValidatorsWithdrawalCredentialsEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconValidatorsWithdrawalCredentialsBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconValidatorsWithdrawalCredentialsBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1Validators() == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	var (
		epoch          uint32
		epochStartTime time.Time
	)

	if client := event.GetMeta().GetClient(); client != nil {
		if extra := client.GetEthV1Validators(); extra != nil {
			if epochData := extra.GetEpoch(); epochData != nil {
				if number := epochData.GetNumber(); number != nil {
					epoch = uint32(number.GetValue())
				}

				if start := epochData.GetStartDateTime(); start != nil {
					epochStartTime = start.AsTime()
				}
			}
		}
	}

	now := time.Now()

	for _, validator := range event.GetEthV1Validators().GetValidators() {
		// Canonical data is gap-sensitive: halt rather than skip or empty-fill.
		if validator == nil {
			return fmt.Errorf("nil validator entry: %w", route.ErrInvalidEvent)
		}

		if validator.GetIndex() == nil {
			return fmt.Errorf("nil validator Index: %w", route.ErrInvalidEvent)
		}

		if validator.GetData().GetWithdrawalCredentials().GetValue() == "" {
			return fmt.Errorf("empty validator WithdrawalCredentials: %w", route.ErrInvalidEvent)
		}

		b.UpdatedDateTime.Append(now)
		b.Epoch.Append(epoch)
		b.EpochStartDateTime.Append(epochStartTime)

		//nolint:gosec // G115: validator index bounded by uint32 column
		b.Index.Append(uint32(validator.GetIndex().GetValue()))
		b.WithdrawalCredentials.Append(validator.GetData().GetWithdrawalCredentials().GetValue())

		b.appendMetadata(event)
		b.rows++
	}

	return nil
}

func (b *canonicalBeaconValidatorsWithdrawalCredentialsBatch) validate(event *xatu.DecoratedEvent) error {
	extra := event.GetMeta().GetClient().GetEthV1Validators()
	if extra == nil || extra.GetEpoch() == nil || extra.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}
