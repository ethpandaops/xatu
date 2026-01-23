package extractors

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "bls_to_execution_change",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
		ActivationFork: spec.DataVersionCapella,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractBLSToExecutionChanges,
	})
}

// ExtractBLSToExecutionChanges extracts BLS to execution change events from a beacon block.
func ExtractBLSToExecutionChanges(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	_ cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	changes, err := block.BLSToExecutionChanges()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain BLS to execution changes")
	}

	if len(changes) == 0 {
		return []*xatu.DecoratedEvent{}, nil
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(changes))

	for _, change := range changes {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &xatuethv2.SignedBLSToExecutionChangeV2{
				Message: &xatuethv2.BLSToExecutionChangeV2{
					ValidatorIndex:     wrapperspb.UInt64(uint64(change.Message.ValidatorIndex)),
					FromBlsPubkey:      change.Message.FromBLSPubkey.String(),
					ToExecutionAddress: change.Message.ToExecutionAddress.String(),
				},
				Signature: change.Signature.String(),
			},
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &xatu.ClientMeta_AdditionalEthV2BeaconBlockBLSToExecutionChangeData{
				Block: blockID,
			},
		}

		events = append(events, event)
	}

	return events, nil
}
