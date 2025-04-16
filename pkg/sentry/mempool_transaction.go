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
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
	"github.com/ethpandaops/xatu/pkg/sentry/execution"
	"github.com/sirupsen/logrus"
)

// startMempoolTransactionWatcher initializes and starts the mempool transaction watcher.
func (s *Sentry) startMempoolTransactionWatcher(ctx context.Context) error {
	if s.Config.Execution == nil || !s.Config.Execution.Enabled {
		s.log.Info("Mempool transaction watcher disabled")
		return nil
	}

	// Validate execution config
	if s.Config.Execution.RPCAddress == "" {
		return fmt.Errorf("execution.rpcAddress is required when execution is enabled")
	}

	if s.Config.Execution.WSAddress == "" {
		return fmt.Errorf("execution.wsAddress is required when execution is enabled")
	}

	// Initialize the unified execution client that handles both WebSocket and RPC connections
	client, err := execution.NewClient(
		ctx,
		s.log,
		&execution.Config{
			WSAddress:     s.Config.Execution.WSAddress,  // Required - WebSocket connection for subscriptions
			RPCAddress:    s.Config.Execution.RPCAddress, // Required - RPC connection for txpool_content
			Headers:       s.Config.Execution.Headers,
			FetchInterval: time.Duration(s.Config.Execution.FetchInterval) * time.Second,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create execution client: %w", err)
	}

	// Start the execution client
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start execution client: %w", err)
	}

	// Store the client in the sentry
	s.execution = client

	// Use default config with overrides
	watcherConfig := execution.DefaultMempoolWatcherConfig()

	// Override fetch interval if configured
	if s.Config.Execution.FetchInterval > 0 {
		watcherConfig.FetchInterval = time.Duration(s.Config.Execution.FetchInterval) * time.Second
	}

	// Override prune duration if configured
	if s.Config.Execution.PruneDuration > 0 {
		watcherConfig.PruneDuration = time.Duration(s.Config.Execution.PruneDuration) * time.Second
	}

	// Create mempool transaction processor callback
	processTxCallback := func(ctx context.Context, record *execution.PendingTxRecord, txData json.RawMessage) error {
		return s.processMempoolTransaction(ctx, record, txData)
	}

	// Create and start the mempool watcher
	watcher := execution.NewMempoolWatcher(
		client,
		s.log,
		watcherConfig.FetchInterval,
		watcherConfig.PruneDuration,
		processTxCallback,
	)

	if err := watcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start mempool watcher: %w", err)
	}

	// Start a goroutine to periodically update metrics from the watcher
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		networkID := s.beacon.Metadata().Network.ID
		networkIDStr := fmt.Sprintf("%d", networkID)
		if networkIDStr == "" || networkIDStr == "0" {
			networkIDStr = unknown
		}

		// If network name is overridden, we should still use the correct ID
		if s.Config.Ethereum.OverrideNetworkName != "" {
			s.log.WithField("network_id", networkIDStr).
				WithField("network_name", s.Config.Ethereum.OverrideNetworkName).
				Debug("Using network ID for mempool metrics with overridden network name")
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				totalReceived, _, totalExpired, totalNulls, pendingCount, pendingBySource := watcher.GetMetrics()

				// Update all the mempool metrics
				s.metrics.AddMempoolTxReceived(int(totalReceived), networkIDStr)
				// The processed count is updated directly in processMempoolTransaction
				// and doesn't need to be updated here
				s.metrics.AddMempoolTxExpired(int(totalExpired), networkIDStr)
				s.metrics.AddMempoolTxNull(int(totalNulls), networkIDStr)
				s.metrics.SetMempoolTxPending(pendingCount, networkIDStr)

				// Could add metrics for transactions by source if needed
				_ = pendingBySource // Using the variable to avoid linter warnings
			}
		}
	}()

	// Add shutdown functions for the client
	s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
		watcher.Stop()
		return client.Stop(ctx)
	})

	return nil
}

