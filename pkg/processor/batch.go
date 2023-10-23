/*
 * This processor was adapted from the OpenTelemetry Collector's batch processor.
 *
 * Authors: OpenTelemetry
 * URL: https://github.com/open-telemetry/opentelemetry-go/blob/496c086ece129182662c14d6a023a2b2da09fe30/sdk/trace/batch_span_processor.go
 */

package processor

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/sirupsen/logrus"
)

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

// Defaults for BatchItemProcessorOptions.
const (
	DefaultMaxQueueSize       = 51200
	DefaultScheduleDelay      = 5000
	DefaultExportTimeout      = 30000
	DefaultMaxExportBatchSize = 512
	DefaultShippingMethod     = ShippingMethodAsync
	DefaultNumWorkers         = 1
)

type ShippingMethod string

const (
	ShippingMethodUnknown ShippingMethod = "unknown"
	ShippingMethodAsync   ShippingMethod = "async"
	ShippingMethodSync    ShippingMethod = "sync"
)

// BatchItemProcessorOption configures a BatchItemProcessor.
type BatchItemProcessorOption func(o *BatchItemProcessorOptions)

// BatchItemProcessorOptions is configuration settings for a
// BatchItemProcessor.
type BatchItemProcessorOptions struct {
	// MaxQueueSize is the maximum queue size to buffer items for delayed processing. If the
	// queue gets full it drops the items.
	// The default value of MaxQueueSize is 51200.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available items when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// ExportTimeout specifies the maximum duration for exporting items. If the timeout
	// is reached, the export will be cancelled.
	// The default value of ExportTimeout is 30000 msec.
	ExportTimeout time.Duration

	// MaxExportBatchSize is the maximum number of items to process in a single batch.
	// If there are more than one batch worth of items then it processes multiple batches
	// of items one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int

	// ShippingMethod is the method used to ship items to the exporter.
	ShippingMethod ShippingMethod

	// Number of workers to process items.
	Workers int
}

// BatchItemProcessor is a buffer that batches asynchronously-received
// items and sends them to a exporter when complete.
type BatchItemProcessor[T any] struct {
	e ItemExporter[T]
	o BatchItemProcessorOptions

	log logrus.FieldLogger

	queue   chan *T
	dropped uint32
	name    string

	metrics *Metrics

	batches    chan []*T
	batchReady chan bool

	batch      []*T
	batchMutex sync.Mutex

	timer         *time.Timer
	stopWait      sync.WaitGroup
	stopOnce      sync.Once
	stopCh        chan struct{}
	stopWorkersCh chan struct{}
}

// NewBatchItemProcessor creates a new ItemProcessor that will send completed
// item batches to the exporter with the supplied options.
//
// If the exporter is nil, the item processor will preform no action.
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

	metrics := DefaultMetrics

	bvp := BatchItemProcessor[T]{
		e:             exporter,
		o:             o,
		log:           log,
		name:          name,
		metrics:       metrics,
		batch:         make([]*T, 0, o.MaxExportBatchSize),
		timer:         time.NewTimer(o.BatchTimeout),
		queue:         make(chan *T, o.MaxQueueSize),
		stopCh:        make(chan struct{}),
		stopWorkersCh: make(chan struct{}),
	}

	bvp.batches = make(chan []*T, o.Workers) // Buffer the channel to hold batches for each worker
	bvp.batchReady = make(chan bool, 1)

	bvp.stopWait.Add(o.Workers)

	for i := 0; i < o.Workers; i++ {
		go func() {
			defer bvp.stopWait.Done()

			bvp.worker(context.Background())
		}()
	}

	go bvp.batchBuilder(context.Background()) // Start building batches

	return &bvp, nil
}

// OnEnd method enqueues a item for later processing.
func (bvp *BatchItemProcessor[T]) Write(ctx context.Context, s []*T) error {
	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.Write")
	defer span.End()

	bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

	if bvp.o.ShippingMethod == ShippingMethodSync {
		return bvp.ImmediatelyExportItems(ctx, s)
	}

	bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

	// Do not enqueue items if we are just going to drop them.
	if bvp.e == nil {
		return nil
	}

	for _, i := range s {
		bvp.enqueue(i)
	}

	return nil
}

