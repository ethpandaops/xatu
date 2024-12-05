package processor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

// ItemExporter is an interface for exporting items.
type ItemExporter[T any] interface {
	// ExportItems exports a batch of items.
	//
	// This function is called synchronously, so there is no concurrency
	// safety requirement. However, due to the synchronous calling pattern,
	// it is critical that all timeouts and cancellations contained in the
	// passed context must be honored.
	//
	// Any retry logic must be contained in this function. The SDK that
	// calls this function will not implement any retry logic. All errors
	// returned by this function are considered unrecoverable and will be
	// reported to a configured error Handler.
	ExportItems(ctx context.Context, items []*T) error

	// Shutdown notifies the exporter of a pending halt to operations. The
	// exporter is expected to preform any cleanup or synchronization it
	// requires while honoring all timeouts and cancellations contained in
	// the passed context.
	Shutdown(ctx context.Context) error
}

// StreamingItemExporter is an interface supporting streaming export of items.
type StreamingItemExporter[T any] interface {
	ItemExporter[T]
	// IsStreaming returns true if the exporter supports streaming.
	IsStreaming() bool
	// InitStream initializes a new stream.
	InitStream(ctx context.Context) error
	// ExportItemsViaStream sends items via an existing stream.
	ExportItemsViaStream(ctx context.Context, items []*T) error
	// CloseStream closes the stream.
	CloseStream(ctx context.Context) error
}

// BatchItemProcessor is a processor that batches items for export.
type BatchItemProcessor[T any] struct {
	e ItemExporter[T]
	o BatchItemProcessorOptions

	log logrus.FieldLogger

	queue   chan *TraceableItem[T]
	batchCh chan []*TraceableItem[T]
	name    string

	timer         *time.Timer
	stopWait      sync.WaitGroup
	stopOnce      sync.Once
	stopCh        chan struct{}
	stopWorkersCh chan struct{}

	// Metrics
	metrics *Metrics

	// Streaming related attributes.
	streamLock         sync.Mutex
	streamActive       bool
	streamFailureCount int
	streamBackoffUntil time.Time
}

type TraceableItem[T any] struct {
	item        *T
	errCh       chan error
	completedCh chan struct{}
}

// NewBatchItemProcessor creates a new batch item processor.
func NewBatchItemProcessor[T any](exporter ItemExporter[T], name string, log logrus.FieldLogger, options ...BatchItemProcessorOption) (*BatchItemProcessor[T], error) {
	maxQueueSize := DefaultMaxQueueSize
	maxExportBatchSize := DefaultMaxExportBatchSize

	if maxExportBatchSize > maxQueueSize {
		if DefaultMaxExportBatchSize > maxQueueSize {
			maxExportBatchSize = maxQueueSize
		} else {
			maxExportBatchSize = DefaultMaxExportBatchSize
		}
	}

	o := BatchItemProcessorOptions{
		BatchTimeout:       time.Duration(DefaultScheduleDelay) * time.Millisecond,
		ExportTimeout:      time.Duration(DefaultExportTimeout) * time.Millisecond,
		MaxQueueSize:       maxQueueSize,
		MaxExportBatchSize: maxExportBatchSize,
		ShippingMethod:     DefaultShippingMethod,
		Workers:            DefaultNumWorkers,
	}
	for _, opt := range options {
		opt(&o)
	}

	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch item processor options: %w: %s", err, name)
	}

	metrics := o.Metrics
	if metrics == nil {
		metrics = DefaultMetrics
	}

	bvp := BatchItemProcessor[T]{
		e:             exporter,
		o:             o,
		log:           log,
		name:          name,
		metrics:       metrics,
		timer:         time.NewTimer(o.BatchTimeout),
		queue:         make(chan *TraceableItem[T], o.MaxQueueSize),
		batchCh:       make(chan []*TraceableItem[T], o.Workers),
		stopCh:        make(chan struct{}),
		stopWorkersCh: make(chan struct{}),
	}

	bvp.log.WithFields(
		logrus.Fields{
			"workers":               bvp.o.Workers,
			"batch_timeout":         bvp.o.BatchTimeout,
			"export_timeout":        bvp.o.ExportTimeout,
			"max_export_batch_size": bvp.o.MaxExportBatchSize,
			"max_queue_size":        bvp.o.MaxQueueSize,
			"shipping_method":       bvp.o.ShippingMethod,
		},
	).Info("Batch item processor initialized")

	return &bvp, nil
}

