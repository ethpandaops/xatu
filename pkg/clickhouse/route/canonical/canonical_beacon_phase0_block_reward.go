package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconPhase0BlockRewardEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_PHASE0_BLOCK_REWARD,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconPhase0BlockRewardTableName,
		canonicalBeaconPhase0BlockRewardEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconPhase0BlockRewardBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconPhase0BlockRewardBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconPhase0BlockReward() == nil {
		return fmt.Errorf("nil eth_v1_beacon_phase0_block_reward payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.appendMetadata(event)
	b.appendPayload(event)
	appendBlockIdentifier(event.GetMeta().GetClient().GetEthV1BeaconPhase0BlockReward().GetBlock(),
		&b.Slot, &b.SlotStartDateTime, &b.Epoch, &b.EpochStartDateTime, nil, &b.BlockRoot)
	b.rows++

	return nil
}

func (b *canonicalBeaconPhase0BlockRewardBatch) validate(event *xatu.DecoratedEvent) error {
	reward := event.GetEthV1BeaconPhase0BlockReward()

	if reward.GetProposerIndex() == nil {
		return fmt.Errorf("nil ProposerIndex: %w", route.ErrInvalidEvent)
	}

	if reward.GetTotal() == nil {
		return fmt.Errorf("nil Total: %w", route.ErrInvalidEvent)
	}

	if reward.GetAttestations() == nil {
		return fmt.Errorf("nil Attestations: %w", route.ErrInvalidEvent)
	}

	if reward.GetProposerSlashings() == nil {
		return fmt.Errorf("nil ProposerSlashings: %w", route.ErrInvalidEvent)
	}

	if reward.GetAttesterSlashings() == nil {
		return fmt.Errorf("nil AttesterSlashings: %w", route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthV1BeaconPhase0BlockReward().GetBlock() == nil {
		return fmt.Errorf("nil Block: %w", route.ErrInvalidEvent)
	}

	return nil
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse column schema.
func (b *canonicalBeaconPhase0BlockRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	reward := event.GetEthV1BeaconPhase0BlockReward()

	b.ProposerIndex.Append(uint32(reward.GetProposerIndex().GetValue()))
	b.Total.Append(reward.GetTotal().GetValue())
	b.Attestations.Append(reward.GetAttestations().GetValue())
	b.ProposerSlashings.Append(reward.GetProposerSlashings().GetValue())
	b.AttesterSlashings.Append(reward.GetAttesterSlashings().GetValue())
}
