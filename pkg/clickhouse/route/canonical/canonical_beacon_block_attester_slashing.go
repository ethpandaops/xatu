package canonical

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockAttesterSlashingEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockAttesterSlashingTableName,
		canonicalBeaconBlockAttesterSlashingEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockAttesterSlashingBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV2BeaconBlockAttesterSlashing() == nil {
		return fmt.Errorf("nil eth_v2_beacon_block_attester_slashing payload: %w", route.ErrInvalidEvent)
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

func (b *canonicalBeaconBlockAttesterSlashingBatch) validate(event *xatu.DecoratedEvent) error {
	slashing := event.GetEthV2BeaconBlockAttesterSlashing()

	if err := validateIndexedAttestation("Attestation_1", slashing.GetAttestation_1()); err != nil {
		return err
	}

	return validateIndexedAttestation("Attestation_2", slashing.GetAttestation_2())
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendPayload(event *xatu.DecoratedEvent) {
	slashing := event.GetEthV2BeaconBlockAttesterSlashing()
	b.appendIndexedAttestation1(slashing.GetAttestation_1())
	b.appendIndexedAttestation2(slashing.GetAttestation_2())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockAttesterSlashingBatch) appendIndexedAttestation1(att *ethv1.IndexedAttestationV2) {
	b.Attestation1AttestingIndices.Append(wrappedUint64SliceToUint32(att.GetAttestingIndices()))
	b.Attestation1Signature.Append(att.GetSignature())

	data := att.GetData()
	b.Attestation1DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
	b.Attestation1DataSlot.Append(uint32(data.GetSlot().GetValue()))
	b.Attestation1DataIndex.Append(uint32(data.GetIndex().GetValue()))

	source := data.GetSource()
	b.Attestation1DataSourceEpoch.Append(uint32(source.GetEpoch().GetValue()))
	b.Attestation1DataSourceRoot.Append([]byte(source.GetRoot()))

	target := data.GetTarget()
	b.Attestation1DataTargetEpoch.Append(uint32(target.GetEpoch().GetValue()))
	b.Attestation1DataTargetRoot.Append([]byte(target.GetRoot()))
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockAttesterSlashingBatch) appendIndexedAttestation2(att *ethv1.IndexedAttestationV2) {
	b.Attestation2AttestingIndices.Append(wrappedUint64SliceToUint32(att.GetAttestingIndices()))
	b.Attestation2Signature.Append(att.GetSignature())

	data := att.GetData()
	b.Attestation2DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))
	b.Attestation2DataSlot.Append(uint32(data.GetSlot().GetValue()))
	b.Attestation2DataIndex.Append(uint32(data.GetIndex().GetValue()))

	source := data.GetSource()
	b.Attestation2DataSourceEpoch.Append(uint32(source.GetEpoch().GetValue()))
	b.Attestation2DataSourceRoot.Append([]byte(source.GetRoot()))

	target := data.GetTarget()
	b.Attestation2DataTargetEpoch.Append(uint32(target.GetEpoch().GetValue()))
	b.Attestation2DataTargetRoot.Append([]byte(target.GetRoot()))
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockAttesterSlashing()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)
}

// validateIndexedAttestation returns route.ErrInvalidEvent when a required field
// of an indexed attestation is absent, so the slashing is dropped rather than
// written with fabricated zero slot/index/epoch values.
func validateIndexedAttestation(name string, att *ethv1.IndexedAttestationV2) error {
	if att == nil {
		return fmt.Errorf("nil %s: %w", name, route.ErrInvalidEvent)
	}

	data := att.GetData()
	if data == nil {
		return fmt.Errorf("nil %s.Data: %w", name, route.ErrInvalidEvent)
	}

	if data.GetSlot() == nil {
		return fmt.Errorf("nil %s.Data.Slot: %w", name, route.ErrInvalidEvent)
	}

	if data.GetIndex() == nil {
		return fmt.Errorf("nil %s.Data.Index: %w", name, route.ErrInvalidEvent)
	}

	source := data.GetSource()
	if source == nil || source.GetEpoch() == nil {
		return fmt.Errorf("nil %s.Data.Source: %w", name, route.ErrInvalidEvent)
	}

	target := data.GetTarget()
	if target == nil || target.GetEpoch() == nil {
		return fmt.Errorf("nil %s.Data.Target: %w", name, route.ErrInvalidEvent)
	}

	return nil
}

// wrappedUint64SliceToUint32 converts a slice of wrapperspb.UInt64Value to []uint32.
func wrappedUint64SliceToUint32(values []*wrapperspb.UInt64Value) []uint32 {
	out := make([]uint32, 0, len(values))

	for _, v := range values {
		if v == nil {
			continue
		}

		if v.GetValue() > uint64(^uint32(0)) {
			continue
		}

		out = append(out, uint32(v.GetValue())) //nolint:gosec // G115
	}

	return out
}
