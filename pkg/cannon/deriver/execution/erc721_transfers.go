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
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/cryo"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Erc721TransfersDeriverName is the cannon type produced by the erc721 transfers deriver.
const Erc721TransfersDeriverName = xatu.CannonType_EXECUTION_CANONICAL_ERC721_TRANSFERS

const erc721TransfersDataset = "erc721_transfers"

// erc721TransfersColumns restricts cryo output to exactly the columns mapped.
// Note: cryo's contract column is named `erc20` even for erc721 transfers, and
// the token id is taken from token_id (parquet output token_id_string).
var erc721TransfersColumns = []string{
	"block_number",
	"transaction_index",
	"log_index",
	"transaction_hash",
	"erc20",
	"from_address",
	"to_address",
	// cryo's `token_id` expands (with --hex) to token_id_binary/token_id_string/token_id_f64;
	// erc721TransferRow reads token_id_string (decimal) and parquet-go projects the rest.
	"token_id",
}

// erc721TransferRow mirrors the cryo `erc721_transfers` parquet schema (--hex).
type erc721TransferRow struct {
	BlockNumber      uint32 `parquet:"block_number"`
	TransactionIndex uint32 `parquet:"transaction_index"`
	LogIndex         uint32 `parquet:"log_index"`
	TransactionHash  string `parquet:"transaction_hash"`
	Erc721           string `parquet:"erc20"`
	FromAddress      string `parquet:"from_address"`
	ToAddress        string `parquet:"to_address"`
	Token            string `parquet:"token_id_string"`

	InternalIndex uint32
}

// Erc721TransfersDeriverConfig configures the erc721 transfers deriver.
type Erc721TransfersDeriverConfig struct {
	Enabled   bool                            `yaml:"enabled" default:"false"`
	ChunkSize int                             `yaml:"chunkSize" default:"100"`
	Iterator  iterator.BackfillingBlockConfig `yaml:"iterator"`
}

// Erc721TransfersDeriver derives canonical_execution_erc721_transfers events via cryo.
type Erc721TransfersDeriver struct {
	base

	cfg        *Erc721TransfersDeriverConfig
	cryo       *cryo.Runner
	clientMeta *xatu.ClientMeta
}

// NewErc721TransfersDeriver creates an erc721 transfers deriver.
func NewErc721TransfersDeriver(
	log observability.ContextualLogger,
	config *Erc721TransfersDeriverConfig,
	iter *iterator.BackfillingBlock,
	runner *cryo.Runner,
	beacon *ethereum.BeaconNode,
	clientMeta *xatu.ClientMeta,
) *Erc721TransfersDeriver {
	return &Erc721TransfersDeriver{
		base: base{
			log: log.WithFields(logrus.Fields{
				"module": "cannon/event/execution/erc721_transfers",
				"type":   Erc721TransfersDeriverName.String(),
			}),
			name:     Erc721TransfersDeriverName.String(),
			beacon:   beacon,
			iterator: iter,
		},
		cfg:        config,
		cryo:       runner,
		clientMeta: clientMeta,
	}
}

// CannonType returns the deriver's cannon type.
func (b *Erc721TransfersDeriver) CannonType() xatu.CannonType {
	return Erc721TransfersDeriverName
}

// ActivationFork returns Phase0 so the deriver starts as soon as the beacon
// node is ready; the iterator gates EL progress on CL finality.
func (b *Erc721TransfersDeriver) ActivationFork() spec.DataVersion {
	return spec.DataVersionPhase0
}

// Name returns the deriver name.
func (b *Erc721TransfersDeriver) Name() string {
	return b.name
}

// Start begins the deriver loop.
func (b *Erc721TransfersDeriver) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.log.WithContext(ctx).Info("Execution erc721 transfers deriver disabled")

		return nil
	}

	b.log.WithContext(ctx).Info("Execution erc721 transfers deriver enabled")

	if err := b.iterator.Start(ctx); err != nil {
		return errors.Wrap(err, "failed to start iterator")
	}

	b.run(ctx, b.processRange)

	return nil
}

// Stop is a no-op for the erc721 transfers deriver.
func (b *Erc721TransfersDeriver) Stop(_ context.Context) error {
	return nil
}

// processRange runs cryo for [from, to] and builds chunked DecoratedEvents.
func (b *Erc721TransfersDeriver) processRange(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error) {
	ctx, span := observability.Tracer().Start(ctx,
		"Erc721TransfersDeriver.processRange",
		trace.WithAttributes(
			attribute.Int64("from", int64(from)), //nolint:gosec // block numbers are far below int64 max.
			attribute.Int64("to", int64(to)),     //nolint:gosec // block numbers are far below int64 max.
		),
	)
	defer span.End()

	collection, err := b.cryo.Collect(ctx, erc721TransfersDataset, from, to, erc721TransfersColumns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to collect erc721 transfers via cryo")
	}

	defer func() {
		if cErr := collection.Close(); cErr != nil {
			b.log.WithError(cErr).WithContext(ctx).Warn("Failed to clean up cryo output")
		}
	}()

	rows, err := cryo.ReadParquet[erc721TransferRow](collection.Files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cryo erc721 transfers parquet")
	}

	stampInternalIndex(rows,
		func(r *erc721TransferRow) uint64 { return uint64(r.BlockNumber) },
		func(r *erc721TransferRow) string { return r.TransactionHash },
		func(r *erc721TransferRow, idx uint32) { r.InternalIndex = idx },
	)

	return chunkEvents(rows, b.cfg.ChunkSize, b.createEvent)
}

// createEvent builds a single DecoratedEvent carrying a chunk of erc721 transfer rows.
func (b *Erc721TransfersDeriver) createEvent(rows []erc721TransferRow) (*xatu.DecoratedEvent, error) {
	metadata, ok := proto.Clone(b.clientMeta).(*xatu.ClientMeta)
	if !ok {
		return nil, errors.New("failed to clone client metadata")
	}

	transfers := make([]*xatu.ExecutionErc721Transfer, 0, len(rows))

	for i := range rows {
		transfers = append(transfers, erc721TransferRowToProto(&rows[i]))
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_ERC721_TRANSFERS,
			DateTime: timestamppb.New(time.Now()),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{Client: metadata},
		Data: &xatu.DecoratedEvent_ExecutionCanonicalErc721Transfers{
			ExecutionCanonicalErc721Transfers: &xatu.ExecutionCanonicalErc721Transfers{Erc721Transfers: transfers},
		},
	}, nil
}

// erc721TransferRowToProto converts a cryo erc721 transfer row to proto.
func erc721TransferRowToProto(row *erc721TransferRow) *xatu.ExecutionErc721Transfer {
	return &xatu.ExecutionErc721Transfer{
		BlockNumber:      uint64(row.BlockNumber),
		TransactionIndex: uint64(row.TransactionIndex),
		TransactionHash:  row.TransactionHash,
		InternalIndex:    row.InternalIndex,
		LogIndex:         uint64(row.LogIndex),
		Erc721:           row.Erc721,
		FromAddress:      row.FromAddress,
		ToAddress:        row.ToAddress,
		Token:            row.Token,
	}
}
