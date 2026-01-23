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
		Name:           "voluntary_exit",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractVoluntaryExits,
	})
}

// ExtractVoluntaryExits extracts voluntary exit events from a beacon block.
func ExtractVoluntaryExits(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	exits, err := block.VoluntaryExits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain voluntary exits")
	}

	if len(exits) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(exits))

	for _, exit := range exits {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &xatuethv1.SignedVoluntaryExitV2{
				Message: &xatuethv1.VoluntaryExitV2{
					Epoch:          wrapperspb.UInt64(uint64(exit.Message.Epoch)),
					ValidatorIndex: wrapperspb.UInt64(uint64(exit.Message.ValidatorIndex)),
				},
				Signature: exit.Signature.String(),
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}
