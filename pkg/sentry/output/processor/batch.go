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

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

type EventExporter interface {
	// ExportEvents exports a batch of events.
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
	ExportEvents(ctx context.Context, events []*xatu.DecoratedEvent) error

	// Shutdown notifies the exporter of a pending halt to operations. The
	// exporter is expected to preform any cleanup or synchronization it
	// requires while honoring all timeouts and cancellations contained in
	// the passed context.
	Shutdown(ctx context.Context) error
}

// Defaults for BatchDecoratedEventProcessorOptions.
const (
	DefaultMaxQueueSize       = 2048
	DefaultScheduleDelay      = 5000
	DefaultExportTimeout      = 30000
	DefaultMaxExportBatchSize = 512
)

// BatchDecoratedEventProcessorOption configures a BatchDecoratedEventProcessor.
type BatchDecoratedEventProcessorOption func(o *BatchDecoratedEventProcessorOptions)

// BatchDecoratedEventProcessorOptions is configuration settings for a
// BatchDecoratedEventProcessor.
type BatchDecoratedEventProcessorOptions struct {
	// MaxQueueSize is the maximum queue size to buffer events for delayed processing. If the
	// queue gets full it drops the events.
	// The default value of MaxQueueSize is 2048.
	MaxQueueSize int

	// BatchTimeout is the maximum duration for constructing a batch. Processor
	// forcefully sends available events when timeout is reached.
	// The default value of BatchTimeout is 5000 msec.
	BatchTimeout time.Duration

	// ExportTimeout specifies the maximum duration for exporting events. If the timeout
	// is reached, the export will be cancelled.
	// The default value of ExportTimeout is 30000 msec.
	ExportTimeout time.Duration

	// MaxExportBatchSize is the maximum number of events to process in a single batch.
	// If there are more than one batch worth of events then it processes multiple batches
	// of events one batch after the other without any delay.
	// The default value of MaxExportBatchSize is 512.
	MaxExportBatchSize int
}

// BatchDecoratedEventProcessor is a buffer that batches asynchronously-received
// events and sends them to a exporter when complete.
type BatchDecoratedEventProcessor struct {
	e EventExporter
	o BatchDecoratedEventProcessorOptions

	log logrus.FieldLogger

	queue   chan *xatu.DecoratedEvent
	dropped uint32

	batch      []*xatu.DecoratedEvent
	batchMutex sync.Mutex
	timer      *time.Timer
	stopWait   sync.WaitGroup
	stopOnce   sync.Once
	stopCh     chan struct{}
}

// NewBatchDecoratedEventProcessor creates a new DecoratedEventProcessor that will send completed
// event batches to the exporter with the supplied options.
//
// If the exporter is nil, the event processor will preform no action.
func NewBatchDecoratedEventProcessor(exporter EventExporter, log logrus.FieldLogger, options ...BatchDecoratedEventProcessorOption) *BatchDecoratedEventProcessor {
	maxQueueSize := DefaultMaxQueueSize
	maxExportBatchSize := DefaultMaxExportBatchSize

	if maxExportBatchSize > maxQueueSize {
		if DefaultMaxExportBatchSize > maxQueueSize {
			maxExportBatchSize = maxQueueSize
		} else {
			maxExportBatchSize = DefaultMaxExportBatchSize
		}
	}

	o := BatchDecoratedEventProcessorOptions{
		BatchTimeout:       time.Duration(DefaultScheduleDelay) * time.Millisecond,
		ExportTimeout:      time.Duration(DefaultExportTimeout) * time.Millisecond,
		MaxQueueSize:       maxQueueSize,
		MaxExportBatchSize: maxExportBatchSize,
	}
	for _, opt := range options {
		opt(&o)
	}

	bvp := &BatchDecoratedEventProcessor{
		e:      exporter,
		o:      o,
		log:    log,
		batch:  make([]*xatu.DecoratedEvent, 0, o.MaxExportBatchSize),
		timer:  time.NewTimer(o.BatchTimeout),
		queue:  make(chan *xatu.DecoratedEvent, o.MaxQueueSize),
		stopCh: make(chan struct{}),
	}

	bvp.stopWait.Add(1)

	go func() {
		defer bvp.stopWait.Done()
		bvp.processQueue()
		bvp.drainQueue()
	}()

	return bvp
}

