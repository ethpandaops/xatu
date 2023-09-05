package v2

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	DepositDeriverName = "BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT"
)

type DepositDeriver struct {
	log logrus.FieldLogger
}

func NewDepositDeriver(log logrus.FieldLogger) *DepositDeriver {
	return &DepositDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/withdrawal"),
	}
}

func (b *DepositDeriver) Name() string {
	return DepositDeriverName
}

func (b *DepositDeriver) Filter(ctx context.Context) bool {
	return false
}

func (b *DepositDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	deposits, err := b.getDeposits(ctx, block)
	if err != nil {
		return nil, err
	}

	for _, deposit := range deposits {
		event, err := b.createEvent(ctx, metadata, deposit)
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for deposit %s", deposit.String())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *DepositDeriver) getDeposits(ctx context.Context, block *spec.VersionedSignedBeaconBlock) ([]*xatuethv1.DepositV2, error) {
	exits := []*xatuethv1.DepositV2{}

	var deposits []*phase0.Deposit

	switch block.Version {
	case spec.DataVersionPhase0:
		deposits = block.Phase0.Message.Body.Deposits
	case spec.DataVersionAltair:
		deposits = block.Altair.Message.Body.Deposits
	case spec.DataVersionBellatrix:
		deposits = block.Bellatrix.Message.Body.Deposits
	case spec.DataVersionCapella:
		deposits = block.Capella.Message.Body.Deposits
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	for _, deposit := range deposits {
		proof := []string{}
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		exits = append(exits, &xatuethv1.DepositV2{
			Proof: proof,
			Data: &xatuethv1.DepositV2_Data{
				Pubkey:                deposit.Data.PublicKey.String(),
				WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
				Amount:                wrapperspb.UInt64(uint64(deposit.Data.Amount)),
				Signature:             deposit.Data.Signature.String(),
			},
		})
	}

	return exits, nil
}

func (b *DepositDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, deposit *xatuethv1.DepositV2) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: deposit,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockDeposit{
		EthV2BeaconBlockDeposit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockDepositData{
			Block: blockIdentifier,
		},
	}

	return decoratedEvent, nil
}