func (bvp *BatchItemProcessor[T]) Start(ctx context.Context) {
	bvp.stopWait.Add(bvp.o.Workers)

	bvp.metrics.SetWorkerCount(bvp.name, float64(bvp.o.Workers))

	bvp.log.Infof("Starting %d workers for %s", bvp.o.Workers, bvp.name)

	for i := 0; i < bvp.o.Workers; i++ {
		go func(num int) {
			defer bvp.stopWait.Done()
			bvp.worker(ctx, num)
		}(i)
	}

	go func() {
		bvp.batchBuilder(ctx)
		bvp.log.Info("Batch builder exited")
	}()
}

// Write writes items to the queue. If the Processor is configured to use
// the sync shipping method, the items will be written to the queue and this
// function will return when all items have been processed. If the Processor is
// configured to use the async shipping method, the items will be written to
// the queue and this function will return immediately.
func (bvp *BatchItemProcessor[T]) Write(ctx context.Context, s []*T) error {
	if len(s) == 0 {
		return nil
	}

	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.Write")
	defer span.End()

	if bvp.e == nil {
		return errors.New("exporter is nil")
	}

	// Break our items up in to chunks that can be processed at
	// one time by our workers. This is to prevent wasting
	// resources sending items if we've failed an earlier
	// batch.
	batchSize := bvp.o.Workers * bvp.o.MaxExportBatchSize
	for start := 0; start < len(s); start += batchSize {
		end := start + batchSize
		if end > len(s) {
			end = len(s)
		}

		prepared := []*TraceableItem[T]{}

		for _, i := range s[start:end] {
			if i == nil {
				bvp.metrics.IncItemsDroppedBy(bvp.name, float64(1))

				bvp.log.Warnf("Attempted to write a nil item. This item has been dropped. This probably shouldn't happen and is likely a bug.")

				continue
			}

			item := &TraceableItem[T]{
				item: i,
			}

			if bvp.o.ShippingMethod == ShippingMethodSync {
				item.errCh = make(chan error, 1)
				item.completedCh = make(chan struct{}, 1)
			}

			prepared = append(prepared, item)
		}

		for _, i := range prepared {
			if err := bvp.enqueueOrDrop(ctx, i); err != nil {
				return err
			}
		}

		if bvp.o.ShippingMethod == ShippingMethodSync {
			if err := bvp.waitForBatchCompletion(ctx, prepared); err != nil {
				return err
			}
		}
	}

	return nil
}

