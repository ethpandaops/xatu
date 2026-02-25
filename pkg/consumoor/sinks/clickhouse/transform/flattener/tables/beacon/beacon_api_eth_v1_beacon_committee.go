package beacon

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
		beaconApiEthV1BeaconCommitteeTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1BeaconCommitteeBatch()
		},
	))
}

func (b *beaconApiEthV1BeaconCommitteeBatch) FlattenTo(
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
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.CommitteeIndex.Append(strconv.FormatUint(payload.GetIndex().GetValue(), 10))

	// Convert validators from []*wrapperspb.UInt64Value to []uint32.
	validators := payload.GetValidators()
	vals := make([]uint32, 0, len(validators))

	for _, v := range validators {
		vals = append(vals, uint32(v.GetValue())) //nolint:gosec // G115: validator index fits uint32.
	}

	b.Validators.Append(vals)
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1BeaconCommitteeBatch) validate(event *xatu.DecoratedEvent) error {
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
