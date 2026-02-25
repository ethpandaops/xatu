package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const canonicalBeaconValidatorsWithdrawalCredentialsVersion uint32 = 1

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconValidatorsWithdrawalCredentialsTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconValidatorsWithdrawalCredentialsBatch()
		},
	))
}

func (b *canonicalBeaconValidatorsWithdrawalCredentialsBatch) FlattenTo(
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
		if validator.GetIndex() == nil {
			continue
		}

		b.UpdatedDateTime.Append(time.Now())
		b.Version.Append(canonicalBeaconValidatorsWithdrawalCredentialsVersion)
		b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
		b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
		b.Index.Append(uint32(validator.GetIndex().GetValue())) //nolint:gosec // G115: validator index fits uint32.
		b.WithdrawalCredentials.Append(validator.GetData().GetWithdrawalCredentials().GetValue())

		b.appendMetadata(meta)
		b.rows++
	}

	return nil
}

func (b *canonicalBeaconValidatorsWithdrawalCredentialsBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV1Validators()
	if addl == nil {
		return fmt.Errorf("nil EthV1Validators additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
