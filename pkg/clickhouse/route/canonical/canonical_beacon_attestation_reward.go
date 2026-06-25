package canonical

import (
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var canonicalBeaconAttestationRewardEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD,
}

var canonicalBeaconAttestationRewardPredicate = func(event *xatu.DecoratedEvent) bool {
	if event == nil || event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		return false
	}

	stateID := event.GetMeta().GetClient().GetEthV1BeaconAttestationReward().GetStateId()

	return strings.EqualFold(stateID, "finalized")
}

func init() {
	r, err := route.NewStaticRoute(
		canonicalBeaconAttestationRewardTableName,
		canonicalBeaconAttestationRewardEventNames,
		func() route.ColumnarBatch { return newcanonicalBeaconAttestationRewardBatch() },
		route.WithStaticRoutePredicate(canonicalBeaconAttestationRewardPredicate),
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *canonicalBeaconAttestationRewardBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	if event.GetEthV1BeaconAttestationReward() == nil {
		return fmt.Errorf("nil eth_v1_beacon_attestation_reward payload: %w", route.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconAttestationRewardBatch) validate(event *xatu.DecoratedEvent) error {
	reward := event.GetEthV1BeaconAttestationReward()

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

	if reward.GetInactivity() == nil {
		return fmt.Errorf("nil Inactivity: %w", route.ErrInvalidEvent)
	}

	additional := event.GetMeta().GetClient().GetEthV1BeaconAttestationReward()
	if additional == nil || additional.GetEpoch() == nil || additional.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Epoch: %w", route.ErrInvalidEvent)
	}

	return nil
}

func (b *canonicalBeaconAttestationRewardBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconAttestationRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	reward := event.GetEthV1BeaconAttestationReward()

	b.ValidatorIndex.Append(uint32(reward.GetValidatorIndex().GetValue()))
	b.Head.Append(reward.GetHead().GetValue())
	b.Target.Append(reward.GetTarget().GetValue())
	b.Source.Append(reward.GetSource().GetValue())

	if inclusionDelay := reward.GetInclusionDelay(); inclusionDelay != nil {
		b.InclusionDelay.Append(proto.NewNullable[uint64](inclusionDelay.GetValue()))
	} else {
		b.InclusionDelay.Append(proto.Nullable[uint64]{})
	}

	b.Inactivity.Append(reward.GetInactivity().GetValue())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconAttestationRewardBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	epochData := event.GetMeta().GetClient().GetEthV1BeaconAttestationReward().GetEpoch()

	b.Epoch.Append(uint32(epochData.GetNumber().GetValue()))

	if startDateTime := epochData.GetStartDateTime(); startDateTime != nil {
		b.EpochStartDateTime.Append(startDateTime.AsTime())
	} else {
		b.EpochStartDateTime.Append(time.Time{})
	}
}