// OnEnd method enqueues a *xatu.DecoratedEvent for later processing.
func (bvp *BatchDecoratedEventProcessor) Write(s *xatu.DecoratedEvent) {
	// Do not enqueue events if we are just going to drop them.
	if bvp.e == nil {
		return
	}

	bvp.enqueue(s)
}

// Shutdown flushes the queue and waits until all events are processed.
// It only executes once. Subsequent call does nothing.
func (bvp *BatchDecoratedEventProcessor) Shutdown(ctx context.Context) error {
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

// ForceFlush exports all ended events that have not yet been exported.
func (bvp *BatchDecoratedEventProcessor) ForceFlush(ctx context.Context) error {
	var err error

	if bvp.e != nil {
		wait := make(chan error)

		go func() {
			wait <- bvp.exportEvents(ctx)
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

// WithMaxQueueSize returns a BatchDecoratedEventProcessorOption that configures the
// maximum queue size allowed for a BatchDecoratedEventProcessor.
func WithMaxQueueSize(size int) BatchDecoratedEventProcessorOption {
	return func(o *BatchDecoratedEventProcessorOptions) {
		o.MaxQueueSize = size
	}
}

// WithMaxExportBatchSize returns a BatchDecoratedEventProcessorOption that configures
// the maximum export batch size allowed for a BatchDecoratedEventProcessor.
func WithMaxExportBatchSize(size int) BatchDecoratedEventProcessorOption {
	return func(o *BatchDecoratedEventProcessorOptions) {
		o.MaxExportBatchSize = size
	}
}

// WithBatchTimeout returns a BatchDecoratedEventProcessorOption that configures the
// maximum delay allowed for a BatchDecoratedEventProcessor before it will export any
// held event (whether the queue is full or not).
func WithBatchTimeout(delay time.Duration) BatchDecoratedEventProcessorOption {
	return func(o *BatchDecoratedEventProcessorOptions) {
		o.BatchTimeout = delay
	}
}

// WithExportTimeout returns a BatchDecoratedEventProcessorOption that configures the
// amount of time a BatchDecoratedEventProcessor waits for an exporter to export before
// abandoning the export.
func WithExportTimeout(timeout time.Duration) BatchDecoratedEventProcessorOption {
	return func(o *BatchDecoratedEventProcessorOptions) {
		o.ExportTimeout = timeout
	}
}

// exportEvents is a subroutine of processing and draining the queue.
func (bvp *BatchDecoratedEventProcessor) exportEvents(ctx context.Context) error {
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
		}).Debug("exporting events")

		err := bvp.e.ExportEvents(ctx, bvp.batch)

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

// processQueue removes events from the `queue` channel until processor
// is shut down. It calls the exporter in batches of up to MaxExportBatchSize
// waiting up to BatchTimeout to form a batch.
func (bvp *BatchDecoratedEventProcessor) processQueue() {
	defer bvp.timer.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-bvp.stopCh:
			return
		case <-bvp.timer.C:
			if err := bvp.exportEvents(ctx); err != nil {
				bvp.log.WithError(err).Error("failed to export events")
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

				if err := bvp.exportEvents(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export events")
				}
			}
		}
	}
}

// drainQueue awaits the any caller that had added to bvp.stopWait
// to finish the enqueue, then exports the final batch.
func (bvp *BatchDecoratedEventProcessor) drainQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case sd := <-bvp.queue:
			if sd == nil {
				if err := bvp.exportEvents(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export events")
				}

				return
			}

			bvp.batchMutex.Lock()
			bvp.batch = append(bvp.batch, sd)
			shouldExport := len(bvp.batch) == bvp.o.MaxExportBatchSize
			bvp.batchMutex.Unlock()

			if shouldExport {
				if err := bvp.exportEvents(ctx); err != nil {
					bvp.log.WithError(err).Error("failed to export events")
				}
			}
		default:
			close(bvp.queue)
		}
	}
}

func (bvp *BatchDecoratedEventProcessor) enqueue(sd *xatu.DecoratedEvent) {
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

func (bvp *BatchDecoratedEventProcessor) enqueueDrop(ctx context.Context, sd *xatu.DecoratedEvent) bool {
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
