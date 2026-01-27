package extractors

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/deriver"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func init() {
	deriver.Register(&deriver.DeriverSpec{
		Name:           "execution_transaction",
		CannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
		ActivationFork: spec.DataVersionBellatrix,
		Mode:           deriver.ProcessingModeSlot,
		BlockExtractor: ExtractExecutionTransactions,
	})
}

// ExtractExecutionTransactions extracts execution transaction events from a beacon block.
func ExtractExecutionTransactions(
	ctx context.Context,
	block *spec.VersionedSignedBeaconBlock,
	blockID *xatu.BlockIdentifier,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) ([]*xatu.DecoratedEvent, error) {
	log := logrus.WithField("extractor", "execution_transaction")

	// Fetch blob sidecars for Deneb+ blocks
	blobSidecars := []*deneb.BlobSidecar{}

	slot, err := block.Slot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block slot")
	}

	if block.Version >= spec.DataVersionDeneb {
		sidecars, fetchErr := beacon.FetchBeaconBlockBlobs(ctx, xatuethv1.SlotAsString(slot))
		if fetchErr != nil {
			var apiErr *api.Error
			if errors.As(fetchErr, &apiErr) {
				switch apiErr.StatusCode {
				case 404:
					log.WithField("slot", slot).Debug("no beacon block blob sidecars found for slot")
				case 503:
					return nil, errors.New("beacon node is syncing")
				default:
					return nil, errors.Wrapf(err, "failed to get beacon block blob sidecars for slot %d", slot)
				}
			} else {
				return nil, errors.Wrapf(err, "failed to get beacon block blob sidecars for slot %d", slot)
			}
		} else {
			blobSidecars = sidecars
		}
	}

	blobSidecarsMap := make(map[string]*deneb.BlobSidecar, len(blobSidecars))

	for _, blobSidecar := range blobSidecars {
		versionedHash := cldata.ConvertKzgCommitmentToVersionedHash(blobSidecar.KZGCommitment[:])
		blobSidecarsMap[versionedHash.String()] = blobSidecar
	}

	// Get execution transactions
	txBytes, err := block.ExecutionTransactions()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution transactions: %w", err)
	}

	transactions := make([]*types.Transaction, 0, len(txBytes))

	for _, txData := range txBytes {
		ethTransaction := new(types.Transaction)
		if err := ethTransaction.UnmarshalBinary(txData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
		}

		transactions = append(transactions, ethTransaction)
	}

	chainID := new(big.Int).SetUint64(ctxProvider.DepositChainID())
	if chainID.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("failed to get chain ID from context provider")
	}

	signer := types.LatestSignerForChainID(chainID)
	builder := deriver.NewEventBuilder(ctxProvider)
	events := make([]*xatu.DecoratedEvent, 0, len(transactions))

	for index, transaction := range transactions {
		from, err := types.Sender(signer, transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction sender: %w", err)
		}

		gasPrice, err := getGasPrice(block, transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction gas price: %w", err)
		}

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

		tx := &xatuethv1.Transaction{
			Nonce:     wrapperspb.UInt64(transaction.Nonce()),
			Gas:       wrapperspb.UInt64(transaction.Gas()),
			GasPrice:  gasPrice.String(),
			GasTipCap: transaction.GasTipCap().String(),
			GasFeeCap: transaction.GasFeeCap().String(),
			To:        to,
			From:      from.Hex(),
			Value:     value.String(),
			Input:     hex.EncodeToString(transaction.Data()),
			Hash:      transaction.Hash().Hex(),
			ChainId:   chainID.String(),
			Type:      wrapperspb.UInt32(uint32(transaction.Type())),
		}

		sidecarsEmptySize := 0
		sidecarsSize := 0

		if transaction.Type() == 3 {
			blobHashes := make([]string, len(transaction.BlobHashes()))

			if len(transaction.BlobHashes()) == 0 {
				log.WithField("transaction", transaction.Hash().Hex()).Warn("no versioned hashes for type 3 transaction")
			}

			for i := 0; i < len(transaction.BlobHashes()); i++ {
				hash := transaction.BlobHashes()[i]
				blobHashes[i] = hash.String()
				sidecar := blobSidecarsMap[hash.String()]

				if sidecar != nil {
					sidecarsSize += len(sidecar.Blob)
					sidecarsEmptySize += cldata.CountConsecutiveEmptyBytes(sidecar.Blob[:], 4)
				} else {
					log.WithField("versioned hash", hash.String()).WithField("transaction", transaction.Hash().Hex()).Warn("missing blob sidecar")
				}
			}

			tx.BlobGas = wrapperspb.UInt64(transaction.BlobGas())
			tx.BlobGasFeeCap = transaction.BlobGasFeeCap().String()
			tx.BlobHashes = blobHashes
		}

		event, err := builder.CreateDecoratedEvent(ctx, xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION)
		if err != nil {
			return nil, err
		}

		event.Data = &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: tx,
		}

		event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
				Block: blockID,
				//nolint:gosec // index from range is bounded by block transaction limit
				PositionInBlock:       wrapperspb.UInt64(uint64(index)),
				Size:                  strconv.FormatFloat(float64(transaction.Size()), 'f', 0, 64),
				CallDataSize:          fmt.Sprintf("%d", len(transaction.Data())),
				BlobSidecarsSize:      fmt.Sprint(sidecarsSize),
				BlobSidecarsEmptySize: fmt.Sprint(sidecarsEmptySize),
			},
		}

		events = append(events, event)
	}

	return events, nil
}

// getGasPrice calculates the effective gas price for a transaction based on its type and block version.
func getGasPrice(block *spec.VersionedSignedBeaconBlock, transaction *types.Transaction) (*big.Int, error) {
	if transaction.Type() == 0 || transaction.Type() == 1 {
		return transaction.GasPrice(), nil
	}

	if transaction.Type() == 2 || transaction.Type() == 3 || transaction.Type() == 4 { // EIP-1559/blob/7702 transactions
		baseFee := new(big.Int)

		switch block.Version {
		case spec.DataVersionBellatrix:
			baseFee = new(big.Int).SetBytes(block.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[:])
		case spec.DataVersionCapella:
			baseFee = new(big.Int).SetBytes(block.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:])
		case spec.DataVersionDeneb:
			executionPayload := block.Deneb.Message.Body.ExecutionPayload
			baseFee.SetBytes(executionPayload.BaseFeePerGas.Bytes())
		case spec.DataVersionElectra:
			executionPayload := block.Electra.Message.Body.ExecutionPayload
			baseFee.SetBytes(executionPayload.BaseFeePerGas.Bytes())
		case spec.DataVersionFulu:
			executionPayload := block.Fulu.Message.Body.ExecutionPayload
			baseFee.SetBytes(executionPayload.BaseFeePerGas.Bytes())
		default:
			return nil, fmt.Errorf("unknown block version: %d", block.Version)
		}

		// Calculate Effective Gas Price: min(max_fee_per_gas, base_fee + max_priority_fee_per_gas)
		gasPrice := new(big.Int).Add(baseFee, transaction.GasTipCap())
		if gasPrice.Cmp(transaction.GasFeeCap()) > 0 {
			gasPrice = transaction.GasFeeCap()
		}

		return gasPrice, nil
	}

	return nil, fmt.Errorf("unknown transaction type: %d", transaction.Type())
}