// Shutdown gracefully shuts down the batch item processor. It stops accepting new items,
// attempts to process remaining items, and cleans up resources. The sequence looks like:
// 1. Stop accepting new items.
// 2. Attempt to drain existing items from the queue.
// 3. Wait for workers to finish processing.
// 4. Shutdown the underlying exporter.
func (bvp *BatchItemProcessor[T]) Shutdown(ctx context.Context) error {
	var err error

	bvp.stopOnce.Do(func() {
		bvp.log.Info("Beginning shutdown sequence")

		// stopCh is used to prevent new items from being enqueued.
		close(bvp.stopCh)

		// stopWorkersCh signals workers to stop processing.
		close(bvp.stopWorkersCh)

		// Stop the batch timeout timer to prevent any new batches from being created.
		bvp.timer.Stop()

		// Mark streaming as inactive to prevent new stream initialization attempts.
		// No harm in doing this even if the exporter does not support streaming.
		bvp.streamLock.Lock()
		bvp.streamActive = false
		bvp.streamLock.Unlock()

		// Clear any items already in the queue, give it a timeout to complete, as
		// if there's a problem with one of the workers, we don't want to hang.
		drainCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if e := bvp.drainQueueWithTimeout(drainCtx); e != nil {
			bvp.log.WithError(e).Warn("Failed to drain queue during shutdown")
			close(bvp.queue)
		}

		// Wait for all workers to finish what they're doing. Again, give them
		// a timeout to prevent the potential of hanging and not being able to
		// shutdown cleanly.
		waitCh := make(chan struct{})
		go func() {
			bvp.stopWait.Wait()
			close(waitCh)
		}()

		// Give workers 2 seconds to finish before continuing shutdown
		select {
		case <-waitCh:
			bvp.log.Info("All workers stopped successfully")
		case <-time.After(5 * time.Second):
			bvp.log.Warn("Timeout waiting for workers to stop")
		}

		// Finally, shutdown the underlying exporter. It's the individual exporters
		// responsibility to honour timeouts and cancellations.
		if bvp.e != nil {
			if err = bvp.e.Shutdown(ctx); err != nil {
				bvp.log.WithError(err).Error("Failed to shutdown exporter")
			}
		}

		bvp.log.Info("Shutdown sequence completed")
	})

	return err
}

// exportWithTimeout exports items with a timeout. It handles both streaming and non-streaming
// export methods, with automatic fallback from streaming to unary if streaming fails.
// The function ensures proper cleanup of resources and updates metrics regardless of
// export success or failure.
func (bvp *BatchItemProcessor[T]) exportWithTimeout(ctx context.Context, itemsBatch []*TraceableItem[T]) error {
	if len(itemsBatch) == 0 {
		return nil
	}

	bvp.metrics.IncWorkerExportInProgress(bvp.name)
	defer bvp.metrics.DecWorkerExportInProgress(bvp.name)

	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.exportWithTimeout")
	defer span.End()
	span.SetAttributes(attribute.String("processor", bvp.name))
	span.SetAttributes(attribute.Int("batch_size", len(itemsBatch)))

	// Apply export timeout if configured. This ensures individual exports don't hang indefinitely.
	if bvp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bvp.o.ExportTimeout)
		defer cancel()
	}

	// Convert TraceableItems to raw items for export. TraceableItems contain
	// additional metadata used for tracking and error reporting.
	items := make([]*T, len(itemsBatch))
	for i, item := range itemsBatch {
		items[i] = item.item
	}

	var (
		err       error
		startTime = time.Now()
	)

	// Attempt streaming export if supported by the exporter. If streaming fails,
	// we fall back to unary export to ensure reliability.
	if streamExporter, ok := bvp.e.(StreamingItemExporter[T]); ok && streamExporter.IsStreaming() {
		bvp.log.WithFields(logrus.Fields{"batch_size": len(items), "processor": bvp.name}).Debug("Using streaming export")

		err = bvp.handleStreamingExport(ctx, streamExporter, items)
		if err != nil {
			bvp.log.WithError(err).Debugf("Streaming export failed, falling back to unary export with %d items to export", len(items))
			err = bvp.e.ExportItems(ctx, items)
		}
	} else {
		// Use standard unary export if streaming is not supported
		bvp.log.WithFields(logrus.Fields{"batch_size": len(items), "processor": bvp.name}).Debug("Using unary export")

		err = bvp.e.ExportItems(ctx, items)
	}

	duration := time.Since(startTime)
	bvp.metrics.ObserveExportDuration(bvp.name, duration)

	if err != nil {
		bvp.metrics.IncItemsFailedBy(bvp.name, float64(len(items)))
	} else {
		bvp.metrics.IncItemsExportedBy(bvp.name, float64(len(items)))
		bvp.metrics.ObserveBatchSize(bvp.name, float64(len(items)))
	}

	for _, item := range itemsBatch {
		if item.errCh != nil {
			item.errCh <- err
			close(item.errCh)
		}

		if item.completedCh != nil {
			item.completedCh <- struct{}{}
			close(item.completedCh)
		}
	}

	return err
}