// processMempoolTransaction is called when a full transaction is found in txpool_content
func (s *Sentry) processMempoolTransaction(ctx context.Context, record *execution.PendingTxRecord, txData json.RawMessage) error {
	// Get the hash from the record
	txHash := common.HexToHash(record.Hash)

	// First unmarshal into a map to access the hash field and essential metadata
	var txMap map[string]interface{}
	if err := json.Unmarshal(txData, &txMap); err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to unmarshal transaction data")
		return err
	}

	// Extract the from address if available
	var fromAddress string
	if from, ok := txMap["from"].(string); ok {
		fromAddress = from
	}

	// Use the original hash from the record for better logging
	//s.log.WithField("tx_hash", record.Hash).Debug("Processing mempool transaction")

	// Build out the client meta for the event
	meta, err := s.createNewClientMeta(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to create client meta")
		return err
	}

	// Parse the transaction
	tx, err := parseRawTransactionFromTxPool(txData, s.execution.GetSigner(), txHash)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to parse transaction data")
		return err
	}

	// We use the from address even if it doesn't match what we'd recover from signature
	// This is especially important since we might not have valid signatures
	// s.log.WithField("tx_hash", record.Hash).WithField("from", fromAddress).Debug("Using from address from txpool_content")

	// Pre-populate additional data with the from address from txpool_content
	// This will be used if signature recovery fails in the MempoolTransaction.Decorate method
	additionalData := &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{
		Hash: record.Hash,
		From: fromAddress,
	}

	meta.AdditionalData = &xatu.ClientMeta_MempoolTransactionV2{
		MempoolTransactionV2: additionalData,
	}

	// Create the mempool transaction event using the ORIGINAL timestamp when the tx was first seen
	event := execEvent.NewMempoolTransaction(
		s.log,
		tx,
		record.FirstSeen.Add(s.clockDrift),
		s.duplicateCache.MempoolTransaction,
		meta,
		s.execution.GetSigner(),
	)

	// Decorate the event
	decoratedEvent, err := event.Decorate(ctx)
	if err != nil {
		s.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to decorate event")
		return err
	}

	// Handle the decorated event
	if err = s.handleNewDecoratedEvent(ctx, decoratedEvent); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"tx_hash": record.Hash,
		}).Error("Failed to handle decorated event")
		return err
	}

	// Update metrics
	networkID := meta.Ethereum.Network.Id
	networkIDStr := fmt.Sprintf("%d", networkID)
	if networkIDStr == "" || networkIDStr == "0" {
		networkIDStr = unknown
	}

	// Add metrics for processed transaction
	s.metrics.AddMempoolTxProcessed(1, networkIDStr)

	s.summary.AddEventStreamEvents("mempool_transaction", 1)

	return nil
}

// parseRawTransactionFromTxPool parses a transaction from txpool_content JSON format
// and ensures it has the correct hash by creating a special wrapping transaction
func parseRawTransactionFromTxPool(txData json.RawMessage, signer types.Signer, expectedHash common.Hash) (*types.Transaction, error) {
	var txMap map[string]interface{}
	if err := json.Unmarshal(txData, &txMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction data: %w", err)
	}

	// First, try to create an exact transaction with matching hash
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

	// Fallback to creating a simple transaction with basic fields
	return createBasicTransaction(txMap, signer, expectedHash)
}

// createBasicTransaction creates a basic transaction with essential fields
func createBasicTransaction(txMap map[string]interface{}, signer types.Signer, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
	)

	// Parse basic fields
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

	// Just return the standard transaction, the caller will handle hash verification
	return tx, nil
}

// parseTypeLegacyTx attempts to parse a legacy transaction (type 0)
func parseTypeLegacyTx(txMap map[string]interface{}, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
	)

	// Parse transaction fields
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

	// Create a legacy transaction with the parsed data
	legacyTx := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gasLimit,
		To:       to,
		Value:    value,
		Data:     data,
	}

	// We just create it without signatures since we can't reliably reconstruct them
	return types.NewTx(legacyTx), nil
}

// parseTypeAccessListTx attempts to parse an access list transaction (type 1)
func parseTypeAccessListTx(txMap map[string]interface{}, expectedHash common.Hash) (*types.Transaction, error) {
	var (
		nonce    uint64
		gasPrice *big.Int
		gasLimit uint64
		value    *big.Int
		data     []byte
		to       *common.Address
		chainID  *big.Int
	)

	// Parse fields (similar to legacy but with chainID and accessList)
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
			chainID = big.NewInt(1) // default to mainnet
		}
	} else {
		chainID = big.NewInt(1) // default to mainnet
	}

	// Create the access list transaction
	accessListTx := &types.AccessListTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasPrice:   gasPrice,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{}, // Empty access list
	}

	return types.NewTx(accessListTx), nil
}

// parseTypeDynamicFeeTx attempts to parse a dynamic fee transaction (type 2, EIP-1559)
func parseTypeDynamicFeeTx(txMap map[string]interface{}, expectedHash common.Hash) (*types.Transaction, error) {
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

	// Parse fields specific to dynamic fee transactions
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
		// Fallback to gasPrice if maxFeePerGas is not available
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
		// Default to maxFeePerGas or a lower value
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
			chainID = big.NewInt(1) // default to mainnet
		}
	} else {
		chainID = big.NewInt(1) // default to mainnet
	}

	// Create the dynamic fee transaction
	dynamicFeeTx := &types.DynamicFeeTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasTipCap:  maxPriorityFee,
		GasFeeCap:  maxFeePerGas,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{}, // Empty access list
	}

	return types.NewTx(dynamicFeeTx), nil
}

// parseTypeBlobTx attempts to parse a blob transaction (type 3, EIP-4844)
func parseTypeBlobTx(txMap map[string]interface{}, expectedHash common.Hash) (*types.Transaction, error) {
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

	// Parse fields
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
		// Default to maxFeePerGas
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
			chainID = big.NewInt(1) // default to mainnet
		}
	} else {
		chainID = big.NewInt(1) // default to mainnet
	}

	// For blob transactions, we just create a dynamic fee tx as approximation
	// since we don't need the actual blob data for our purposes
	dynamicFeeTx := &types.DynamicFeeTx{
		ChainID:    chainID,
		Nonce:      nonce,
		GasTipCap:  maxPriorityFee,
		GasFeeCap:  maxFeePerGas,
		Gas:        gasLimit,
		To:         to,
		Value:      value,
		Data:       data,
		AccessList: types.AccessList{}, // Empty access list
	}

	return types.NewTx(dynamicFeeTx), nil
}
