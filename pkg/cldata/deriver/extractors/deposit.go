package extractors

import (
	"context"
	"fmt"

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
		Name:           "deposit",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractDeposits,
	})
}

// ExtractDeposits extracts deposit events from a beacon block.
func ExtractDeposits(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	deposits, err := block.Deposits()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain deposits")
	}

	if len(deposits) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(deposits))

	for _, deposit := range deposits {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT)
		if err != nil {
			return nil, err
		}

		proof := make([]string, 0, len(deposit.Proof))
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &xatuethv1.DepositV2{
				Proof: proof,
				Data: &xatuethv1.DepositV2_Data{
					Pubkey:                deposit.Data.PublicKey.String(),
					WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
					Amount:                wrapperspb.UInt64(uint64(deposit.Data.Amount)),
					Signature:             deposit.Data.Signature.String(),
				},
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}
