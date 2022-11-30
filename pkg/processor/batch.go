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
}

// BatchItemProcessor is a buffer that batches asynchronously-received
// items and sends them to a exporter when complete.
type BatchItemProcessor[T any] struct {
	e ItemExporter[T]
	o BatchItemProcessorOptions

	log logrus.FieldLogger

	queue   chan *T
	dropped uint32

	batch      []*T
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}
}

// NewBatchItemProcessor creates a new ItemProcessor that will send completed
// item batches to the exporter with the supplied options.
//
// If the exporter is nil, the item processor will preform no action.
func NewBatchItemProcessor[T any](exporter ItemExporter[T], log logrus.FieldLogger, options ...BatchItemProcessorOption) *BatchItemProcessor[T] {
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
	}
	for _, opt := range options {
		opt(&o)
	}

	bvp := BatchItemProcessor[T]{
		e:      exporter,
		o:      o,
		log:    log,
		batch:  make([]*T, 0, o.MaxExportBatchSize),
		timer:  time.NewTimer(o.BatchTimeout),
		queue:  make(chan *T, o.MaxQueueSize),
		stopCh: make(chan struct{}),
	}

	bvp.stopWait.Add(1)

	go func() {
		defer bvp.stopWait.Done()
		bvp.processQueue()
		bvp.drainQueue()
	}()

	return &bvp
}

// OnEnd method enqueues a item for later processing.
func (bvp *BatchItemProcessor[T]) Write(s *T) {
	// Do not enqueue items if we are just going to drop them.
	if bvp.e == nil {
		return
	}

	bvp.enqueue(s)
}

// Shutdown flushes the queue and waits until all items are processed.
// It only executes once. Subsequent call does nothing.
func (bvp *BatchItemProcessor[T]) Shutdown(ctx context.Context) error {
	var err error

	bvp.stopOnce.Do(func() {
		wait := make(chan struct{})
		go func() {
			close(bvp.stopCh)
			bvp.stopWait.Wait()
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

// ForceFlush exports all ended items that have not yet been exported.
func (bvp *BatchItemProcessor[T]) ForceFlush(ctx context.Context) error {
	var err error

	if bvp.e != nil {
		wait := make(chan error)

		go func() {
			wait <- bvp.exportItems(ctx)
			close(wait)
		}()

		// Wait until the export is finished or the context is cancelled/timed out
		select {
		case err = <-wait:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

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

// exportItems is a subroutine of processing and draining the queue.
func (bvp *BatchItemProcessor[T]) exportItems(ctx context.Context) error {
	bvp.timer.Reset(bvp.o.BatchTimeout)

	bvp.batchMutex.Lock()
	defer bvp.batchMutex.Unlock()

	if bvp.o.ExportTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, bvp.o.ExportTimeout)

		defer cancel()
	}

	if l := len(bvp.batch); l > 0 {
		bvp.log.WithFields(logrus.Fields{
			"count":         len(bvp.batch),
			"total_dropped": atomic.LoadUint32(&bvp.dropped),
		}).Debug("exporting items")

		err := bvp.e.ExportItems(ctx, bvp.batch)

		// A new batch is always created after exporting, even if the batch failed to be exported.
		//
		// It is up to the exporter to implement any type of retry logic if a batch is failing
		// to be exported, since it is specific to the protocol and backend being sent to.
		bvp.batch = bvp.batch[:0]

		if err != nil {
			return err
		}
	}

	return nil
}

// processQueue removes items from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bvp *BatchItemProcessor[T]) processQueue() {
	defer bvp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-bvp.stopCh:
			return
		case <-bvp.timer.C:
			if err := bvp.exportItems(ctx); err != nil {
				bvp.log.WithError(err).Error("failed to export items")
			}
		case sd := <-bvp.queue:
			bvp.batchMutex.Lock()
			bvp.batch = append(bvp.batch, sd)
			shouldExport := len(bvp.batch) >= bvp.o.MaxExportBatchSize
			bvp.batchMutex.Unlock()

			if shouldExport {
				if !bvp.timer.Stop() {
					<-bvp.timer.C
				}

				if err := bvp.exportItems(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export items")
				}
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bvp.stopWait
// to finish the enqueue, then exports the final batch.
func (bvp *BatchItemProcessor[T]) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case sd := <-bvp.queue:
			if sd == nil {
				if err := bvp.exportItems(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export items")
				}

				return
			}

			bvp.batchMutex.Lock()
			bvp.batch = append(bvp.batch, sd)
			shouldExport := len(bvp.batch) == bvp.o.MaxExportBatchSize
			bvp.batchMutex.Unlock()

			if shouldExport {
				if err := bvp.exportItems(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export items")
				}
			}
		default:
			close(bvp.queue)
		}
	}
}

func (bvp *BatchItemProcessor[T]) enqueue(sd *T) {
	ctx := context.TODO()
	bvp.enqueueDrop(ctx, sd)
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

func (bvp *BatchItemProcessor[T]) enqueueDrop(ctx context.Context, sd *T) bool {
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
	}

	return false
}
