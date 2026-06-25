package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconSyncCommitteeRewardEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconSyncCommitteeRewardTableName,
		canonicalBeaconSyncCommitteeRewardEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconSyncCommitteeRewardBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconSyncCommitteeRewardBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconSyncCommitteeReward() == nil {
		return fmt.Errorf("nil eth_v1_beacon_sync_committee_reward payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime()
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconSyncCommitteeRewardBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1BeaconSyncCommitteeReward()

	if payload.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", route.ErrInvalidEvent)
	}

	if payload.GetReward() == nil {
		return fmt.Errorf("nil Reward: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconSyncCommitteeRewardBatch) appendRuntime() {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema
func (b *canonicalBeaconSyncCommitteeRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	payload := event.GetEthV1BeaconSyncCommitteeReward()

	b.ValidatorIndex.Append(uint32(payload.GetValidatorIndex().GetValue()))
	b.Reward.Append(payload.GetReward().GetValue())
}

func (b *canonicalBeaconSyncCommitteeRewardBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconSyncCommitteeReward()
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
