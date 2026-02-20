package canonical

import (
	"time"

	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockAttesterSlashingEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockAttesterSlashingTableName,
		canonicalBeaconBlockAttesterSlashingEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockAttesterSlashingBatch() },
	))
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if meta == nil {
		meta = metadata.Extract(event)
	}

	b.appendRuntime(event)
	b.appendMetadata(meta)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendPayload(event *xatu.DecoratedEvent) {
	slashing := event.GetEthV2BeaconBlockAttesterSlashing()
	if slashing == nil {
		b.appendNullIndexedAttestation1()
		b.appendNullIndexedAttestation2()

		return
	}

	b.appendIndexedAttestation1(slashing.GetAttestation_1())
	b.appendIndexedAttestation2(slashing.GetAttestation_2())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockAttesterSlashingBatch) appendIndexedAttestation1(att *ethv1.IndexedAttestationV2) {
	if att == nil {
		b.appendNullIndexedAttestation1()

		return
	}

	b.Attestation1AttestingIndices.Append(wrappedUint64SliceToUint32(att.GetAttestingIndices()))
	b.Attestation1Signature.Append(att.GetSignature())

	if data := att.GetData(); data != nil {
		b.Attestation1DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

		if slot := data.GetSlot(); slot != nil {
			b.Attestation1DataSlot.Append(uint32(slot.GetValue()))
		} else {
			b.Attestation1DataSlot.Append(0)
		}

		if index := data.GetIndex(); index != nil {
			b.Attestation1DataIndex.Append(uint32(index.GetValue()))
		} else {
			b.Attestation1DataIndex.Append(0)
		}

		if source := data.GetSource(); source != nil {
			if epoch := source.GetEpoch(); epoch != nil {
				b.Attestation1DataSourceEpoch.Append(uint32(epoch.GetValue()))
			} else {
				b.Attestation1DataSourceEpoch.Append(0)
			}

			b.Attestation1DataSourceRoot.Append([]byte(source.GetRoot()))
		} else {
			b.Attestation1DataSourceEpoch.Append(0)
			b.Attestation1DataSourceRoot.Append(nil)
		}

		if target := data.GetTarget(); target != nil {
			if epoch := target.GetEpoch(); epoch != nil {
				b.Attestation1DataTargetEpoch.Append(uint32(epoch.GetValue()))
			} else {
				b.Attestation1DataTargetEpoch.Append(0)
			}

			b.Attestation1DataTargetRoot.Append([]byte(target.GetRoot()))
		} else {
			b.Attestation1DataTargetEpoch.Append(0)
			b.Attestation1DataTargetRoot.Append(nil)
		}
	} else {
		b.Attestation1DataBeaconBlockRoot.Append(nil)
		b.Attestation1DataSlot.Append(0)
		b.Attestation1DataIndex.Append(0)
		b.Attestation1DataSourceEpoch.Append(0)
		b.Attestation1DataSourceRoot.Append(nil)
		b.Attestation1DataTargetEpoch.Append(0)
		b.Attestation1DataTargetRoot.Append(nil)
	}
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendNullIndexedAttestation1() {
	b.Attestation1AttestingIndices.Append([]uint32{})
	b.Attestation1Signature.Append("")
	b.Attestation1DataBeaconBlockRoot.Append(nil)
	b.Attestation1DataSlot.Append(0)
	b.Attestation1DataIndex.Append(0)
	b.Attestation1DataSourceEpoch.Append(0)
	b.Attestation1DataSourceRoot.Append(nil)
	b.Attestation1DataTargetEpoch.Append(0)
	b.Attestation1DataTargetRoot.Append(nil)
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconBlockAttesterSlashingBatch) appendIndexedAttestation2(att *ethv1.IndexedAttestationV2) {
	if att == nil {
		b.appendNullIndexedAttestation2()

		return
	}

	b.Attestation2AttestingIndices.Append(wrappedUint64SliceToUint32(att.GetAttestingIndices()))
	b.Attestation2Signature.Append(att.GetSignature())

	if data := att.GetData(); data != nil {
		b.Attestation2DataBeaconBlockRoot.Append([]byte(data.GetBeaconBlockRoot()))

		if slot := data.GetSlot(); slot != nil {
			b.Attestation2DataSlot.Append(uint32(slot.GetValue()))
		} else {
			b.Attestation2DataSlot.Append(0)
		}

		if index := data.GetIndex(); index != nil {
			b.Attestation2DataIndex.Append(uint32(index.GetValue()))
		} else {
			b.Attestation2DataIndex.Append(0)
		}

		if source := data.GetSource(); source != nil {
			if epoch := source.GetEpoch(); epoch != nil {
				b.Attestation2DataSourceEpoch.Append(uint32(epoch.GetValue()))
			} else {
				b.Attestation2DataSourceEpoch.Append(0)
			}

			b.Attestation2DataSourceRoot.Append([]byte(source.GetRoot()))
		} else {
			b.Attestation2DataSourceEpoch.Append(0)
			b.Attestation2DataSourceRoot.Append(nil)
		}

		if target := data.GetTarget(); target != nil {
			if epoch := target.GetEpoch(); epoch != nil {
				b.Attestation2DataTargetEpoch.Append(uint32(epoch.GetValue()))
			} else {
				b.Attestation2DataTargetEpoch.Append(0)
			}

			b.Attestation2DataTargetRoot.Append([]byte(target.GetRoot()))
		} else {
			b.Attestation2DataTargetEpoch.Append(0)
			b.Attestation2DataTargetRoot.Append(nil)
		}
	} else {
		b.Attestation2DataBeaconBlockRoot.Append(nil)
		b.Attestation2DataSlot.Append(0)
		b.Attestation2DataIndex.Append(0)
		b.Attestation2DataSourceEpoch.Append(0)
		b.Attestation2DataSourceRoot.Append(nil)
		b.Attestation2DataTargetEpoch.Append(0)
		b.Attestation2DataTargetRoot.Append(nil)
	}
}

func (b *canonicalBeaconBlockAttesterSlashingBatch) appendNullIndexedAttestation2() {
	b.Attestation2AttestingIndices.Append([]uint32{})
	b.Attestation2Signature.Append("")
	b.Attestation2DataBeaconBlockRoot.Append(nil)
	b.Attestation2DataSlot.Append(0)
	b.Attestation2DataIndex.Append(0)
	b.Attestation2DataSourceEpoch.Append(0)
	b.Attestation2DataSourceRoot.Append(nil)
	b.Attestation2DataTargetEpoch.Append(0)
	b.Attestation2DataTargetRoot.Append(nil)
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
