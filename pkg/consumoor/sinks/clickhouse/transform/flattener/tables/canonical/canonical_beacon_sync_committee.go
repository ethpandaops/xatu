package canonical

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var canonicalBeaconSyncCommitteeEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
}

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		canonicalBeaconSyncCommitteeTableName,
		canonicalBeaconSyncCommitteeEventNames,
		func() flattener.ColumnarBatch { return newcanonicalBeaconSyncCommitteeBatch() },
	))
}

func (b *canonicalBeaconSyncCommitteeBatch) FlattenTo(
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

func (b *canonicalBeaconSyncCommitteeBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *canonicalBeaconSyncCommitteeBatch) appendPayload(event *xatu.DecoratedEvent) {
	committeeData := event.GetEthV1BeaconSyncCommittee()
	if committeeData == nil {
		b.ValidatorAggregates.Append([][]uint32{})

		return
	}

	syncCommittee := committeeData.GetSyncCommittee()
	if syncCommittee == nil {
		b.ValidatorAggregates.Append([][]uint32{})

		return
	}

	aggregates := syncCommittee.GetValidatorAggregates()
	validatorAggregates := make([][]uint32, 0, len(aggregates))

	for _, aggregate := range aggregates {
		if aggregate == nil {
			continue
		}

		validatorAggregates = append(validatorAggregates, syncCommitteeWrappedUint32Slice(aggregate.GetValidators()))
	}

	b.ValidatorAggregates.Append(validatorAggregates)
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconSyncCommitteeBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconSyncCommittee()
	if additional == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.SyncCommitteePeriod.Append(0)

		return
	}

	if epochData := additional.GetEpoch(); epochData != nil {
		if epochNumber := epochData.GetNumber(); epochNumber != nil {
			b.Epoch.Append(uint32(epochNumber.GetValue()))
		} else {
			b.Epoch.Append(0)
		}

		if startDateTime := epochData.GetStartDateTime(); startDateTime != nil {
			b.EpochStartDateTime.Append(startDateTime.AsTime())
		} else {
			b.EpochStartDateTime.Append(time.Time{})
		}
	} else {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
	}

	if syncCommitteePeriod := additional.GetSyncCommitteePeriod(); syncCommitteePeriod != nil {
		b.SyncCommitteePeriod.Append(syncCommitteePeriod.GetValue())
	} else {
		b.SyncCommitteePeriod.Append(0)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func syncCommitteeWrappedUint32Slice(values []*wrapperspb.UInt64Value) []uint32 {
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
