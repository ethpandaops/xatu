package extractors

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "proposer_slashing",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractProposerSlashings,
	})
}

// ExtractProposerSlashings extracts proposer slashing events from a beacon block.
func ExtractProposerSlashings(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	slashings, err := block.ProposerSlashings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain proposer slashings")
	}

	if len(slashings) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(slashings))

	for _, slashing := range slashings {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &xatuethv1.ProposerSlashingV2{
				SignedHeader_1: &xatuethv1.SignedBeaconBlockHeaderV2{
					Message: &xatuethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.Slot)),
						ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader1.Message.ProposerIndex)),
						ParentRoot:    xatuethv1.RootAsString(slashing.SignedHeader1.Message.ParentRoot),
						StateRoot:     xatuethv1.RootAsString(slashing.SignedHeader1.Message.StateRoot),
						BodyRoot:      xatuethv1.RootAsString(slashing.SignedHeader1.Message.BodyRoot),
					},
					Signature: slashing.SignedHeader1.Signature.String(),
				},
				SignedHeader_2: &xatuethv1.SignedBeaconBlockHeaderV2{
					Message: &xatuethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.Slot)),
						ProposerIndex: wrapperspb.UInt64(uint64(slashing.SignedHeader2.Message.ProposerIndex)),
						ParentRoot:    xatuethv1.RootAsString(slashing.SignedHeader2.Message.ParentRoot),
						StateRoot:     xatuethv1.RootAsString(slashing.SignedHeader2.Message.StateRoot),
						BodyRoot:      xatuethv1.RootAsString(slashing.SignedHeader2.Message.BodyRoot),
					},
					Signature: slashing.SignedHeader2.Signature.String(),
				},
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}
