package processor

import (
	"context"
	"errors"
	"fmt"
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

const (
	DefaultMaxQueueSize       = 51200
	DefaultScheduleDelay      = 5000
	DefaultExportTimeout      = 30000
	DefaultMaxExportBatchSize = 512
	DefaultShippingMethod     = ShippingMethodAsync
	DefaultNumWorkers         = 5
)

// ShippingMethod is the method of shipping items for export.
type ShippingMethod string

const (
	ShippingMethodUnknown ShippingMethod = "unknown"
	ShippingMethodAsync   ShippingMethod = "async"
	ShippingMethodSync    ShippingMethod = "sync"
)

// BatchItemProcessorOption is a functional option for the batch item processor.
type BatchItemProcessorOption func(o *BatchItemProcessorOptions)

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
	// MaxExportBatchSize is the maximum number of items to include in a batch.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int
	// ShippingMethod is the method of shipping items for export. The default value
	// of ShippingMethod is "async".
	ShippingMethod ShippingMethod
	// Workers is the number of workers to process batches.
	// The default value of Workers is 5.
	Workers int
	// Metrics is the metrics instance to use.
	Metrics *Metrics
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

// exportWithTimeout exports items with a timeout.
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

	if bvp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bvp.o.ExportTimeout)

		defer cancel()
	}

	// Since the batch processor filters out nil items upstream,
	// we can optimize by pre-allocating the full slice size.
	// Worst case is a few wasted allocations if any nil items slip through.
	items := make([]*T, len(itemsBatch))

	for i, item := range itemsBatch {
		if item == nil {
			bvp.log.Warnf("Attempted to export a nil item. This item has been dropped. This probably shouldn't happen and is likely a bug.")

			continue
		}

		items[i] = item.item
	}

	startTime := time.Now()

	err := bvp.e.ExportItems(ctx, items)

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

	return nil
}

// Shutdown shuts down the batch item processor.
func (bvp *BatchItemProcessor[T]) Shutdown(ctx context.Context) error {
	var err error

	bvp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			bvp.log.Info("Stopping processor")

			close(bvp.stopCh)

			bvp.timer.Stop()

			bvp.drainQueue()

			close(bvp.stopWorkersCh)

			bvp.stopWait.Wait()

			if bvp.e != nil {
				if err = bvp.e.Shutdown(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to shutdown processor")
				}
			}

			close(wait)
		}()
		select {
		case <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})

	return err
}

func WithMaxQueueSize(size int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.MaxQueueSize = size
	}
}

func WithMaxExportBatchSize(size int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

func WithBatchTimeout(delay time.Duration) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.BatchTimeout = delay
	}
}

func WithExportTimeout(timeout time.Duration) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.ExportTimeout = timeout
	}
}

func WithShippingMethod(method ShippingMethod) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.ShippingMethod = method
	}
}

func WithWorkers(workers int) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.Workers = workers
	}
}

func WithMetrics(metrics *Metrics) BatchItemProcessorOption {
	return func(o *BatchItemProcessorOptions) {
		o.Metrics = metrics
	}
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
		case <-bvp.stopWorkersCh:
			bvp.log.Infof("Stopping worker %d", number)

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

func (bvp *BatchItemProcessor[T]) drainQueue() {
	bvp.log.Info("Draining queue: waiting for the batch builder to process remaining items")

	// First wait for queue to be processed
	for len(bvp.queue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	bvp.log.Info("Draining queue: waiting for workers to finish processing batches")

	// Then wait for any in-flight batches
	for len(bvp.batchCh) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	bvp.log.Info("Draining queue: all items processed")

	close(bvp.queue)
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
