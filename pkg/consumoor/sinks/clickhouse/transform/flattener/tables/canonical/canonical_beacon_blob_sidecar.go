package canonical

import (
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlobSidecarEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlobSidecarTableName,
		canonicalBeaconBlobSidecarEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlobSidecarBatch() },
	))
}

func (b *canonicalBeaconBlobSidecarBatch) FlattenTo(
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

func (b *canonicalBeaconBlobSidecarBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

func (b *canonicalBeaconBlobSidecarBatch) appendPayload(event *xatu.DecoratedEvent) {
	blob := event.GetEthV1BeaconBlockBlobSidecar()
	if blob == nil {
		b.Slot.Append(0)
		b.BlockRoot.Append(nil)
		b.BlockParentRoot.Append(nil)
		b.ProposerIndex.Append(0)
		b.KzgCommitment.Append(nil)
		b.KzgProof.Append(nil)
		b.BlobIndex.Append(0)

		return
	}

	if slot := blob.GetSlot(); slot != nil {
		b.Slot.Append(uint32(slot.GetValue())) //nolint:gosec // G115
	} else {
		b.Slot.Append(0)
	}

	b.BlockRoot.Append([]byte(blob.GetBlockRoot()))
	b.BlockParentRoot.Append([]byte(blob.GetBlockParentRoot()))

	if proposerIndex := blob.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue())) //nolint:gosec // G115
	} else {
		b.ProposerIndex.Append(0)
	}

	b.KzgCommitment.Append([]byte(blob.GetKzgCommitment()))
	b.KzgProof.Append([]byte(blob.GetKzgProof()))

	if index := blob.GetIndex(); index != nil {
		b.BlobIndex.Append(index.GetValue())
	} else {
		b.BlobIndex.Append(0)
	}
}

func (b *canonicalBeaconBlobSidecarBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconBlobSidecar()
	if additional == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SlotStartDateTime.Append(time.Time{})
		b.VersionedHash.Append(nil)
		b.BlobSize.Append(0)
		b.BlobEmptySize.Append(proto.Nullable[uint32]{})

		return
	}

	if epoch := additional.GetEpoch(); epoch != nil {
		if epochNumber := epoch.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue())) //nolint:gosec // G115
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epoch.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if slot := additional.GetSlot(); slot != nil {
		if startDateTime := slot.GetStartDateTime(); startDateTime != nil {
			b.SlotStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.SlotStartDateTime.Append(time.Time{})
		}
	} else {
		b.SlotStartDateTime.Append(time.Time{})
	}

	b.VersionedHash.Append([]byte(additional.GetVersionedHash()))

	if dataSize := additional.GetDataSize(); dataSize != nil {
		b.BlobSize.Append(uint32(dataSize.GetValue())) //nolint:gosec // G115
	} else {
		b.BlobSize.Append(0)
	}

	if dataEmptySize := additional.GetDataEmptySize(); dataEmptySize != nil {
		b.BlobEmptySize.Append(proto.NewNullable[uint32](uint32(dataEmptySize.GetValue()))) //nolint:gosec // G115
	} else {
		b.BlobEmptySize.Append(proto.Nullable[uint32]{})
	}
}
