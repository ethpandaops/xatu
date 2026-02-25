package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconElaboratedAttestationTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconElaboratedAttestationBatch()
		},
	))
}

//nolint:gosec // G115: attestation field values fit uint32.
func (b *canonicalBeaconElaboratedAttestationBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockElaboratedAttestation()
	if payload == nil {
		return fmt.Errorf("nil ElaboratedAttestation payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockElaboratedAttestation()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.BlockSlot.Append(uint32(blk.GetSlot().GetNumber().GetValue()))
	b.BlockSlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.BlockEpoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue()))
	b.BlockEpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.PositionInBlock.Append(uint32(addl.GetPositionInBlock().GetValue()))
	b.BlockRoot.Append([]byte(blk.GetRoot()))

	validators := make([]uint32, 0, len(payload.GetValidatorIndexes()))
	for _, v := range payload.GetValidatorIndexes() {
		validators = append(validators, uint32(v.GetValue()))
	}

	b.Validators.Append(validators)

	data := payload.GetData()
	b.CommitteeIndex.Append(fmt.Sprintf("%d", data.GetIndex().GetValue()))
	b.BeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

	// Attestation slot/epoch from additional data.
	b.Slot.Append(uint32(addl.GetSlot().GetNumber().GetValue()))
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())

	// Source.
	b.SourceEpoch.Append(uint32(addl.GetSource().GetEpoch().GetNumber().GetValue()))
	b.SourceEpochStartDateTime.Append(addl.GetSource().GetEpoch().GetStartDateTime().AsTime())
	b.SourceRoot.Append([]byte(data.GetSource().GetRoot()))

	// Target.
	b.TargetEpoch.Append(uint32(addl.GetTarget().GetEpoch().GetNumber().GetValue()))
	b.TargetEpochStartDateTime.Append(addl.GetTarget().GetEpoch().GetStartDateTime().AsTime())
	b.TargetRoot.Append([]byte(data.GetTarget().GetRoot()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconElaboratedAttestationBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockElaboratedAttestation()
	if addl == nil {
		return fmt.Errorf("nil ElaboratedAttestation additional data: %w", flattener.ErrInvalidEvent)
	}

	blk := addl.GetBlock()
	if blk == nil {
		return fmt.Errorf("nil Block identifier: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetEpoch() == nil || blk.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Block Epoch: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetSlot() == nil || blk.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Block Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil || addl.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSource() == nil || addl.GetSource().GetEpoch() == nil ||
		addl.GetSource().GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Source: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetTarget() == nil || addl.GetTarget().GetEpoch() == nil ||
		addl.GetTarget().GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Target: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
