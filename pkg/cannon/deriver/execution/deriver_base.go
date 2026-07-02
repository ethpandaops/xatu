package execution

import (
	"context"
	"fmt"
	"strconv"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// eventProducer derives the events for an inclusive block range.
type eventProducer func(ctx context.Context, from, to uint64) ([]*xatu.DecoratedEvent, error)

// stampInternalIndex assigns each row a 1-based ordinal within its
// (block_number, transaction_hash) group, in the order cryo emitted the rows.
// ClickHouse does not preserve insert order across a cluster, so this ordinal
// is what makes rows uniquely orderable (it is part of the ReplacingMergeTree
// ORDER BY). This replicates the legacy pipeline's
// `groupby(block_number, transaction_hash).cumcount() + 1`. Rows MUST be passed
// in cryo's parquet order and stamped before chunking.
func stampInternalIndex[T any](rows []T, blockNumber func(*T) uint64, txHash func(*T) string, set func(*T, uint32)) {
	seen := make(map[string]uint32, len(rows))

	for i := range rows {
		key := strconv.FormatUint(blockNumber(&rows[i]), 10) + ":" + txHash(&rows[i])
		seen[key]++
		set(&rows[i], seen[key])
	}
}

// defaultChunkSize bounds the rows per DecoratedEvent when a deriver config
// leaves chunkSize unset.
const defaultChunkSize = 100

// zeroHexIfEmpty maps an empty cryo value to "0x" for NON-nullable hex columns,
// matching the legacy pipeline (which read such columns as a non-null String and
// emitted concat('0x', hex(x)) — i.e. "0x" for empty/absent bytes).
func zeroHexIfEmpty(s string) string {
	if s == "" {
		return "0x"
	}

	return s
}

// chunkEvents splits rows into chunks of chunkSize and builds one
// DecoratedEvent per chunk via create. This is the shared "chunked
// repeated-payload" fan-out used by every EL deriver.
func chunkEvents[T any](rows []T, chunkSize int, create func([]T) (*xatu.DecoratedEvent, error)) ([]*xatu.DecoratedEvent, error) {
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	events := make([]*xatu.DecoratedEvent, 0, (len(rows)/chunkSize)+1)

	for start := 0; start < len(rows); start += chunkSize {
		end := min(start+chunkSize, len(rows))

		event, err := create(rows[start:end])
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// base holds the machinery shared by every EL cannon deriver: the iterator
// loop, the callback fan-out, and the retry/backoff. Each concrete deriver
// embeds base and supplies an eventProducer (its processRange) to run.
type base struct {
	log               observability.ContextualLogger
	name              string
	beacon            *ethereum.BeaconNode
	iterator          *iterator.BackfillingBlock
	onEventsCallbacks []func(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// OnEventsDerived registers a callback for derived events.
func (b *base) OnEventsDerived(_ context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	b.onEventsCallbacks = append(b.onEventsCallbacks, fn)
}

// run is the deriver main loop: pull the next range, produce events, fan them
// out to sinks, then advance the cursor — retrying the whole unit on failure.
func (b *base) run(rctx context.Context, produce eventProducer) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = 3 * time.Minute

	tracer := observability.Tracer()

	for {
		select {
		case <-rctx.Done():
			return
		default:
			operation := func() (string, error) {
				ctx, span := tracer.Start(rctx, fmt.Sprintf("Derive %s", b.name),
					trace.WithAttributes(
						attribute.String("network", string(b.beacon.Metadata().Network.Name))),
				)
				defer span.End()

				time.Sleep(100 * time.Millisecond)

				if err := b.beacon.Synced(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				position, err := b.iterator.Next(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				events, err := produce(ctx, position.From, position.To)
				if err != nil {
					b.log.WithError(err).WithContext(ctx).Error("Failed to process block range")

					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				for _, fn := range b.onEventsCallbacks {
					if err := fn(ctx, events); err != nil {
						span.SetStatus(codes.Error, err.Error())

						return "", errors.Wrap(err, "failed to send events")
					}
				}

				if err := b.iterator.UpdateLocation(ctx, position.From, position.To, position.Direction); err != nil {
					span.SetStatus(codes.Error, err.Error())

					return "", err
				}

				bo.Reset()

				return "", nil
			}

			retryOpts := []backoff.RetryOption{
				backoff.WithBackOff(bo),
				backoff.WithNotify(func(err error, timer time.Duration) {
					b.log.WithError(err).WithField("next_attempt", timer).WithContext(rctx).Warn("Failed to process")
				}),
			}

			if _, err := backoff.Retry(rctx, operation, retryOpts...); err != nil {
				b.log.WithError(err).WithContext(rctx).Warn("Failed to process")
			}
		}
	}
}
