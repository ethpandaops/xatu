package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconStateBuilderEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconStateBuilderTableName,
		canonicalBeaconStateBuilderEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconStateBuilderBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconStateBuilderBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconStateBuilder()
	if payload == nil {
		return fmt.Errorf("nil payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconStateBuilderBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconStateBuilder()

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil Index: %w", route.ErrInvalidEvent)
	}

	if payload.GetPubkey() == "" {
		return fmt.Errorf("nil Pubkey: %w", route.ErrInvalidEvent)
	}

	if payload.GetVersion() == nil {
		return fmt.Errorf("nil Version: %w", route.ErrInvalidEvent)
	}

	if payload.GetExecutionAddress() == "" {
		return fmt.Errorf("nil ExecutionAddress: %w", route.ErrInvalidEvent)
	}

	if payload.GetBalance() == nil {
		return fmt.Errorf("nil Balance: %w", route.ErrInvalidEvent)
	}

	if payload.GetDepositEpoch() == nil {
		return fmt.Errorf("nil DepositEpoch: %w", route.ErrInvalidEvent)
	}

	if payload.GetWithdrawableEpoch() == nil {
		return fmt.Errorf("nil WithdrawableEpoch: %w", route.ErrInvalidEvent)
	}

	if payload.GetStatus() == "" {
		return fmt.Errorf("nil Status: %w", route.ErrInvalidEvent)
	}

	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilder()
	if extra == nil {
		return fmt.Errorf("nil additional data: %w", route.ErrInvalidEvent)
	}

	return validateEpochData(extra.GetEpoch())
}

func (b *canonicalBeaconStateBuilderBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconStateBuilderBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconStateBuilder()

	b.BuilderIndex.Append(payload.GetIndex().GetValue())
	b.Pubkey.Append([]byte(payload.GetPubkey()))
	b.Version.Append(uint8(payload.GetVersion().GetValue())) //nolint:gosec // builder version is a single byte
	b.ExecutionAddress.Append([]byte(payload.GetExecutionAddress()))
	b.Balance.Append(payload.GetBalance().GetValue())
	b.DepositEpoch.Append(payload.GetDepositEpoch().GetValue())
	b.WithdrawableEpoch.Append(payload.GetWithdrawableEpoch().GetValue())
	b.Status.Append(payload.GetStatus())
}

func (b *canonicalBeaconStateBuilderBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilder()

	b.Epoch.Append(uint32(extra.GetEpoch().GetNumber().GetValue())) //nolint:gosec // bounded by uint32 column
	b.EpochStartDateTime.Append(timeOrZero(extra.GetEpoch().GetStartDateTime()))
	b.StateID.Append(extra.GetStateId())
}
