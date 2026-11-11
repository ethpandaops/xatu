package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
)

// startMempoolTransactionWatcher starts the mempool transaction watcher on the shared execution client.
func (s *Sentry) startMempoolTransactionWatcher(ctx context.Context, client *execution.Client, metrics *execution.Metrics) (*runningWatcher, error) {
	processTxCallback := func(ctx context.Context, record *execution.PendingTxRecord, txData json.RawMessage) error {
		return s.processMempoolTransaction(ctx, client.GetSigner(), record, txData)
	}

	watcher := execution.NewMempoolWatcher(
		client,
		s.log,
		s.Config.Execution,
		processTxCallback,
		metrics,
	)

	if s.Config.Execution.RemovedTxsEnabled {
		watcher.OnRemovedTransactions(s.processRemovedMempoolTransactions)
	}

	if err := watcher.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start mempool watcher: %w", err)
	}

	return &runningWatcher{
		healthy: watcher.Healthy,
		stop: func() error {
			watcher.Stop()

			return nil
		},
	}, nil
}

// processMempoolTransaction is called when a full transaction is found in txpool_content.
func (s *Sentry) processMempoolTransaction(ctx context.Context, signer types.Signer, record *execution.PendingTxRecord, txData json.RawMessage) error {
	// Get the hash from the record.
	txHash := common.HexToHash(record.Hash)

	// First unmarshal into a map to access the hash field and essential metadata.
	var txMap map[string]any
	if err := json.Unmarshal(txData, &txMap); err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to unmarshal transaction data")

		return err
	}

	// Extract the from address if available.
	var fromAddress string
	if from, ok := txMap["from"].(string); ok {
		fromAddress = from
	}

	// Build out the client meta for the event.
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to create client meta")

		return err
	}

	// Parse the transaction.
	tx, err := parseRawTransactionFromTxPool(txData, signer, txHash)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to parse transaction data")

		return err
	}

	// Pre-populate additional data with the from address from txpool_content.
	// This will be used if signature recovery fails in the MempoolTransaction.Decorate method.
	additionalData := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Hash: record.Hash,
		From: fromAddress,
	}

	meta.AdditionalData = &xatu.ClientMeta_MempoolTransactionV2{
		MempoolTransactionV2: additionalData,
	}

	// Create the mempool transaction event using the ORIGINAL timestamp when the tx was first seen.
	event := execEvent.NewMempoolTransaction(
		s.log,
		tx,
		record.FirstSeen.Add(s.clockDrift),
		s.duplicateCache.MempoolTransaction,
		meta,
		signer,
	)

	// Decorate the event.
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).WithContext(ctx).Error("Failed to decorate event")

		return err
	}

	// Handle the decorated event.
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"tx_hash": record.Hash,
		}).WithContext(ctx).Error("Failed to handle decorated event")

		return err
	}

	// Note: No need to add metrics here anymore as the MempoolWatcher keeps track of metrics directly.
	s.summary.AddEventStreamEvents("mempool_transaction", 1)

	return nil
}

// processRemovedMempoolTransactions emits a MEMPOOL_TRANSACTION_REMOVED event for each transaction that left the
// txpool without being included.
func (s *Sentry) processRemovedMempoolTransactions(ctx context.Context, removed []*execution.RemovedTx) {
	for _, tx := range removed {
		meta, err := s.createNewClientMeta(ctx)
		if err != nil {
			s.log.WithError(err).WithContext(ctx).Error("Failed to create client meta")

			return
		}

		if err := s.handleNewDecoratedEvent(ctx, newMempoolTransactionRemovedEvent(tx, meta, s.clockDrift)); err != nil {
			s.log.WithError(err).WithField("tx_hash", tx.Hash).WithContext(ctx).Error("Failed to handle decorated event")

			continue
		}

		s.summary.AddEventStreamEvents("mempool_transaction_removed", 1)
	}
}

