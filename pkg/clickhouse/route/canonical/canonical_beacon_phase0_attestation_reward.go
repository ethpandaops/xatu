package canonical

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconPhase0AttestationRewardEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_PHASE0_ATTESTATION_REWARD,
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconPhase0AttestationRewardTableName,
		canonicalBeaconPhase0AttestationRewardEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconPhase0AttestationRewardBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconPhase0AttestationRewardBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconPhase0AttestationReward() == nil {
		return fmt.Errorf("nil eth_v1_beacon_phase0_attestation_reward payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.UpdatedDateTime.Append(time.Now())
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconPhase0AttestationRewardBatch) validate(event *xatu.DecoratedEvent) error {
	reward := event.GetEthV1BeaconPhase0AttestationReward()

	if reward.GetValidatorIndex() == nil {
		return fmt.Errorf("nil ValidatorIndex: %w", route.ErrInvalidEvent)
	}

	if reward.GetHead() == nil {
		return fmt.Errorf("nil Head: %w", route.ErrInvalidEvent)
	}

	if reward.GetTarget() == nil {
		return fmt.Errorf("nil Target: %w", route.ErrInvalidEvent)
	}

	if reward.GetSource() == nil {
		return fmt.Errorf("nil Source: %w", route.ErrInvalidEvent)
	}

	if reward.GetInclusionDelay() == nil {
		return fmt.Errorf("nil InclusionDelay: %w", route.ErrInvalidEvent)
	}

	if reward.GetInactivity() == nil {
		return fmt.Errorf("nil Inactivity: %w", route.ErrInvalidEvent)
	}

	epoch := event.GetMeta().GetClient().GetEthV1BeaconPhase0AttestationReward().GetEpoch()
	if epoch.GetNumber() == nil || epoch.GetStartDateTime() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	if event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName() == "" {
		return fmt.Errorf("empty meta_network_name: %w", route.ErrInvalidEvent)
	}

	return nil
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconPhase0AttestationRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	reward := event.GetEthV1BeaconPhase0AttestationReward()

	b.ValidatorIndex.Append(uint32(reward.GetValidatorIndex().GetValue()))
	b.Head.Append(reward.GetHead().GetValue())
	b.Target.Append(reward.GetTarget().GetValue())
	b.Source.Append(reward.GetSource().GetValue())
	b.InclusionDelay.Append(reward.GetInclusionDelay().GetValue())
	b.Inactivity.Append(reward.GetInactivity().GetValue())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconPhase0AttestationRewardBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	epoch := event.GetMeta().GetClient().GetEthV1BeaconPhase0AttestationReward().GetEpoch()

	b.Epoch.Append(uint32(epoch.GetNumber().GetValue()))
	b.EpochStartDateTime.Append(epoch.GetStartDateTime().AsTime())
}
