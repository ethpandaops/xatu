package canonical

import (
	"fmt"
	"strings"
	"time"

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

	b.appendRuntime(event)
	b.appendMetadata(event)
	b.appendPayload(event)
	b.appendAdditionalData(event)
	b.rows++

	return nil
}

func (b *canonicalBeaconAttestationRewardBatch) appendRuntime(_ *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconAttestationRewardBatch) appendPayload(event *xatu.DecoratedEvent) {
	reward := event.GetEthV1BeaconAttestationReward()

	if validatorIndex := reward.GetValidatorIndex(); validatorIndex != nil {
		b.ValidatorIndex.Append(uint32(validatorIndex.GetValue()))
	} else {
		b.ValidatorIndex.Append(0)
	}

	if head := reward.GetHead(); head != nil {
		b.Head.Append(head.GetValue())
	} else {
		b.Head.Append(0)
	}

	if target := reward.GetTarget(); target != nil {
		b.Target.Append(target.GetValue())
	} else {
		b.Target.Append(0)
	}

	if source := reward.GetSource(); source != nil {
		b.Source.Append(source.GetValue())
	} else {
		b.Source.Append(0)
	}

	if inclusionDelay := reward.GetInclusionDelay(); inclusionDelay != nil {
		b.InclusionDelay.Append(inclusionDelay.GetValue())
	} else {
		b.InclusionDelay.Append(0)
	}

	if inactivity := reward.GetInactivity(); inactivity != nil {
		b.Inactivity.Append(inactivity.GetValue())
	} else {
		b.Inactivity.Append(0)
	}
}

//nolint:gosec // G115: proto uint64 values are bounded by ClickHouse uint32 column schema
func (b *canonicalBeaconAttestationRewardBatch) appendAdditionalData(event *xatu.DecoratedEvent) {
	additional := event.GetMeta().GetClient().GetEthV1BeaconAttestationReward()
	if additional == nil {
		b.Epoch.Append(0)
		b.EpochStartDateTime.Append(time.Time{})

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
}
