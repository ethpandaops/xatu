package v2

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethereum/go-ethereum/core/types"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type ExecutionTransactionDeriver struct {
	log logrus.FieldLogger
}

const (
	ExecutionTransactionDeriverName = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION"
)

func NewExecutionTransactionDeriver(log logrus.FieldLogger) *ExecutionTransactionDeriver {
	return &ExecutionTransactionDeriver{
		log: log.WithField("module", "cannon/event/beacon/eth/v2/execution_transaction"),
	}
}

func (b *ExecutionTransactionDeriver) Name() string {
	return ExecutionTransactionDeriverName
}

func (b *ExecutionTransactionDeriver) Filter(ctx context.Context) bool {
	return false
}

func (b *ExecutionTransactionDeriver) Process(ctx context.Context, metadata *BeaconBlockMetadata, block *spec.VersionedSignedBeaconBlock) ([]*xatu.DecoratedEvent, error) {
	events := []*xatu.DecoratedEvent{}

	transactions, err := b.getExecutionTransactions(ctx, block, metadata)
	if err != nil {
		return nil, err
	}

	for index, transaction := range transactions {
		from, err := types.Sender(types.LatestSignerForChainID(transaction.ChainId()), transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction sender: %v", err)
		}

		gasPrice := transaction.GasPrice()
		if gasPrice == nil {
			return nil, fmt.Errorf("failed to get transaction gas price")
		}

		value := transaction.Value()
		if value == nil {
			return nil, fmt.Errorf("failed to get transaction value")
		}

		to := ""

		if transaction.To() != nil {
			to = transaction.To().Hex()
		}

		chainID := transaction.ChainId()
		if chainID == nil {
			return nil, fmt.Errorf("failed to get transaction chain ID")
		}

		tx := &xatuethv1.Transaction{
			Nonce:    wrapperspb.UInt64(transaction.Nonce()),
			GasPrice: gasPrice.String(),
			Gas:      wrapperspb.UInt64(transaction.Gas()),
			To:       to,
			From:     from.Hex(),
			Value:    value.String(),
			Input:    hex.EncodeToString(transaction.Data()),
			Hash:     transaction.Hash().Hex(),
			ChainId:  chainID.String(),
			Type:     wrapperspb.UInt32(uint32(transaction.Type())),
		}

		event, err := b.createEvent(ctx, metadata, tx, uint64(index))
		if err != nil {
			b.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for execution transaction %s", transaction.Hash())
		}

		events = append(events, event)
	}

	return events, nil
}

func (b *ExecutionTransactionDeriver) getExecutionTransactions(ctx context.Context, block *spec.VersionedSignedBeaconBlock, metadata *BeaconBlockMetadata) ([]*types.Transaction, error) {
	transactions := []*types.Transaction{}

	switch block.Version {
	case spec.DataVersionPhase0:
		return transactions, nil
	case spec.DataVersionAltair:
		return transactions, nil
	case spec.DataVersionBellatrix:
		for _, transaction := range block.Bellatrix.Message.Body.ExecutionPayload.Transactions {
			ethTransaction := new(types.Transaction)
			if err := ethTransaction.UnmarshalBinary(transaction); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
			}

			transactions = append(transactions, ethTransaction)
		}
	case spec.DataVersionCapella:
		for _, transaction := range block.Capella.Message.Body.ExecutionPayload.Transactions {
			ethTransaction := new(types.Transaction)
			if err := ethTransaction.UnmarshalBinary(transaction); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
			}

			transactions = append(transactions, ethTransaction)
		}
	default:
		return nil, fmt.Errorf("unsupported block version: %s", block.Version.String())
	}

	return transactions, nil
}

func (b *ExecutionTransactionDeriver) createEvent(ctx context.Context, metadata *BeaconBlockMetadata, transaction *xatuethv1.Transaction, positionInBlock uint64) (*xatu.DecoratedEvent, error) {
	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			DateTime: timestamppb.New(metadata.Now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata.ClientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: transaction,
		},
	}

	blockIdentifier, err := metadata.BlockIdentifier()
	if err != nil {
		return nil, err
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
		EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
			Block:           blockIdentifier,
			PositionInBlock: wrapperspb.UInt64(positionInBlock),
		},
	}

	return decoratedEvent, nil
}
