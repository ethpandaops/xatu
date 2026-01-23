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
		Name:           "withdrawal",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		ActivationFork: spec.DataVersionCapella,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractWithdrawals,
	})
}

// ExtractWithdrawals extracts withdrawal events from a beacon block.
func ExtractWithdrawals(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	withdrawals, err := block.Withdrawals()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain withdrawals")
	}

	if len(withdrawals) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(withdrawals))

	for _, withdrawal := range withdrawals {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: &xatuethv1.WithdrawalV2{
				Index:          &wrapperspb.UInt64Value{Value: uint64(withdrawal.Index)},
				ValidatorIndex: &wrapperspb.UInt64Value{Value: uint64(withdrawal.ValidatorIndex)},
				Address:        withdrawal.Address.String(),
				Amount:         &wrapperspb.UInt64Value{Value: uint64(withdrawal.Amount)},
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: &xatu.ClientMeta_AdditionalEthV2BeaconBlockWithdrawalData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}
