package extractors

import (
	"context"
	"encoding/hex"
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
		Name:           "proposer_duty",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeEpoch,
		EpochProcessor: ProcessProposerDuties,
	})
}

// ProcessProposerDuties fetches and creates events for all proposer duties in an epoch.
func ProcessProposerDuties(
	ctx context.Context,
	epoch phase0.Epoch,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	duties, err := beacon.FetchProposerDuties(ctx, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch proposer duties")
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(duties))

	for _, duty := range duties {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV1ProposerDuty{
			EthV1ProposerDuty: &xatuethv1.ProposerDuty{
				Slot:           wrapperspb.UInt64(uint64(duty.Slot)),
				Pubkey:         fmt.Sprintf("0x%s", hex.EncodeToString(duty.PubKey[:])),
				ValidatorIndex: wrapperspb.UInt64(uint64(duty.ValidatorIndex)),
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1ProposerDuty{
			EthV1ProposerDuty: &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
				StateId: xatuethv1.StateIDFinalized,
				Slot:    builder.BuildSlotV2(uint64(duty.Slot)),
				Epoch:   builder.BuildEpochV2FromSlot(uint64(duty.Slot)),
			},
		}

		events = append(events, event)
	}

	return events, nil
}
