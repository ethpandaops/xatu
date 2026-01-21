package deriver

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/xatu/pkg/cldata"
	"github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ExecutionTransactionDeriverName = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION
)

// ExecutionTransactionDeriverConfig holds the configuration for the ExecutionTransactionDeriver.
type ExecutionTransactionDeriverConfig struct {
	Enabled bool `yaml:"enabled" default:"true"`
}

// ExecutionTransactionDeriver derives execution transaction events from the consensus layer.
// It processes epochs of blocks and emits decorated events for each execution transaction.
type ExecutionTransactionDeriver struct {
	log               logrus.FieldLogger
	cfg               *ExecutionTransactionDeriverConfig
	iterator          iterator.Iterator
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
	beacon            cldata.BeaconClient
	ctx               cldata.ContextProvider
}

// NewExecutionTransactionDeriver creates a new ExecutionTransactionDeriver instance.
func NewExecutionTransactionDeriver(
	log logrus.FieldLogger,
	config *ExecutionTransactionDeriverConfig,
	iter iterator.Iterator,
	beacon cldata.BeaconClient,
	ctxProvider cldata.ContextProvider,
) *ExecutionTransactionDeriver {
	return &ExecutionTransactionDeriver{
		log: log.WithFields(logrus.Fields{
			"module": "cldata/deriver/execution_transaction",
			"type":   ExecutionTransactionDeriverName.String(),
		}),
		cfg:      config,
		iterator: iter,
		beacon:   beacon,
		ctx:      ctxProvider,
	}
}

func (e *ExecutionTransactionDeriver) CannonType() xatu.CannonType {
	return ExecutionTransactionDeriverName
}

func (e *ExecutionTransactionDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionBellatrix
}

func (e *ExecutionTransactionDeriver) Name() string {
	return ExecutionTransactionDeriverName.String()
}

func (e *ExecutionTransactionDeriver) OnEventsDerived(
	ctx context.Context,
	fn func(ctx context.Context, events []*xatu.DecoratedEvent) error,
) {
	e.onEventsCallbacks = append(e.onEventsCallbacks, fn)
}

func (e *ExecutionTransactionDeriver) Start(ctx context.Context) error {
	if !e.cfg.Enabled {
		e.log.Info("Execution transaction deriver disabled")

		return nil
	}

	e.log.Info("Execution transaction deriver enabled")

	if err := e.iterator.Start(ctx, e.ActivationFork()); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	// Start our main loop
	e.run(ctx)

	return nil
}

func (e *ExecutionTransactionDeriver) Stop(ctx context.Context) error {
	return nil
}

func (e *ExecutionTransactionDeriver) run(rctx context.Context) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", e.Name()),
					trace.WithAttributes(
						attribute.String("network", e.ctx.NetworkName())),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := e.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Get the next position
				position, err := e.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				// Look ahead
				e.lookAhead(ctx, position.LookAheadEpochs)

				// Process the epoch
				events, err := e.processEpoch(ctx, position.Epoch)
				if err != nil {
					e.log.WithError(err).Error("Failed to process epoch")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Epoch processing complete. Sending events...")

				// Send the events
				for _, fn := range e.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				span.AddEvent("Events sent. Updating location...")

				// Update our location
				if err := e.iterator.UpdateLocation(ctx, position); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				span.AddEvent("Location updated. Done.")

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					e.log.WithError(err).WithField("next_attempt", timer).Warn("Failed to process")
				}),
			}
			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				e.log.WithError(err).Warn("Failed to process")
			}
		}
	}
}

// lookAhead takes the upcoming epochs and looks ahead to do any pre-processing that might be required.
func (e *ExecutionTransactionDeriver) lookAhead(ctx context.Context, epochs []phase0.Epoch) {
	_, span := observability.Tracer().Start(ctx,
		"ExecutionTransactionDeriver.lookAhead",
	)
	defer span.End()

	sp, err := e.beacon.Node().Spec()
	if err != nil {
		e.log.WithError(err).Warn("Failed to look ahead at epoch")

		return
	}

	for _, epoch := range epochs {
		for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
			slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

			// Add the block to the preload queue so it's available when we need it
			e.beacon.LazyLoadBeaconBlock(xatuethv1.SlotAsString(slot))
		}
	}
}