// handleStreamingExport manages the streaming export process, including stream initialization,
// backoff handling, and actual export. Its thread-safe and implements exponential
// backoff for failed stream initializations.
func (bvp *BatchItemProcessor[T]) handleStreamingExport(ctx context.Context, exporter StreamingItemExporter[T], items []*T) error {
	// Ensure thread-safe access to streaming state
	bvp.streamLock.Lock()
	defer bvp.streamLock.Unlock()

	var (
		log = bvp.log.WithFields(logrus.Fields{
			"batch_size":    len(items),
			"processor":     bvp.name,
			"stream_active": bvp.streamActive,
		})
		now = time.Now()
	)

	if !bvp.streamActive {
		// If we're in a backoff period, skip initialization attempt and return error.
		// This error will trigger fallback to unary export in the caller.
		if now.Before(bvp.streamBackoffUntil) {
			log.WithField("backoff_remaining", bvp.streamBackoffUntil.Sub(now)).
				Debug("Skipping stream initialization due to backoff")

			return fmt.Errorf("stream initialization in backoff period")
		}

		// Attempt to initialize the stream. This is a blocking operation that
		// establishes a new gRPC stream with the server.
		log.Debug("Stream not active, attempting to initialize")

		if err := exporter.InitStream(ctx); err != nil {
			log.WithError(err).Error("Failed to initialize stream")

			// Something's not right. Calculate the next backoff period. Return
			// the error to trigger fallback to unary export in the caller.
			bvp.streamFailureCount++
			backoff := calculateBackoff(bvp.streamFailureCount)
			bvp.streamBackoffUntil = now.Add(backoff)

			log.WithFields(logrus.Fields{
				"failure_count": bvp.streamFailureCount,
				"backoff":       backoff,
			}).Debug("Stream initialization failed, backing off")

			return err
		}

		// Stream initialized successfully, reset failure tracking
		bvp.streamFailureCount = 0
		bvp.streamBackoffUntil = time.Time{}
		bvp.streamActive = true

		log.Debug("Stream initialized successfully")
	}

	// At this point we have an active stream (either existing or newly initialized).
	// Attempt to send items via the stream.
	log.Debug("Attempting to send via stream")

	if err := exporter.ExportItemsViaStream(ctx, items); err != nil {
		// If sending fails, mark stream as inactive so next attempt will
		// have a crack at reinitialising the stream. The error will propagate
		// up and may trigger fallback to unary export.
		log.WithError(err).Error("Failed to send via stream")

		bvp.streamActive = false

		return err
	}

	return nil
}

