package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockAttesterSlashingTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockAttesterSlashingBatch()
		},
	))
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockAttesterSlashing()
	if payload == nil {
		return fmt.Errorf("nil AttesterSlashing payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockAttesterSlashing()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(blk.GetSlot().GetNumber().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(blk.GetRoot()))
	b.BlockVersion.Append(blk.GetVersion())

	appendIndexedAttestation(b, payload.GetAttestation_1(), true)
	appendIndexedAttestation(b, payload.GetAttestation_2(), false)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockAttesterSlashing()
	if addl == nil {
		return fmt.Errorf("nil AttesterSlashing additional data: %w", flattener.ErrInvalidEvent)
	}

	blk := addl.GetBlock()
	if blk == nil {
		return fmt.Errorf("nil Block identifier: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetEpoch() == nil || blk.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if blk.GetSlot() == nil || blk.GetSlot().GetNumber() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}

//nolint:gosec // G115: indices fit uint32.
func appendIndexedAttestation(
	b *canonicalBeaconBlockAttesterSlashingBatch,
	att *v1.IndexedAttestationV2,
	isFirst bool,
) {
	indices := make([]uint32, 0, len(att.GetAttestingIndices()))
	for _, idx := range att.GetAttestingIndices() {
		indices = append(indices, uint32(idx.GetValue()))
	}

	data := att.GetData()

	if isFirst {
		b.Attestation1AttestingIndices.Append(indices)
		b.Attestation1Signature.Append(att.GetSignature())
		b.Attestation1DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
		b.Attestation1DataSlot.Append(uint32(data.GetSlot().GetValue()))
		b.Attestation1DataIndex.Append(uint32(data.GetIndex().GetValue()))
		b.Attestation1DataSourceEpoch.Append(uint32(data.GetSource().GetEpoch().GetValue()))
		b.Attestation1DataSourceRoot.Append([]byte(data.GetSource().GetRoot()))
		b.Attestation1DataTargetEpoch.Append(uint32(data.GetTarget().GetEpoch().GetValue()))
		b.Attestation1DataTargetRoot.Append([]byte(data.GetTarget().GetRoot()))
	} else {
		b.Attestation2AttestingIndices.Append(indices)
		b.Attestation2Signature.Append(att.GetSignature())
		b.Attestation2DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
		b.Attestation2DataSlot.Append(uint32(data.GetSlot().GetValue()))
		b.Attestation2DataIndex.Append(uint32(data.GetIndex().GetValue()))
		b.Attestation2DataSourceEpoch.Append(uint32(data.GetSource().GetEpoch().GetValue()))
		b.Attestation2DataSourceRoot.Append([]byte(data.GetSource().GetRoot()))
		b.Attestation2DataTargetEpoch.Append(uint32(data.GetTarget().GetEpoch().GetValue()))
		b.Attestation2DataTargetRoot.Append([]byte(data.GetTarget().GetRoot()))
	}
}
