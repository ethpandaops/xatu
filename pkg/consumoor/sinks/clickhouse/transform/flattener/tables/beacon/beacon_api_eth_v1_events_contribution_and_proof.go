package beacon

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var beaconApiEthV1EventsContributionAndProofEventNames = []xatu.Event_Name{
	xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
}

func init() {
	catalog.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsContributionAndProofTableName,
		beaconApiEthV1EventsContributionAndProofEventNames,
		func() flattener.ColumnarBatch { return newbeaconApiEthV1EventsContributionAndProofBatch() },
	))
}

func (b *beaconApiEthV1EventsContributionAndProofBatch) FlattenTo(
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

func (b *beaconApiEthV1EventsContributionAndProofBatch) appendRuntime(event *xatu.DecoratedEvent) {
	b.UpdatedDateTime.Append(time.Now())

	if ts := event.GetEvent().GetDateTime(); ts != nil {
		b.EventDateTime.Append(ts.AsTime())
	} else {
		b.EventDateTime.Append(time.Time{})
	}
}

func (b *beaconApiEthV1EventsContributionAndProofBatch) appendPayload(event *xatu.DecoratedEvent) {
	eventContributionV2 := event.GetEthV1EventsContributionAndProofV2()
	if eventContributionV2 == nil {
		b.Signature.Append("")
		b.AggregatorIndex.Append(0)
		b.SelectionProof.Append("")
		b.ContributionSlot.Append(0)
		b.ContributionBeaconBlockRoot.Append(nil)
		b.ContributionSubcommitteeIndex.Append("")
		b.ContributionAggregationBits.Append("")
		b.ContributionSignature.Append("")

		return
	}

	b.Signature.Append(eventContributionV2.GetSignature())

	message := eventContributionV2.GetMessage()
	if message == nil {
		b.AggregatorIndex.Append(0)
		b.SelectionProof.Append("")
		b.ContributionSlot.Append(0)
		b.ContributionBeaconBlockRoot.Append(nil)
		b.ContributionSubcommitteeIndex.Append("")
		b.ContributionAggregationBits.Append("")
		b.ContributionSignature.Append("")

		return
	}

	if aggregatorIndex := message.GetAggregatorIndex(); aggregatorIndex != nil {
		b.AggregatorIndex.Append(uint32(aggregatorIndex.GetValue())) //nolint:gosec // aggregator index fits uint32
	} else {
		b.AggregatorIndex.Append(0)
	}

	b.SelectionProof.Append(message.GetSelectionProof())

	contribution := message.GetContribution()
	if contribution == nil {
		b.ContributionSlot.Append(0)
		b.ContributionBeaconBlockRoot.Append(nil)
		b.ContributionSubcommitteeIndex.Append("")
		b.ContributionAggregationBits.Append("")
		b.ContributionSignature.Append("")

		return
	}

	if slot := contribution.GetSlot(); slot != nil {
		b.ContributionSlot.Append(uint32(slot.GetValue())) //nolint:gosec // slot fits uint32
	} else {
		b.ContributionSlot.Append(0)
	}

	b.ContributionBeaconBlockRoot.Append([]byte(contribution.GetBeaconBlockRoot()))

	if subcommitteeIndex := contribution.GetSubcommitteeIndex(); subcommitteeIndex != nil {
		b.ContributionSubcommitteeIndex.Append(fmt.Sprint(subcommitteeIndex.GetValue()))
	} else {
		b.ContributionSubcommitteeIndex.Append("")
	}

	b.ContributionAggregationBits.Append(contribution.GetAggregationBits())
	b.ContributionSignature.Append(contribution.GetSignature())
}

func (b *beaconApiEthV1EventsContributionAndProofBatch) appendAdditionalData(
	event *xatu.DecoratedEvent,
) {
	if event.GetMeta() == nil || event.GetMeta().GetClient() == nil {
		b.ContributionSlotStartDateTime.Append(time.Time{})
		b.ContributionPropagationSlotStartDiff.Append(0)
		b.ContributionEpoch.Append(0)
		b.ContributionEpochStartDateTime.Append(time.Time{})

		return
	}

	additionalV2 := event.GetMeta().GetClient().GetEthV1EventsContributionAndProofV2()
	if additionalV2 == nil {
		b.ContributionSlotStartDateTime.Append(time.Time{})
		b.ContributionPropagationSlotStartDiff.Append(0)
		b.ContributionEpoch.Append(0)
		b.ContributionEpochStartDateTime.Append(time.Time{})

		return
	}

	contribution := additionalV2.GetContribution()
	if contribution == nil {
		b.ContributionSlotStartDateTime.Append(time.Time{})
		b.ContributionPropagationSlotStartDiff.Append(0)
		b.ContributionEpoch.Append(0)
		b.ContributionEpochStartDateTime.Append(time.Time{})

		return
	}

	additional := extractBeaconSlotEpochPropagation(contribution)

	b.ContributionSlotStartDateTime.Append(time.Unix(additional.SlotStartDateTime, 0))
	b.ContributionPropagationSlotStartDiff.Append(uint32(additional.PropagationSlotStartDiff)) //nolint:gosec // propagation diff fits uint32
	b.ContributionEpochStartDateTime.Append(time.Unix(additional.EpochStartDateTime, 0))

	if contributionEpoch := contribution.GetEpoch().GetNumber(); contributionEpoch != nil {
		b.ContributionEpoch.Append(uint32(contributionEpoch.GetValue())) //nolint:gosec // epoch fits uint32
	} else {
		b.ContributionEpoch.Append(0)
	}
}