// ImmediatelyExportItems immediately exports the items to the exporter.
// Useful for propogating errors from the exporter.
func (bvp *BatchItemProcessor[T]) ImmediatelyExportItems(ctx context.Context, items []*T) error {
	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.ImmediatelyExportItems")
	defer span.End()

	if l := len(items); l > 0 {
		countItemsToExport := len(items)

		// Split the items in to chunks of our max batch size
		for i := 0; i < countItemsToExport; i += bvp.o.MaxExportBatchSize {
			end := i + bvp.o.MaxExportBatchSize
			if end > countItemsToExport {
				end = countItemsToExport
			}

			itemsBatch := items[i:end]

			bvp.log.WithFields(logrus.Fields{
				"count": len(itemsBatch),
			}).Debug("Immediately exporting items")

			err := bvp.exportWithTimeout(ctx, itemsBatch)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// exportWithTimeout exports the items with a timeout.
func (bvp *BatchItemProcessor[T]) exportWithTimeout(ctx context.Context, itemsBatch []*T) error {
	if bvp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bvp.o.ExportTimeout)

		defer cancel()
	}

	bvp.metrics.IncItemsExportedBy(bvp.name, float64(len(itemsBatch)))

	err := bvp.e.ExportItems(ctx, itemsBatch)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown flushes the queue and waits until all items are processed.
// It only executes once. Subsequent call does nothing.
func (bvp *BatchItemProcessor[T]) Shutdown(ctx context.Context) error {
	var err error

	bvp.log.Debug("Shutting down processor")

	bvp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			// Stop accepting new items
			close(bvp.stopCh)

			// Drain the queue
			bvp.drainQueue()

			// Stop the timer
			bvp.timer.Stop()

			// Stop the workers
			close(bvp.stopWorkersCh)

			// Wait for the workers to finish
			bvp.stopWait.Wait()

			// Shutdown the exporter
			if bvp.e != nil {
				if err = bvp.e.Shutdown(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to shutdown processor")
				}
			}

			close(wait)
		}()
		// Wait until the wait group is done or the context is cancelled
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})

	return err
}

// WithMaxQueueSize returns a BatchItemProcessorOption that configures the
// maximum queue size allowed for a BatchItemProcessor.
func WithMaxQueueSize(size int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize returns a BatchItemProcessorOption that configures
// the maximum export batch size allowed for a BatchItemProcessor.
func WithMaxExportBatchSize(size int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout returns a BatchItemProcessorOption that configures the
// maximum delay allowed for a BatchItemProcessor before it will export any
// held item (whether the queue is full or not).
func WithBatchTimeout(delay time.Duration) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithExportTimeout returns a BatchItemProcessorOption that configures the
// amount of time a BatchItemProcessor waits for an exporter to export before
// abandoning the export.
func WithExportTimeout(timeout time.Duration) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.ExportTimeout = timeout
	}
}

// WithExportTimeout returns a BatchItemProcessorOption that configures the
// amount of time a BatchItemProcessor waits for an exporter to export before
// abandoning the export.
func WithShippingMethod(method ShippingMethod) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.ShippingMethod = method
	}
}

// WithWorkers returns a BatchItemProcessorOption that configures the
// number of workers to process items.
func WithWorkers(workers int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.Workers = workers
	}
}

func (bvp *BatchItemProcessor[T]) batchBuilder(ctx context.Context) {
	for {
		select {
		case <-bvp.stopWorkersCh:
			return
		case sd := <-bvp.queue:
			bvp.batchMutex.Lock()

			bvp.batch = append(bvp.batch, sd)

			if len(bvp.batch) >= bvp.o.MaxExportBatchSize {
				batchCopy := make([]*T, len(bvp.batch))
				copy(batchCopy, bvp.batch)
				bvp.batches <- batchCopy
				bvp.batch = bvp.batch[:0]
				bvp.batchReady <- true
			}

			bvp.batchMutex.Unlock()
		case <-bvp.timer.C:
			bvp.batchMutex.Lock()

			if len(bvp.batch) > 0 {
				batchCopy := make([]*T, len(bvp.batch))
				copy(batchCopy, bvp.batch)
				bvp.batches <- batchCopy
				bvp.batch = bvp.batch[:0]
				bvp.batchReady <- true
			} else {
				// Reset the timer if there are no items in the batch.
				// If there are items in the batch, one of the workers will reset the timer.
				bvp.timer.Reset(bvp.o.BatchTimeout)
			}

			bvp.batchMutex.Unlock()
		}
	}
}

// worker removes items from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bvp *BatchItemProcessor[T]) worker(ctx context.Context) {
	for {
		select {
		case <-bvp.stopWorkersCh:
			return
		case <-bvp.batchReady:
			bvp.timer.Reset(bvp.o.BatchTimeout)

			batch := <-bvp.batches
			if err := bvp.exportWithTimeout(ctx, batch); err != nil {
				bvp.log.WithError(err).Error("failed to export items")
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bvp.stopWait
// to finish the enqueue, then exports the final batch.
func (bvp *BatchItemProcessor[T]) drainQueue() {
	// Wait for the batch builder to send all remaining items to the workers.
	for len(bvp.queue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for the workers to finish processing all batches.
	for len(bvp.batches) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Close the batches channel since no more batches will be sent.
	close(bvp.batches)
}

func (bvp *BatchItemProcessor[T]) enqueue(sd *T) {
	ctx := context.TODO()
	bvp.enqueueOrDrop(ctx, sd)
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

func (bvp *BatchItemProcessor[T]) enqueueOrDrop(ctx context.Context, sd *T) bool {
	// This ensures the bvp.queue<- below does not panic as the
	// processor shuts down.
	defer recoverSendOnClosedChan()

	select {
	case <-bvp.stopCh:
		return false
	default:
	}

	select {
	case bvp.queue <- sd:
		return true
	default:
		atomic.AddUint32(&bvp.dropped, 1)
		bvp.metrics.IncItemsDroppedBy(bvp.name, float64(1))
	}

	return false
}