func newMempoolTransactionRemovedEvent(tx *execution.RemovedTx, meta *xatu.ClientMeta, clockDrift time.Duration) *xatu.DecoratedEvent {
	data := &xatu.MempoolTransactionRemoved{
		Hash:      tx.Hash,
		From:      tx.From.Hex(),
		Nonce:     wrapperspb.UInt64(tx.Nonce),
		Type:      wrapperspb.UInt32(uint32(tx.Type)),
		Gas:       wrapperspb.UInt64(tx.Gas),
		GasTipCap: tx.GasTipCap.String(),
		GasFeeCap: tx.GasFeeCap.String(),
		Reason:    tx.Reason,
		FirstSeen: timestamppb.New(tx.FirstSeen.Add(clockDrift)),
		LastSeen:  timestamppb.New(tx.LastSeen.Add(clockDrift)),
		Pool:      tx.Pool,
	}

	if tx.To != nil {
		data.To = tx.To.Hex()
	}

	if tx.BlobGasFeeCap != nil {
		data.BlobGasFeeCap = tx.BlobGasFeeCap.String()
	}

	if tx.Replacement != nil {
		data.ReplacementHash = tx.Replacement.Hash
		data.ReplacementGasTipCap = tx.Replacement.GasTipCap.String()
		data.ReplacementGasFeeCap = tx.Replacement.GasFeeCap.String()
	}

	if tx.AccountNonce != nil {
		data.AccountNonce = wrapperspb.UInt64(*tx.AccountNonce)
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_TRANSACTION_REMOVED,
			DateTime: timestamppb.New(tx.DetectedAt.Add(clockDrift)),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: meta},
		Data: &xatu.DecoratedEvent_MempoolTransactionRemoved{MempoolTransactionRemoved: data},
	}
}

// parseRawTransactionFromTxPool parses a transaction from txpool_content or eth_getTransactionByHash JSON. It decodes
// the signed transaction, so its hash, type, size and blob fields are the real ones, and falls back to rebuilding an
// unsigned transaction from the basic fields when the client's JSON does not decode.
func parseRawTransactionFromTxPool(txData json.RawMessage, signer types.Signer, expectedHash common.Hash) (*types.Transaction, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalJSON(txData); err == nil {
		return tx, nil
	}

	var txMap map[string]any
	if err := json.Unmarshal(txData, &txMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction data: %w", err)
	}

	// First, try to create an exact transaction with matching hash.
	if txType, ok := txMap["type"].(string); ok {
		txTypeVal, err := hexutil.DecodeUint64(txType)
		if err == nil {
			switch txTypeVal {
			case 0:
				return parseTypeLegacyTx(txMap, expectedHash)
			case 1:
				return parseTypeAccessListTx(txMap, expectedHash)
			case 2:
				return parseTypeDynamicFeeTx(txMap, expectedHash)
			case 3:
				return parseTypeBlobTx(txMap, expectedHash)
			}
		}
	}

	// Fallback to creating a simple transaction with basic fields.
	return createBasicTransaction(txMap, signer, expectedHash)
}

// createBasicTransaction creates a basic transaction with essential fields.
func createBasicTransaction(txMap map[string]any, signer types.Signer, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
	)

	// Parse basic fields.
	if nonceHex, ok := txMap["nonce"].(string); ok {
		nonceVal, err := hexutil.DecodeUint64(nonceHex)
		if err == nil {
			nonce = nonceVal
		}
	}

	if gasHex, ok := txMap["gas"].(string); ok {
		gasVal, err := hexutil.DecodeUint64(gasHex)
		if err == nil {
			gasLimit = gasVal
		}
	}

	if gasPriceHex, ok := txMap["gasPrice"].(string); ok {
		gpVal, err := hexutil.DecodeBig(gasPriceHex)
		if err == nil {
			gasPrice = gpVal
		} else {
			gasPrice = big.NewInt(0)
		}
	} else {
		gasPrice = big.NewInt(0)
	}

	if valueHex, ok := txMap["value"].(string); ok {
		valBig, err := hexutil.DecodeBig(valueHex)
		if err == nil {
			value = valBig
		} else {
			value = big.NewInt(0)
		}
	} else {
		value = big.NewInt(0)
	}

	if inputHex, ok := txMap["input"].(string); ok {
		inputData, err := hexutil.Decode(inputHex)
		if err == nil {
			data = inputData
		}
	}

	if toStr, ok := txMap["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	// Create a legacy transaction
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       to,
		Value:    value,
		Data:     data,
	})

	return tx, nil
}

