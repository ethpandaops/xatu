package canonical

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconCommitteeTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconCommitteeBatch()
		},
	))
}

func (b *canonicalBeaconCommitteeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconCommittee()
	if payload == nil {
		return fmt.Errorf("nil EthV1BeaconCommittee payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconCommittee()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.CommitteeIndex.Append(strconv.FormatUint(payload.GetIndex().GetValue(), 10))
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	validators := make([]uint32, 0, len(payload.GetValidators()))
	for _, v := range payload.GetValidators() {
		if v != nil {
			validators = append(validators, uint32(v.GetValue())) //nolint:gosec // G115: validator index fits uint32.
		}
	}

	b.Validators.Append(validators)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconCommitteeBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconCommittee()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconCommittee()
	if addl == nil {
		return fmt.Errorf("nil EthV1BeaconCommittee additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
