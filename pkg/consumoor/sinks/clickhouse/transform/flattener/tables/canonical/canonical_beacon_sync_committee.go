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
		canonicalBeaconSyncCommitteeTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE},
		func() flattener.ColumnarBatch {
			return newcanonicalBeaconSyncCommitteeBatch()
		},
	))
}

func (b *canonicalBeaconSyncCommitteeBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1BeaconSyncCommittee()
	if payload == nil {
		return fmt.Errorf("nil EthV1BeaconSyncCommittee payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	addl := event.GetMeta().GetClient().GetEthV1BeaconSyncCommittee()
	sc := payload.GetSyncCommittee()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Epoch.Append(uint32(addl.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.EpochStartDateTime.Append(addl.GetEpoch().GetStartDateTime().AsTime())
	b.SyncCommitteePeriod.Append(addl.GetSyncCommitteePeriod().GetValue())

	// Build the nested array of validator aggregates.
	aggregates := make([][]uint32, 0, len(sc.GetValidatorAggregates()))

	for _, agg := range sc.GetValidatorAggregates() {
		vals := make([]uint32, 0, len(agg.GetValidators()))
		for _, v := range agg.GetValidators() {
			if v != nil {
				vals = append(vals, uint32(v.GetValue())) //nolint:gosec // G115: validator index fits uint32.
			}
		}

		aggregates = append(aggregates, vals)
	}

	b.ValidatorAggregates.Append(aggregates)

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *canonicalBeaconSyncCommitteeBatch) validate(event *xatu.DecoratedEvent) error {
	addl := event.GetMeta().GetClient().GetEthV1BeaconSyncCommittee()
	if addl == nil {
		return fmt.Errorf(
			"nil EthV1BeaconSyncCommittee additional data: %w",
			flattener.ErrInvalidEvent,
		)
	}

	if addl.GetEpoch() == nil || addl.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addl.GetSyncCommitteePeriod() == nil {
		return fmt.Errorf("nil SyncCommitteePeriod: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