func (e *ExecutionTransactionDeriver) processEpoch(ctx context.Context, epoch phase0.Epoch) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionTransactionDeriver.processEpoch",
		//nolint:gosec // epoch numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("epoch", int64(epoch))),
	)
	defer span.End()

	sp, err := e.beacon.Node().Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain spec")
	}

	allEvents := make([]*xatu.DecoratedEvent, 0)

	for i := uint64(0); i <= uint64(sp.SlotsPerEpoch-1); i++ {
		slot := phase0.Slot(i + uint64(epoch)*uint64(sp.SlotsPerEpoch))

		events, err := e.processSlot(ctx, slot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process slot %d", slot)
		}

		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

func (e *ExecutionTransactionDeriver) processSlot(ctx context.Context, slot phase0.Slot) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ExecutionTransactionDeriver.processSlot",
		//nolint:gosec // slot numbers won't exceed int64 max in practice
		trace.WithAttributes(attribute.Int64("slot", int64(slot))),
	)
	defer span.End()

	// Get the block
	block, err := e.beacon.GetBeaconBlock(ctx, xatuethv1.SlotAsString(slot))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get beacon block for slot %d", slot)
	}

	if block == nil {
		return []*xatu.DecoratedEvent{}, nil
	}

	blockIdentifier, err := GetBlockIdentifier(block, e.ctx.Wallclock())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get block identifier for slot %d", slot)
	}

	blobSidecars := []*deneb.BlobSidecar{}

	if block.Version >= spec.DataVersionDeneb {
		sidecars, errr := e.beacon.FetchBeaconBlockBlobs(ctx, xatuethv1.SlotAsString(slot))
		if errr != nil {
			var apiErr *api.Error
			if errors.As(errr, &apiErr) {
				switch apiErr.StatusCode {
				case 404:
					e.log.WithError(errr).WithField("slot", slot).Debug("no beacon block blob sidecars found for slot")
				case 503:
					return nil, errors.New("beacon node is syncing")
				default:
					return nil, errors.Wrapf(errr, "failed to get beacon block blob sidecars for slot %d", slot)
				}
			} else {
				return nil, errors.Wrapf(errr, "failed to get beacon block blob sidecars for slot %d", slot)
			}
		}

		blobSidecars = sidecars
	}

	blobSidecarsMap := make(map[string]*deneb.BlobSidecar, len(blobSidecars))

	for _, blobSidecar := range blobSidecars {
		versionedHash := cldata.ConvertKzgCommitmentToVersionedHash(blobSidecar.KZGCommitment[:])
		blobSidecarsMap[versionedHash.String()] = blobSidecar
	}

	events := make([]*xatu.DecoratedEvent, 0)

	transactions, err := e.getExecutionTransactions(ctx, block)
	if err != nil {
		return nil, err
	}

	chainID := new(big.Int).SetUint64(e.ctx.DepositChainID())
	if chainID.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("failed to get chain ID from context provider")
	}

	signer := types.LatestSignerForChainID(chainID)

	for index, transaction := range transactions {
		from, err := types.Sender(signer, transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction sender: %v", err)
		}

		gasPrice, err := GetGasPrice(block, transaction)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction gas price: %v", err)
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
				e.log.WithField("transaction", transaction.Hash().Hex()).Warn("no versioned hashes for type 3 transaction")
			}

			for i := 0; i < len(transaction.BlobHashes()); i++ {
				hash := transaction.BlobHashes()[i]
				blobHashes[i] = hash.String()
				sidecar := blobSidecarsMap[hash.String()]

				if sidecar != nil {
					sidecarsSize += len(sidecar.Blob)
					sidecarsEmptySize += cldata.CountConsecutiveEmptyBytes(sidecar.Blob[:], 4)
				} else {
					e.log.WithField("versioned hash", hash.String()).WithField("transaction", transaction.Hash().Hex()).Warn("missing blob sidecar")
				}
			}

			tx.BlobGas = wrapperspb.UInt64(transaction.BlobGas())
			tx.BlobGasFeeCap = transaction.BlobGasFeeCap().String()
			tx.BlobHashes = blobHashes
		}

		//nolint:gosec // index from range is always non-negative
		event, err := e.createEvent(ctx, tx, uint64(index), blockIdentifier, transaction, sidecarsSize, sidecarsEmptySize)
		if err != nil {
			e.log.WithError(err).Error("Failed to create event")

			return nil, errors.Wrapf(err, "failed to create event for execution transaction %s", transaction.Hash())
		}

		events = append(events, event)
	}

	return events, nil
}

func (e *ExecutionTransactionDeriver) getExecutionTransactions(
	_ context.Context,
	block *spec.VersionedSignedBeaconBlock,
) ([]*types.Transaction, error) {
	transactions := make([]*types.Transaction, 0)

	txs, err := block.ExecutionTransactions()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution transactions: %v", err)
	}

	for _, transaction := range txs {
		ethTransaction := new(types.Transaction)
		if err := ethTransaction.UnmarshalBinary(transaction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction: %v", err)
		}

		transactions = append(transactions, ethTransaction)
	}

	return transactions, nil
}

func (e *ExecutionTransactionDeriver) createEvent(
	ctx context.Context,
	transaction *xatuethv1.Transaction,
	positionInBlock uint64,
	blockIdentifier *xatu.BlockIdentifier,
	rlpTransaction *types.Transaction,
	sidecarsSize, sidecarsEmptySize int,
) (*xatu.DecoratedEvent, error) {
	// Get client metadata
	clientMeta, err := e.ctx.CreateClientMeta(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client metadata")
	}

	// Make a clone of the metadata
	metadata, ok := proto.Clone(clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: metadata,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: transaction,
		},
	}

	decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlockExecutionTransaction{
		EthV2BeaconBlockExecutionTransaction: &xatu.ClientMeta_AdditionalEthV2BeaconBlockExecutionTransactionData{
			Block:                 blockIdentifier,
			PositionInBlock:       wrapperspb.UInt64(positionInBlock),
			Size:                  strconv.FormatFloat(float64(rlpTransaction.Size()), 'f', 0, 64),
			CallDataSize:          fmt.Sprintf("%d", len(rlpTransaction.Data())),
			BlobSidecarsSize:      fmt.Sprint(sidecarsSize),
			BlobSidecarsEmptySize: fmt.Sprint(sidecarsEmptySize),
		},
	}

	return decoratedEvent, nil
}

// GetGasPrice calculates the effective gas price for a transaction based on its type and block version.
func GetGasPrice(block *spec.VersionedSignedBeaconBlock, transaction *types.Transaction) (*big.Int, error) {
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
