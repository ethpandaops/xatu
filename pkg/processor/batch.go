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
	DefaultNumWorkers         = 1
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
	// The default value of Workers is 1.
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

// BatchItemProcessor is a processor that batches items for export.
type BatchItemProcessor[T any] struct {
	e ItemExporter[T]
	o BatchItemProcessorOptions

	log logrus.FieldLogger

	queue   chan traceableItem[T]
	batchCh chan []traceableItem[T]
	dropped uint32
	name    string

	metrics *Metrics

	timer         *time.Timer
	stopWait      sync.WaitGroup
	stopOnce      sync.Once
	stopCh        chan struct{}
	stopWorkersCh chan struct{}
}

type traceableItem[T any] struct {
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

	metrics := DefaultMetrics

	bvp := BatchItemProcessor[T]{
		e:             exporter,
		o:             o,
		log:           log,
		name:          name,
		metrics:       metrics,
		timer:         time.NewTimer(o.BatchTimeout),
		queue:         make(chan traceableItem[T], o.MaxQueueSize),
		batchCh:       make(chan []traceableItem[T], o.Workers),
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

	bvp.stopWait.Add(o.Workers)

	for i := 0; i < o.Workers; i++ {
		go func(num int) {
			defer bvp.stopWait.Done()
			bvp.worker(context.Background(), num)
		}(i)
	}

	go func() {
		bvp.batchBuilder(context.Background())
		bvp.log.Info("Batch builder exited")
	}()

	return &bvp, nil
}

// Write writes items to the queue. If the Processor is configured to use
// the sync shipping method, the items will be written to the queue and this
// function will return when all items have been processed. If the Processor is
// configured to use the async shipping method, the items will be written to
// the queue and this function will return immediately.
func (bvp *BatchItemProcessor[T]) Write(ctx context.Context, s []*T) error {
	_, span := observability.Tracer().Start(ctx, "BatchItemProcessor.Write")
	defer span.End()

	bvp.metrics.SetItemsQueued(bvp.name, float64(len(bvp.queue)))

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

		prepared := []traceableItem[T]{}
		for _, i := range s[start:end] {
			prepared = append(prepared, traceableItem[T]{
				item:        i,
				errCh:       make(chan error, 1),
				completedCh: make(chan struct{}, 1),
			})
		}

		for _, i := range prepared {
			if err := bvp.enqueueOrDrop(ctx, i); err != nil {
				return err
			}
		}

		if bvp.o.ShippingMethod == ShippingMethodSync {
			for _, item := range prepared {
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
		}
	}

	return nil
}

// exportWithTimeout exports items with a timeout.
func (bvp *BatchItemProcessor[T]) exportWithTimeout(ctx context.Context, itemsBatch []traceableItem[T]) error {
	if bvp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bvp.o.ExportTimeout)

		defer cancel()
	}

	items := make([]*T, len(itemsBatch))
	for i, item := range itemsBatch {
		items[i] = item.item
	}

	err := bvp.e.ExportItems(ctx, items)
	if err != nil {
		bvp.metrics.IncItemsFailedBy(bvp.name, float64(len(itemsBatch)))
	} else {
		bvp.metrics.IncItemsExportedBy(bvp.name, float64(len(itemsBatch)))
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

func (bvp *BatchItemProcessor[T]) batchBuilder(ctx context.Context) {
	log := bvp.log.WithField("module", "batch_builder")

	var batch []traceableItem[T]

	for {
		select {
		case <-bvp.stopWorkersCh:
			log.Info("Stopping batch builder")

			return
		case item := <-bvp.queue:
			batch = append(batch, item)

			if len(batch) >= bvp.o.MaxExportBatchSize {
				bvp.sendBatch(batch, "max_export_batch_size")

				batch = []traceableItem[T]{}
			}
		case <-bvp.timer.C:
			if len(batch) > 0 {
				bvp.sendBatch(batch, "timer")
				batch = []traceableItem[T]{}
			} else {
				bvp.timer.Reset(bvp.o.BatchTimeout)
			}
		}
	}
}
func (bvp *BatchItemProcessor[T]) sendBatch(batch []traceableItem[T], reason string) {
	log := bvp.log.WithField("reason", reason)
	log.Tracef("Creating a batch of %d items", len(batch))

	batchCopy := make([]traceableItem[T], len(batch))
	copy(batchCopy, batch)

	log.Tracef("Batch items copied")

	bvp.batchCh <- batchCopy

	log.Tracef("Batch sent to batch channel")
}

func (bvp *BatchItemProcessor[T]) worker(ctx context.Context, number int) {
	bvp.log.Infof("Starting worker %d", number)

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
		}
	}
}

func (bvp *BatchItemProcessor[T]) drainQueue() {
	bvp.log.Info("Draining queue: waiting for the batch builder to pull all the items from the queue")

	for len(bvp.queue) > 0 {
		time.Sleep(10 * time.Millisecond)
	}

	bvp.log.Info("Draining queue: waiting for workers to finish processing batches")

	for len(bvp.queue) > 0 {
		<-bvp.queue
	}

	bvp.log.Info("Draining queue: all batches finished")

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

func (bvp *BatchItemProcessor[T]) enqueueOrDrop(ctx context.Context, item traceableItem[T]) error {
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
		return nil
	default:
		atomic.AddUint32(&bvp.dropped, 1)
		bvp.metrics.IncItemsDroppedBy(bvp.name, float64(1))
	}

	return errors.New("queue is full")
}
