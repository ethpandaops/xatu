package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconBlockRewardEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconBlockRewardTableName,
		canonicalBeaconBlockRewardEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconBlockRewardBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconBlockRewardBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconBlockReward() == nil {
		return fmt.Errorf("nil eth_v1_beacon_block_reward payload: %w", route.ErrInvalidEvent)
	}

	b.appendRuntime()
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconBlockRewardBatch) appendRuntime() {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema.
func (b *canonicalBeaconBlockRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	reward := event.GetEthV1BeaconBlockReward()

	if proposerIndex := reward.GetProposerIndex(); proposerIndex != nil {
		b.ProposerIndex.Append(uint32(proposerIndex.GetValue()))
	} else {
		b.ProposerIndex.Append(0)
	}

	b.Total.Append(reward.GetTotal().GetValue())
	b.Attestations.Append(reward.GetAttestations().GetValue())
	b.SyncAggregate.Append(reward.GetSyncAggregate().GetValue())
	b.ProposerSlashings.Append(reward.GetProposerSlashings().GetValue())
	b.AttesterSlashings.Append(reward.GetAttesterSlashings().GetValue())
}

func (b *canonicalBeaconBlockRewardBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconBlockReward()
	if additional == nil {
		b.Slot.Append(0)
		b.SlotStartDateTime.Append(time.Time{})
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})
		b.BlockRoot.Append(nil)

		return
	}

	appendBlockIdentifier(additional.GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, nil, &b.BlockRoot)
}
