package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var canonicalBeaconBlockSyncAggregateEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconBlockSyncAggregateTableName,
		canonicalBeaconBlockSyncAggregateEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconBlockSyncAggregateBatch() },
	))
}

func (b *canonicalBeaconBlockSyncAggregateBatch) FlattenTo(
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

func (b *canonicalBeaconBlockSyncAggregateBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *canonicalBeaconBlockSyncAggregateBatch) appendPayload(event *xatu.DecoratedEvent) {
	aggregate := event.GetEthV2BeaconBlockSyncAggregate()
	if aggregate == nil {
		b.SyncCommitteeBits.Append("")
		b.SyncCommitteeSignature.Append("")
		b.ValidatorsParticipated.Append([]uint32{})
		b.ValidatorsMissed.Append([]uint32{})
		b.ParticipationCount.Append(0)

		return
	}

	b.SyncCommitteeBits.Append(aggregate.GetSyncCommitteeBits())
	b.SyncCommitteeSignature.Append(aggregate.GetSyncCommitteeSignature())
	b.ValidatorsParticipated.Append(syncAggregateWrappedUint32Slice(aggregate.GetValidatorsParticipated()))
	b.ValidatorsMissed.Append(syncAggregateWrappedUint32Slice(aggregate.GetValidatorsMissed()))

	if participationCount := aggregate.GetParticipationCount(); participationCount != nil {
		b.ParticipationCount.Append(uint16(participationCount.GetValue()))
	} else {
		b.ParticipationCount.Append(0)
	}
}

func (b *canonicalBeaconBlockSyncAggregateBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV2BeaconBlockSyncAggregate()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockVersion.Append("")
		b.BlockRoot.Append(nil)
		b.SyncCommitteePeriod.Append(0)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, &b.BlockVersion, &b.BlockRoot)

	if syncCommitteePeriod := additional.GetSyncCommitteePeriod(); syncCommitteePeriod != nil {
		b.SyncCommitteePeriod.Append(syncCommitteePeriod.GetValue())
	} else {
		b.SyncCommitteePeriod.Append(0)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func syncAggregateWrappedUint32Slice(values []*wrapperspb.UInt64Value) []uint32 {
	out := make([]uint32, 0, len(values))

	for _, value := range values {
		if value == nil {
			continue
		}

		if value.GetValue() > uint64(^uint32(0)) {
			continue
		}

		out = append(out, uint32(value.GetValue()))
	}

	return out
}
