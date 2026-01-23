package extractors

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "beacon_committee",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeEpoch,
		EpochProcessor: ProcessBeaconCommittees,
	})
}

// ProcessBeaconCommittees fetches and creates events for all beacon committees in an epoch.
func ProcessBeaconCommittees(
	ctx context.Context,
	epoch phase0.Epoch,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	sp, err := beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get beacon spec")
	}

	committees, err := beacon.FetchBeaconCommittee(ctx, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch beacon committees")
	}

	// Validate committees belong to the correct epoch.
	minSlot := phase0.Slot(epoch) * sp.SlotsPerEpoch
	maxSlot := (phase0.Slot(epoch) * sp.SlotsPerEpoch) + sp.SlotsPerEpoch - 1

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(committees))

	for _, committee := range committees {
		if committee.Slot < minSlot || committee.Slot > maxSlot {
			return nil, fmt.Errorf(
				"beacon committee slot outside of epoch. (epoch: %d, slot: %d, min: %d, max: %d)",
				epoch, committee.Slot, minSlot, maxSlot,
			)
		}

		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE)
		if err != nil {
			return nil, err
		}

		validators := make([]*wrapperspb.UInt64Value, 0, len(committee.Validators))
		for _, validator := range committee.Validators {
			validators = append(validators, wrapperspb.UInt64(uint64(validator)))
		}

		event.Data = &xatu.DecoratedEvent_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &xatuethv1.Committee{
				Slot:       wrapperspb.UInt64(uint64(committee.Slot)),
				Index:      wrapperspb.UInt64(uint64(committee.Index)),
				Validators: validators,
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
				StateId: xatuethv1.StateIDFinalized,
				Slot:    builder.BuildSlotV2(uint64(committee.Slot)),
				Epoch:   builder.BuildEpochV2FromSlot(uint64(committee.Slot)),
			},
		}

		events = append(events, event)
	}

	return events, nil
}
