package execution

import (
	"context"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/cryo"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ContractsDeriverName is the cannon type produced by the contracts deriver.
const ContractsDeriverName = xatu.CannonType_EXECUTION_CANONICAL_CONTRACTS

const contractsDataset = "contracts"

// contractsColumns restricts cryo output to exactly the columns mapped.
var contractsColumns = []string{
	"block_number",
	"create_index",
	"transaction_hash",
	"contract_address",
	"deployer",
	"factory",
	"init_code",
	"code",
	"init_code_hash",
	"n_init_code_bytes",
	"n_code_bytes",
	"code_hash",
}

// contractsRow mirrors the cryo `contracts` parquet schema collected with --hex.
type contractsRow struct {
	BlockNumber     uint32 `parquet:"block_number"`
	CreateIndex     uint32 `parquet:"create_index"`
	TransactionHash string `parquet:"transaction_hash"`
	ContractAddress string `parquet:"contract_address"`
	Deployer        string `parquet:"deployer"`
	Factory         string `parquet:"factory"`
	InitCode        string `parquet:"init_code"`
	Code            string `parquet:"code"`
	InitCodeHash    string `parquet:"init_code_hash"`
	NInitCodeBytes  uint32 `parquet:"n_init_code_bytes"`
	NCodeBytes      uint32 `parquet:"n_code_bytes"`
	CodeHash        string `parquet:"code_hash"`
	InternalIndex   uint32
}

// ContractsDeriverConfig configures the contracts deriver.
type ContractsDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// ContractsDeriver derives canonical_execution_contracts events via cryo.
type ContractsDeriver struct {
	base

	cfg        *ContractsDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewContractsDeriver creates a contracts deriver.
func NewContractsDeriver(
	log observability.ContextualLogger,
	config *ContractsDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *ContractsDeriver {
	return &ContractsDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/contracts",
				"type":   ContractsDeriverName.String(),
			}),
			name:     ContractsDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *ContractsDeriver) CannonType() xatu.CannonType {
	return ContractsDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *ContractsDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *ContractsDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *ContractsDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution contracts deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution contracts deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the contracts deriver.
func (b *ContractsDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *ContractsDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"ContractsDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, contractsDataset, from, to, contractsColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect contracts via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[contractsRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo contracts parquet")
	}

	stampInternalIndex(rows,
		func(r *contractsRow) uint64 { return uint64(r.BlockNumber) },
		func(r *contractsRow) string { return r.TransactionHash },
		func(r *contractsRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of contract rows.
func (b *ContractsDeriver) createEvent(rows []contractsRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	contracts := make([]*xatu.ExecutionContract, 0, len(rows))

	for i := range rows {
		contracts = append(contracts, contractsRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_CONTRACTS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalContracts{
			ExecutionCanonicalContracts: &xatu.ExecutionCanonicalContracts{Contracts: contracts},
		},
	}, nil
}

// contractsRowToProto converts a cryo contract row (collected with --hex) to proto.
func contractsRowToProto(row *contractsRow) *xatu.ExecutionContract {
	out := &xatu.ExecutionContract{
		BlockNumber:     uint64(row.BlockNumber),
		TransactionHash: row.TransactionHash,
		InternalIndex:   row.InternalIndex,
		CreateIndex:     row.CreateIndex,
		ContractAddress: row.ContractAddress,
		Deployer:        row.Deployer,
		Factory:         row.Factory,
		InitCode:        row.InitCode,
		InitCodeHash:    row.InitCodeHash,
		NInitCodeBytes:  row.NInitCodeBytes,
		NCodeBytes:      row.NCodeBytes,
		CodeHash:        row.CodeHash,
	}

	if row.Code != "" {
		out.Code = wrapperspb.String(row.Code)
	}

	return out
}
