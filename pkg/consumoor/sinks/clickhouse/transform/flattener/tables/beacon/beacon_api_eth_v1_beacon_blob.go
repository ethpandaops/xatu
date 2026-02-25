package beacon

import (
	"fmt"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1BeaconBlobTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1BeaconBlobBatch()
		},
	))
}

func (b *beaconApiEthV1BeaconBlobBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconBlob()
	if payload == nil {
		return fmt.Errorf("nil EthV1BeaconBlob payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconBlob()

	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(payload.GetSlot().GetValue())) //nolint:gosec // G115: slot fits uint32.
	b.SlotStartDateTime.Append(addl.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(payload.GetBlockRoot()))
	b.BlockParentRoot.Append([]byte(payload.GetBlockParentRoot()))
	b.ProposerIndex.Append(uint32(payload.GetProposerIndex().GetValue())) //nolint:gosec // G115: proposer index fits uint32.
	b.BlobIndex.Append(payload.GetIndex().GetValue())
	b.KzgCommitment.Append([]byte(payload.GetKzgCommitment()))
	b.VersionedHash.Append([]byte(payload.GetVersionedHash()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1BeaconBlobBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconBlob()

	if payload.GetSlot() == nil {
		return fmt.Errorf("nil Slot: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetIndex() == nil {
		return fmt.Errorf("nil Index: %w", flattener.ErrInvalidEvent)
	}

	if payload.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconBlob()
	if addl == nil {
		return fmt.Errorf("nil EthV1BeaconBlob additional data: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSlot() == nil {
		return fmt.Errorf("nil additional Slot: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
