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

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/execution/cryo"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// AddressAppearancesDeriverName is the cannon type produced by the address
// appearances deriver.
const AddressAppearancesDeriverName = xatu.CannonType_EXECUTION_CANONICAL_ADDRESS_APPEARANCES

const addressAppearancesDataset = "address_appearances"

// addressAppearancesColumns restricts cryo output to exactly the columns mapped.
var addressAppearancesColumns = []string{
	"block_number",
	"transaction_hash",
	"address",
	"relationship",
}

// addressAppearanceRow mirrors the cryo `address_appearances` parquet schema
// (--hex). InternalIndex has no parquet tag; it is stamped post-read.
type addressAppearanceRow struct {
	BlockNumber     uint32 `parquet:"block_number"`
	TransactionHash string `parquet:"transaction_hash"`
	Address         string `parquet:"address"`
	Relationship    string `parquet:"relationship"`
	InternalIndex   uint32
}

// AddressAppearancesDeriverConfig configures the address appearances deriver.
type AddressAppearancesDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// AddressAppearancesDeriver derives canonical_execution_address_appearances
// events via cryo.
type AddressAppearancesDeriver struct {
	base

	cfg        *AddressAppearancesDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewAddressAppearancesDeriver creates an address appearances deriver.
func NewAddressAppearancesDeriver(
	log observability.ContextualLogger,
	config *AddressAppearancesDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *AddressAppearancesDeriver {
	return &AddressAppearancesDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/address_appearances",
				"type":   AddressAppearancesDeriverName.String(),
			}),
			name:     AddressAppearancesDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *AddressAppearancesDeriver) CannonType() xatu.CannonType {
	return AddressAppearancesDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *AddressAppearancesDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *AddressAppearancesDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *AddressAppearancesDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution address appearances deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution address appearances deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the address appearances deriver.
func (b *AddressAppearancesDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *AddressAppearancesDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"AddressAppearancesDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, addressAppearancesDataset, from, to, addressAppearancesColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect address appearances via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[addressAppearanceRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo address appearances parquet")
	}

	stampInternalIndex(rows,
		func(r *addressAppearanceRow) uint64 { return uint64(r.BlockNumber) },
		func(r *addressAppearanceRow) string { return r.TransactionHash },
		func(r *addressAppearanceRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of address
// appearance rows.
func (b *AddressAppearancesDeriver) createEvent(rows []addressAppearanceRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	appearances := make([]*xatu.ExecutionAddressAppearance, 0, len(rows))

	for i := range rows {
		appearances = append(appearances, addressAppearanceRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_ADDRESS_APPEARANCES,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalAddressAppearances{
			ExecutionCanonicalAddressAppearances: &xatu.ExecutionCanonicalAddressAppearances{AddressAppearances: appearances},
		},
	}, nil
}

// addressAppearanceRowToProto converts a cryo address appearance row to proto.
func addressAppearanceRowToProto(row *addressAppearanceRow) *xatu.ExecutionAddressAppearance {
	return &xatu.ExecutionAddressAppearance{
		BlockNumber:     uint64(row.BlockNumber),
		TransactionHash: row.TransactionHash,
		InternalIndex:   row.InternalIndex,
		Address:         row.Address,
		Relationship:    row.Relationship,
	}
}