// parseTypeLegacyTx attempts to parse a legacy transaction (type 0).
func parseTypeLegacyTx(txMap map[string]any, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
	)

	// Parse transaction fields.
	if nonceHex, ok := txMap["nonce"].(string); ok {
		nonceVal, err := hexutil.DecodeUint64(nonceHex)
		if err == nil {
			nonce = nonceVal
		}
	}

	if gasHex, ok := txMap["gas"].(string); ok {
		gasVal, err := hexutil.DecodeUint64(gasHex)
		if err == nil {
			gasLimit = gasVal
		}
	}

	if gasPriceHex, ok := txMap["gasPrice"].(string); ok {
		gpVal, err := hexutil.DecodeBig(gasPriceHex)
		if err == nil {
			gasPrice = gpVal
		} else {
			gasPrice = big.NewInt(0)
		}
	} else {
		gasPrice = big.NewInt(0)
	}

	if valueHex, ok := txMap["value"].(string); ok {
		valBig, err := hexutil.DecodeBig(valueHex)
		if err == nil {
			value = valBig
		} else {
			value = big.NewInt(0)
		}
	} else {
		value = big.NewInt(0)
	}

	if inputHex, ok := txMap["input"].(string); ok {
		inputData, err := hexutil.Decode(inputHex)
		if err == nil {
			data = inputData
		}
	}

	if toStr, ok := txMap["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	// Create a legacy transaction with the parsed data.
	legacyTx := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       to,
		Value:    value,
		Data:     data,
	}

	return types.NewTx(legacyTx), nil
}

// parseTypeAccessListTx attempts to parse an access list transaction (type 1).
func parseTypeAccessListTx(txMap map[string]any, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
		chainID  *big.Int
	)

	// Parse fields (similar to legacy but with chainID and accessList).
	if nonceHex, ok := txMap["nonce"].(string); ok {
		nonceVal, err := hexutil.DecodeUint64(nonceHex)
		if err == nil {
			nonce = nonceVal
		}
	}

	if gasHex, ok := txMap["gas"].(string); ok {
		gasVal, err := hexutil.DecodeUint64(gasHex)
		if err == nil {
			gasLimit = gasVal
		}
	}

	if gasPriceHex, ok := txMap["gasPrice"].(string); ok {
		gpVal, err := hexutil.DecodeBig(gasPriceHex)
		if err == nil {
			gasPrice = gpVal
		} else {
			gasPrice = big.NewInt(0)
		}
	} else {
		gasPrice = big.NewInt(0)
	}

	if valueHex, ok := txMap["value"].(string); ok {
		valBig, err := hexutil.DecodeBig(valueHex)
		if err == nil {
			value = valBig
		} else {
			value = big.NewInt(0)
		}
	} else {
		value = big.NewInt(0)
	}

	if inputHex, ok := txMap["input"].(string); ok {
		inputData, err := hexutil.Decode(inputHex)
		if err == nil {
			data = inputData
		}
	}

	if toStr, ok := txMap["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	if chainIDHex, ok := txMap["chainId"].(string); ok {
		id, err := hexutil.DecodeBig(chainIDHex)
		if err == nil {
			chainID = id
		} else {
			chainID = big.NewInt(1)
		}
	} else {
		chainID = big.NewInt(1)
	}

	// Create the access list transaction.
	accessListTx := &types.AccessListTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasPrice:   gasPrice,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{},
	}

	return types.NewTx(accessListTx), nil
}

