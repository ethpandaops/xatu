package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1ProposerDutyTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1ProposerDutyBatch()
		},
	))
}

func (b *beaconApiEthV1ProposerDutyBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1ProposerDuty()
	if payload == nil {
		return fmt.Errorf("nil EthV1ProposerDuty payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1ProposerDuty()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.ProposerValidatorIndex.Append(uint32(payload.GetValidatorIndex().GetValue())) //nolint:gosec // G115: validator index fits uint32.
	b.ProposerPubkey.Append(payload.GetPubkey())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1ProposerDutyBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1ProposerDuty()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1ProposerDuty()
	if addl == nil {
		return fmt.Errorf("nil EthV1ProposerDuty additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