func (bvp *BatchItemProcessor[T]) waitForBatchCompletion(ctx context.Context, items []*TraceableItem[T]) error {
	for _, item := range items {
		select {
		case err := <-item.errCh:
			if err != nil {
				return err
			}
		case <-item.completedCh:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (bvp *BatchItemProcessor[T]) batchBuilder(ctx context.Context) {
	log := bvp.log.WithField("module", "batch_builder")
	var batch []*TraceableItem[T]

	for {
		select {
		case <-bvp.stopWorkersCh:
			log.Info("Stopping batch builder")
			return
		case item, ok := <-bvp.queue:
			if !ok {
				// Channel is closed, send any remaining items in the batch for processing
				// before shutting down.
				if len(batch) > 0 {
					bvp.sendBatch(batch, "shutdown")
				}

				return
			}

			if item == nil {
				bvp.metrics.IncItemsDroppedBy(bvp.name, float64(1))
				bvp.log.Warnf("Attempted to build a batch with a nil item. This item has been dropped. This probably shouldn't happen and is likely a bug.")
				continue
			}

			batch = append(batch, item)

			if len(batch) >= bvp.o.MaxExportBatchSize {
				bvp.sendBatch(batch, "max_export_batch_size")

				batch = []*TraceableItem[T]{}
			}
		case <-bvp.timer.C:
			if len(batch) > 0 {
				bvp.sendBatch(batch, "timer")
				batch = []*TraceableItem[T]{}
			} else {
				bvp.timer.Reset(bvp.o.BatchTimeout)
			}
		}
	}
}

func (bvp *BatchItemProcessor[T]) sendBatch(batch []*TraceableItem[T], reason string) {
	log := bvp.log.WithField("reason", reason)
	log.Tracef("Creating a batch of %d items", len(batch))

	log.Tracef("Batch items copied")

	bvp.batchCh <- batch

	log.Tracef("Batch sent to batch channel")
}

func (bvp *BatchItemProcessor[T]) worker(ctx context.Context, number int) {
	for {
		select {
		case <-ctx.Done():
			bvp.log.Infof("Worker %d stopping due to context cancellation", number)

			return
		case <-bvp.stopWorkersCh:
			bvp.log.Infof("Worker %d stopping due to stop signal", number)

			return
		case batch := <-bvp.batchCh:
			bvp.timer.Reset(bvp.o.BatchTimeout)

			if err := bvp.exportWithTimeout(ctx, batch); err != nil {
				bvp.log.WithError(err).Error("failed to export items")
			}

			bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))
		}
	}
}

// drainQueueWithTimeout attempts to drain any remaining items from both the input queue
// and batch channel during shutdown. It polls the queues periodically until they're empty
// or until timeout/cancellation occurs.
//
// The function uses two channels to track items:
// - queue: Contains individual items waiting to be batched.
// - batchCh: Contains batches of items waiting to be processed by workers.
//
// Both must be empty for a successful drain.
func (bvp *BatchItemProcessor[T]) drainQueueWithTimeout(ctx context.Context) error {
	bvp.log.Info("Attempting to drain queue")

	// Create a ticker to periodically check queue status.
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Set a hard timeout to prevent hanging if queues aren't draining.
	// This is separate from the context timeout to ensure we don't
	// wait indefinitely even if the context has no deadline.
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			// Context was cancelled or timed out.
			return ctx.Err()
		case <-timeout:
			// Hard timeout reached before queues were drained.
			return fmt.Errorf("queue drain timed out")
		case <-ticker.C:
			// Check if both queues are empty. We need to check both because:
			// 1. queue holds individual items waiting to be batched.
			// 2. batchCh holds batches waiting to be processed.
			if len(bvp.queue) == 0 && len(bvp.batchCh) == 0 {
				bvp.log.Info("Queue drained successfully")

				return nil
			}

			bvp.log.WithFields(logrus.Fields{
				"queue_size":    len(bvp.queue),
				"batch_ch_size": len(bvp.batchCh),
			}).Debug("Still draining queue...")
		}
	}
}

func recoverSendOnClosedChan() {
	x := recover()
	switch err := x.(type) {
	case nil:
		return
	case runtime.Error:
		if err.Error() == "send on closed channel" {
			return
		}
	}
	panic(x)
}

func (bvp *BatchItemProcessor[T]) enqueueOrDrop(ctx context.Context, item *TraceableItem[T]) error {
	// This ensures the bvp.queue<- below does not panic as the
	// processor shuts down.
	defer recoverSendOnClosedChan()

	select {
	case <-bvp.stopCh:
		return errors.New("processor is shutting down")
	default:
	}

	select {
	case bvp.queue <- item:
		bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

		return nil
	default:
		bvp.metrics.IncItemsDroppedBy(bvp.name, float64(1))
	}

	return errors.New("queue is full")
}

// calculateBackoff returns a backoff duration based on failure count. Start with 5
// second backoff, double each time, max 5 minutes.
func calculateBackoff(failures int) time.Duration {
	var (
		backoff    = time.Duration(5*math.Pow(2, float64(failures-1))) * time.Second
		maxBackoff = 5 * time.Minute
	)

	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}