// parseTypeDynamicFeeTx attempts to parse a dynamic fee transaction (type 2, EIP-1559).
func parseTypeDynamicFeeTx(txMap map[string]any, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce          uint64
		gasLimit       uint64
		maxFeePerGas   *big.Int
		maxPriorityFee *big.Int
		value          *big.Int
		data           []byte
		to             *common.Address
		chainID        *big.Int
	)

	// Parse fields specific to dynamic fee transactions.
	if nonceHex, ok := txMap["nonce"].(string); ok {
		nonceVal, err := hexutil.DecodeUint64(nonceHex)
		if err == nil {
			nonce = nonceVal
		}
	}

	if gasHex, ok := txMap["gas"].(string); ok {
		gasVal, err := hexutil.DecodeUint64(gasHex)
		if err == nil {
			gasLimit = gasVal
		}
	}

	if maxFeeHex, ok := txMap["maxFeePerGas"].(string); ok {
		mfVal, err := hexutil.DecodeBig(maxFeeHex)
		if err == nil {
			maxFeePerGas = mfVal
		} else {
			maxFeePerGas = big.NewInt(0)
		}
	} else {
		// Fallback to gasPrice if maxFeePerGas is not available.
		if gasPriceHex, ok := txMap["gasPrice"].(string); ok {
			gpVal, err := hexutil.DecodeBig(gasPriceHex)
			if err == nil {
				maxFeePerGas = gpVal
			} else {
				maxFeePerGas = big.NewInt(0)
			}
		} else {
			maxFeePerGas = big.NewInt(0)
		}
	}

	if maxPrioHex, ok := txMap["maxPriorityFeePerGas"].(string); ok {
		mpVal, err := hexutil.DecodeBig(maxPrioHex)
		if err == nil {
			maxPriorityFee = mpVal
		} else {
			maxPriorityFee = big.NewInt(0)
		}
	} else {
		// Default to maxFeePerGas or a lower value.
		maxPriorityFee = maxFeePerGas
	}

	if valueHex, ok := txMap["value"].(string); ok {
		valBig, err := hexutil.DecodeBig(valueHex)
		if err == nil {
			value = valBig
		} else {
			value = big.NewInt(0)
		}
	} else {
		value = big.NewInt(0)
	}

	if inputHex, ok := txMap["input"].(string); ok {
		inputData, err := hexutil.Decode(inputHex)
		if err == nil {
			data = inputData
		}
	}

	if toStr, ok := txMap["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	if chainIDHex, ok := txMap["chainId"].(string); ok {
		id, err := hexutil.DecodeBig(chainIDHex)
		if err == nil {
			chainID = id
		} else {
			chainID = big.NewInt(1)
		}
	} else {
		chainID = big.NewInt(1)
	}

	// Create the dynamic fee transaction.
	dynamicFeeTx := &types.DynamicFeeTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasTipCap:  maxPriorityFee,
		GasFeeCap:  maxFeePerGas,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{},
	}

	return types.NewTx(dynamicFeeTx), nil
}

// parseTypeBlobTx attempts to parse a blob transaction (type 3, EIP-4844).
func parseTypeBlobTx(txMap map[string]any, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce          uint64
		gasLimit       uint64
		maxFeePerGas   *big.Int
		maxPriorityFee *big.Int
		value          *big.Int
		data           []byte
		to             *common.Address
		chainID        *big.Int
	)

	// Parse fields.
	if nonceHex, ok := txMap["nonce"].(string); ok {
		nonceVal, err := hexutil.DecodeUint64(nonceHex)
		if err == nil {
			nonce = nonceVal
		}
	}

	if gasHex, ok := txMap["gas"].(string); ok {
		gasVal, err := hexutil.DecodeUint64(gasHex)
		if err == nil {
			gasLimit = gasVal
		}
	}

	if maxFeeHex, ok := txMap["maxFeePerGas"].(string); ok {
		mfVal, err := hexutil.DecodeBig(maxFeeHex)
		if err == nil {
			maxFeePerGas = mfVal
		} else {
			maxFeePerGas = big.NewInt(0)
		}
	} else {
		// Fallback to gasPrice
		if gasPriceHex, ok := txMap["gasPrice"].(string); ok {
			gpVal, err := hexutil.DecodeBig(gasPriceHex)
			if err == nil {
				maxFeePerGas = gpVal
			} else {
				maxFeePerGas = big.NewInt(0)
			}
		} else {
			maxFeePerGas = big.NewInt(0)
		}
	}

	if maxPrioHex, ok := txMap["maxPriorityFeePerGas"].(string); ok {
		mpVal, err := hexutil.DecodeBig(maxPrioHex)
		if err == nil {
			maxPriorityFee = mpVal
		} else {
			maxPriorityFee = big.NewInt(0)
		}
	} else {
		// Default to maxFeePerGas.
		maxPriorityFee = maxFeePerGas
	}

	if valueHex, ok := txMap["value"].(string); ok {
		valBig, err := hexutil.DecodeBig(valueHex)
		if err == nil {
			value = valBig
		} else {
			value = big.NewInt(0)
		}
	} else {
		value = big.NewInt(0)
	}

	if inputHex, ok := txMap["input"].(string); ok {
		inputData, err := hexutil.Decode(inputHex)
		if err == nil {
			data = inputData
		}
	}

	if toStr, ok := txMap["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		to = &addr
	}

	if chainIDHex, ok := txMap["chainId"].(string); ok {
		id, err := hexutil.DecodeBig(chainIDHex)
		if err == nil {
			chainID = id
		} else {
			chainID = big.NewInt(1)
		}
	} else {
		chainID = big.NewInt(1)
	}

	// TODO: Parse blob data.
	dynamicFeeTx := &types.DynamicFeeTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasTipCap:  maxPriorityFee,
		GasFeeCap:  maxFeePerGas,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{},
	}

	return types.NewTx(dynamicFeeTx), nil
}
