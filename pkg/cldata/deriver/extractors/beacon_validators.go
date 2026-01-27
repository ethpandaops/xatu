package extractors

import (
	"context"
	"fmt"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// DefaultValidatorChunkSize is the default number of validators per event.
const DefaultValidatorChunkSize = 100

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "beacon_validators",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS,
		ActivationFork: spec.DataVersionPhase0,
		Mode:           deriver.ProcessingModeEpoch,
		EpochProcessor: ProcessBeaconValidators,
	})
}

// ProcessBeaconValidators fetches and creates chunked events for all validators in an epoch.
func ProcessBeaconValidators(
	ctx context.Context,
	epoch phase0.Epoch,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	sp, err := beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch spec")
	}

	boundarySlot := phase0.Slot(uint64(epoch) * uint64(sp.SlotsPerEpoch))

	validatorsMap, err := beacon.GetValidators(ctx, xatuethv1.SlotAsString(boundarySlot))
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch validator states")
	}

	// Clean up cache after fetch
	defer beacon.DeleteValidatorsFromCache(xatuethv1.SlotAsString(boundarySlot))

	// Chunk the validators
	chunkSize := DefaultValidatorChunkSize

	var validatorChunks [][]*apiv1.Validator

	currentChunk := make([]*apiv1.Validator, 0, chunkSize)

	for _, validator := range validatorsMap {
		if len(currentChunk) == chunkSize {
			validatorChunks = append(validatorChunks, currentChunk)
			currentChunk = make([]*apiv1.Validator, 0, chunkSize)
		}

		currentChunk = append(currentChunk, validator)
	}

	if len(currentChunk) > 0 {
		validatorChunks = append(validatorChunks, currentChunk)
	}

	builder := deriver.NewEventBuilder(ctxProvider)
	allEvents := make([]*xatu.DecoratedEvent, 0, len(validatorChunks))

	for _, chunk := range validatorChunks {
		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS)
		if err != nil {
			return nil, err
		}

		data := xatu.Validators{}

		for _, validator := range chunk {
			data.Validators = append(data.Validators, &xatuethv1.Validator{
				Index:   wrapperspb.UInt64(uint64(validator.Index)),
				Balance: wrapperspb.UInt64(uint64(validator.Balance)),
				Status:  wrapperspb.String(validator.Status.String()),
				Data: &xatuethv1.ValidatorData{
					Pubkey:                     wrapperspb.String(validator.Validator.PublicKey.String()),
					WithdrawalCredentials:      wrapperspb.String(fmt.Sprintf("%#x", validator.Validator.WithdrawalCredentials)),
					EffectiveBalance:           wrapperspb.UInt64(uint64(validator.Validator.EffectiveBalance)),
					Slashed:                    wrapperspb.Bool(validator.Validator.Slashed),
					ActivationEpoch:            wrapperspb.UInt64(uint64(validator.Validator.ActivationEpoch)),
					ActivationEligibilityEpoch: wrapperspb.UInt64(uint64(validator.Validator.ActivationEligibilityEpoch)),
					ExitEpoch:                  wrapperspb.UInt64(uint64(validator.Validator.ExitEpoch)),
					WithdrawableEpoch:          wrapperspb.UInt64(uint64(validator.Validator.WithdrawableEpoch)),
				},
			})
		}

		event.Data = &xatu.DecoratedEvent_EthV1Validators{
			EthV1Validators: &data,
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1Validators{
			EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
				Epoch: builder.BuildEpochV2(uint64(epoch)),
			},
		}

		allEvents = append(allEvents, event)
	}

	return allEvents, nil
}
