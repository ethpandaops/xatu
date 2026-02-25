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
		canonicalBeaconBlockSyncAggregateTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconBlockSyncAggregateBatch()
		},
	))
}

//nolint:gosec // G115: sync aggregate field values fit their target types.
func (b *canonicalBeaconBlockSyncAggregateBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV2BeaconBlockSyncAggregate()
	if payload == nil {
		return fmt.Errorf("nil SyncAggregate payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockSyncAggregate()
	blk := addl.GetBlock()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Slot.Append(uint32(blk.GetSlot().GetNumber().GetValue()))
	b.SlotStartDateTime.Append(blk.GetSlot().GetStartDateTime().AsTime())
	b.Epoch.Append(uint32(blk.GetEpoch().GetNumber().GetValue()))
	b.EpochStartDateTime.Append(blk.GetEpoch().GetStartDateTime().AsTime())
	b.BlockRoot.Append([]byte(blk.GetRoot()))
	b.BlockVersion.Append(blk.GetVersion())

	b.SyncCommitteePeriod.Append(addl.GetSyncCommitteePeriod().GetValue())
	b.SyncCommitteeBits.Append(payload.GetSyncCommitteeBits())
	b.SyncCommitteeSignature.Append(payload.GetSyncCommitteeSignature())

	participated := make([]uint32, 0, len(payload.GetValidatorsParticipated()))
	for _, v := range payload.GetValidatorsParticipated() {
		participated = append(participated, uint32(v.GetValue()))
	}

	b.ValidatorsParticipated.Append(participated)

	missed := make([]uint32, 0, len(payload.GetValidatorsMissed()))
	for _, v := range payload.GetValidatorsMissed() {
		missed = append(missed, uint32(v.GetValue()))
	}

	b.ValidatorsMissed.Append(missed)

	b.ParticipationCount.Append(uint16(payload.GetParticipationCount().GetValue()))

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockSyncAggregateBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV2BeaconBlockSyncAggregate()
	if addl == nil {
		return fmt.Errorf("nil SyncAggregate additional data: %w", flattener.ErrInvalidEvent)
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
