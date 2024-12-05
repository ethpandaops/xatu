package processor

import (
	"errors"
	"time"
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
