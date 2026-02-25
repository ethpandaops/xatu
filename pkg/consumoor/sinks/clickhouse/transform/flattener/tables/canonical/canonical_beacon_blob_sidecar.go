package canonical

import (
	"fmt"
	"time"

	"github.com/ClickHouse/ch-go/proto"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlobSidecarTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlobSidecarBatch()
		},
	))
}

//nolint:gosec // G115: blob sidecar field values fit their target types.
func (b *canonicalBeaconBlobSidecarBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconBlockBlobSidecar()
	if payload == nil {
		return fmt.Errorf("nil BlobSidecar payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconBlobSidecar()

	b.UpdatedDateTime.Append(time.Now())
	b.Slot.Append(uint32(payload.GetSlot().GetValue()))
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.BlockParentRoot.Append([]byte(payload.GetBlockParentRoot()))
	b.VersionedHash.Append([]byte(addl.GetVersionedHash()))
	b.KzgCommitment.Append([]byte(payload.GetKzgCommitment()))
	b.KzgProof.Append([]byte(payload.GetKzgProof()))
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue()))
	b.BlobIndex.Append(payload.GetIndex().GetValue())

	if addl.GetDataSize() != nil {
		b.BlobSize.Append(uint32(addl.GetDataSize().GetValue()))
	} else {
		b.BlobSize.Append(0)
	}

	if addl.GetDataEmptySize() != nil {
		b.BlobEmptySize.Append(proto.NewNullable[uint32](uint32(addl.GetDataEmptySize().GetValue())))
	} else {
		b.BlobEmptySize.Append(proto.Nullable[uint32]{})
	}

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlobSidecarBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconBlockBlobSidecar()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconBlobSidecar()
	if addl == nil {
		return fmt.Errorf("nil BlobSidecar additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
