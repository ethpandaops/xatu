package beacon

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func init() {
	flattener.MustRegister(flattener.NewStaticRoute(
		beaconApiEthV1EventsContributionAndProofTableName,
		[]xatu.Event_Name{xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2},
		func() flattener.ColumnarBatch {
			return newbeaconApiEthV1EventsContributionAndProofBatch()
		},
	))
}

func (b *beaconApiEthV1EventsContributionAndProofBatch) FlattenTo(
	event *xatu.DecoratedEvent,
	meta *metadata.CommonMetadata,
) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetEthV1EventsContributionAndProofV2()
	if payload == nil {
		return fmt.Errorf("nil EthV1EventsContributionAndProofV2 payload: %w", flattener.ErrInvalidEvent)
	}

	if err := b.validate(event); err != nil {
		return err
	}

	msg := payload.GetMessage()
	contrib := msg.GetContribution()
	addl := event.GetMeta().GetClient().GetEthV1EventsContributionAndProofV2()
	addlContrib := addl.GetContribution()

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.AggregatorIndex.Append(uint32(msg.GetAggregatorIndex().GetValue())) //nolint:gosec // G115: aggregator index fits uint32.
	b.ContributionSlot.Append(uint32(contrib.GetSlot().GetValue()))       //nolint:gosec // G115: slot fits uint32.
	b.ContributionSlotStartDateTime.Append(addlContrib.GetSlot().GetStartDateTime().AsTime())
	b.ContributionPropagationSlotStartDiff.Append(uint32(addlContrib.GetPropagation().GetSlotStartDiff().GetValue())) //nolint:gosec // G115: propagation diff fits uint32.
	b.ContributionBeaconBlockRoot.Append([]byte(contrib.GetBeaconBlockRoot()))
	b.ContributionSubcommitteeIndex.Append(strconv.FormatUint(contrib.GetSubcommitteeIndex().GetValue(), 10))
	b.ContributionAggregationBits.Append(contrib.GetAggregationBits())
	b.ContributionSignature.Append(contrib.GetSignature())
	b.ContributionEpoch.Append(uint32(addlContrib.GetEpoch().GetNumber().GetValue())) //nolint:gosec // G115: epoch fits uint32.
	b.ContributionEpochStartDateTime.Append(addlContrib.GetEpoch().GetStartDateTime().AsTime())
	b.SelectionProof.Append(msg.GetSelectionProof())
	b.Signature.Append(payload.GetSignature())

	b.appendMetadata(meta)
	b.rows++

	return nil
}

func (b *beaconApiEthV1EventsContributionAndProofBatch) validate(event *xatu.DecoratedEvent) error {
	payload := event.GetEthV1EventsContributionAndProofV2()

	if payload.GetMessage() == nil {
		return fmt.Errorf("nil contribution and proof Message: %w", flattener.ErrInvalidEvent)
	}

	msg := payload.GetMessage()

	if msg.GetAggregatorIndex() == nil {
		return fmt.Errorf("nil AggregatorIndex: %w", flattener.ErrInvalidEvent)
	}

	if msg.GetContribution() == nil {
		return fmt.Errorf("nil Contribution: %w", flattener.ErrInvalidEvent)
	}

	contrib := msg.GetContribution()

	if contrib.GetSlot() == nil {
		return fmt.Errorf("nil Contribution Slot: %w", flattener.ErrInvalidEvent)
	}

	addl := event.GetMeta().GetClient().GetEthV1EventsContributionAndProofV2()
	if addl == nil {
		return fmt.Errorf("nil EthV1EventsContributionAndProofV2 additional data: %w", flattener.ErrInvalidEvent)
	}

	addlContrib := addl.GetContribution()
	if addlContrib == nil {
		return fmt.Errorf("nil additional Contribution data: %w", flattener.ErrInvalidEvent)
	}

	if addlContrib.GetEpoch() == nil || addlContrib.GetEpoch().GetNumber() == nil {
		return fmt.Errorf("nil Contribution Epoch: %w", flattener.ErrInvalidEvent)
	}

	if addlContrib.GetSlot() == nil {
		return fmt.Errorf("nil additional Contribution Slot: %w", flattener.ErrInvalidEvent)
	}

	if addlContrib.GetPropagation() == nil || addlContrib.GetPropagation().GetSlotStartDiff() == nil {
		return fmt.Errorf("nil ContributionPropagationSlotStartDiff: %w", flattener.ErrInvalidEvent)
	}

	return nil
}
