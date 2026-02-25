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
		beaconApiEthV1ValidatorAttestationDataTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1ValidatorAttestationDataBatch()
		},
	))
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1ValidatorAttestationData()
	if payload == nil {
		return fmt.Errorf("nil EthV1ValidatorAttestationData payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1ValidatorAttestationData()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.CommitteeIndex.Append(strconv.FormatUint(payload.GetIndex().GetValue(), 10))
	b.BeaconBlockRoot.Append([]byte(payload.GetBeaconBlockRoot()))
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.SourceEpoch.Append(uint32(payload.GetSource().GetEpoch().GetValue())) //nolint:gosec // G115: source epoch fits uint32.
	b.SourceEpochStartDateTime.Append(addl.GetSource().GetEpoch().GetStartDateTime().AsTime())
	b.SourceRoot.Append([]byte(payload.GetSource().GetRoot()))
	b.TargetEpoch.Append(uint32(payload.GetTarget().GetEpoch().GetValue())) //nolint:gosec // G115: target epoch fits uint32.
	b.TargetEpochStartDateTime.Append(addl.GetTarget().GetEpoch().GetStartDateTime().AsTime())
	b.TargetRoot.Append([]byte(payload.GetTarget().GetRoot()))
	b.RequestDateTime.Append(addl.GetSnapshot().GetTimestamp().AsTime())
	b.RequestDuration.Append(uint32(addl.GetSnapshot().GetRequestDurationMs().GetValue()))               //nolint:gosec // G115: request duration fits uint32.
	b.RequestSlotStartDiff.Append(uint32(addl.GetSnapshot().GetRequestedAtSlotStartDiffMs().GetValue())) //nolint:gosec // G115: request slot start diff fits uint32.

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1ValidatorAttestationDataBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1ValidatorAttestationData()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1ValidatorAttestationData()
	if addl == nil {
		return fmt.Errorf("nil EthV1ValidatorAttestationData additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
