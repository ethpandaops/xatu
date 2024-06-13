package processor

import (
	"context"
	"errors"
	"fmt"
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

func (o *BatchItemProcessorOptions) Validate() error {
	if o.MaxExportBatchSize > o.MaxQueueSize {
		return errors.New("max export batch size cannot be greater than max queue size")
	}

	if o.Workers == 0 {
		return errors.New("workers must be greater than 0")
	}

	if o.MaxExportBatchSize < 1 {
		return errors.New("max export batch size must be greater than 0")
	}

	return nil
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

	batches    chan batchWithErr[T]
	batchReady chan struct{}

	batch      []*T
	batchMutex sync.Mutex

	timer         *time.Timer
	stopWait      sync.WaitGroup
	stopOnce      sync.Once
	stopCh        chan struct{}
	stopWorkersCh chan struct{}
}

type batchWithErr[T any] struct {
	items       []*T
	errCh       chan error
	completedCh chan struct{}
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

	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("invalid batch item processor options: %w: %s", err, name)
	}

	metrics := DefaultMetrics

	bvp := BatchItemProcessor[T]{
		e:             exporter,
		o:             o,
		log:           log,
		name:          name,
		metrics:       metrics,
		batch:         make([]*T, 0, o.MaxQueueSize),
		timer:         time.NewTimer(o.BatchTimeout),
		queue:         make(chan *T, o.MaxQueueSize),
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

	bvp.batches = make(chan batchWithErr[T], o.Workers)
	bvp.batchReady = make(chan struct{}, 1)

	bvp.stopWait.Add(o.Workers)

	for i := 0; i < o.Workers; i++ {
		go func(num int) {
			defer bvp.stopWait.Done()
			bvp.worker(context.Background(), num)
		}(i)
	}

	go func() {
		bvp.batchBuilder(context.Background()) // Start building batches
		bvp.log.Info("Batch builder exited")
	}()

	return &bvp, nil
}

func (bvp *BatchItemProcessor[T]) Write(ctx context.Context, s []*T) error {
	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.Write")
	defer span.End()

	bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

	if bvp.o.ShippingMethod == ShippingMethodSync {
		return bvp.immediatelyExportItems(ctx, s)
	}

	bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

	if bvp.e == nil {
		return errors.New("exporter is nil")
	}

	for _, i := range s {
		bvp.enqueue(i)
	}

	return nil
}

// immediatelyExportItems immediately exports the items to the exporter.
// Useful for propagating errors from the exporter.
func (bvp *BatchItemProcessor[T]) immediatelyExportItems(ctx context.Context, items []*T) error {
	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.ImmediatelyExportItems")
	defer span.End()

	if len(items) == 0 {
		return nil
	}

	batchSize := bvp.o.MaxExportBatchSize
	if batchSize == 0 {
		batchSize = 1
	}

	errCh := make(chan error, bvp.o.Workers)
	completedCh := make(chan struct{}, bvp.o.Workers)
	pendingExports := 0

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		itemsBatch := items[i:end]
		bvp.batches <- batchWithErr[T]{items: itemsBatch, errCh: errCh, completedCh: completedCh}
		bvp.batchReady <- struct{}{}

		pendingExports++

		// Only create a new batch if there are already bvp.o.Workers pending exports.
		// We do this so we don't bother queueing up any more batches if
		// a previous batch has failed.
		if pendingExports >= bvp.o.Workers {
			select {
			case batchErr := <-errCh:
				pendingExports--

				if batchErr != nil {
					return batchErr
				}
			case <-completedCh:
				pendingExports--
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	// Wait for remaining pending exports to complete
	for pendingExports > 0 {
		select {
		case batchErr := <-errCh:
			pendingExports--

			if batchErr != nil {
				return batchErr
			}
		case <-completedCh:
			pendingExports--
		case <-ctx.Done():
			return ctx.Err()
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

	err := bvp.e.ExportItems(ctx, itemsBatch)
	if err != nil {
		bvp.metrics.IncItemsFailedBy(bvp.name, float64(len(itemsBatch)))

		return err
	}

	bvp.metrics.IncItemsExportedBy(bvp.name, float64(len(itemsBatch)))

	return nil
}

// Shutdown flushes the queue and waits until all items are processed.
// It only executes once. Subsequent call does nothing.
func (bvp *BatchItemProcessor[T]) Shutdown(ctx context.Context) error {
	var err error

	bvp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			bvp.log.Info("Stopping processor")

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
			bvp.log.Info("Stopping batch builder")

			return
		case sd := <-bvp.queue:
			bvp.log.Info("New item added to queue")

			bvp.batchMutex.Lock()
			bvp.batch = append(bvp.batch, sd)

			if len(bvp.batch) >= bvp.o.MaxExportBatchSize {
				bvp.sendBatch("max_export_batch_size")
			}
			bvp.batchMutex.Unlock()
		case <-bvp.timer.C:
			bvp.batchMutex.Lock()
			bvp.log.Info("Batch timeout reached")

			if len(bvp.batch) > 0 {
				bvp.sendBatch("timer")
			} else {
				bvp.log.Info("No items in batch, resetting timer")
				bvp.timer.Reset(bvp.o.BatchTimeout)
			}
			bvp.batchMutex.Unlock()
		}
	}
}

func (bvp *BatchItemProcessor[T]) sendBatch(reason string) {
	log := bvp.log.WithField("reason", reason)
	log.Infof("Creating a batch of %d items", len(bvp.batch))

	batchCopy := make([]*T, len(bvp.batch))
	log.Infof("Copying batch items")
	copy(batchCopy, bvp.batch)
	log.Infof("Batch items copied")

	bvp.batches <- batchWithErr[T]{items: batchCopy, errCh: make(chan error), completedCh: make(chan struct{})}
	log.Infof("Batch sent to batches channel")

	bvp.batch = bvp.batch[:0]
	log.Infof("Batch cleared")

	bvp.batchReady <- struct{}{}
	log.Infof("Signaled batchReady")
}

// worker removes items from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bvp *BatchItemProcessor[T]) worker(ctx context.Context, number int) {
	bvp.log.Infof("Starting worker %d", number)

	for {
		select {
		case <-bvp.stopWorkersCh:
			bvp.log.Infof("Stopping worker %d", number)

			return
		case <-bvp.batchReady:
			bvp.log.Infof("Worker %d is processing a batch", number)

			bvp.timer.Reset(bvp.o.BatchTimeout)
			batch := <-bvp.batches

			bvp.log.Infof("Worker %d is exporting a batch of %d items", number, len(batch.items))

			if err := bvp.exportWithTimeout(ctx, batch.items); err != nil {
				bvp.log.WithError(err).Error("failed to export items")

				if batch.errCh != nil {
					batch.errCh <- err
				}
			}

			bvp.log.Infof("Batch completed by worker %d", number)

			batch.completedCh <- struct{}{}
		}
	}
}

// drainQueue awaits the any caller that had added to bvp.stopWait
// to finish the enqueue, then exports the final batch.
func (bvp *BatchItemProcessor[T]) drainQueue() {
	bvp.log.Info("Draining queue: waiting for the batch builder to pull all the items from the queue")

	// Wait for the batch builder to send all remaining items to the workers.
	for len(bvp.queue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	bvp.log.Info("Draining queue: waiting for workers to finish processing batches")

	// Wait for the workers to finish processing all batches.
	for len(bvp.batches) > 0 {
		batch := <-bvp.batches
		<-batch.completedCh
	}

	bvp.log.Info("Draining queue: all batches finished")

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
