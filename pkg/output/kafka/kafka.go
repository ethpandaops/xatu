package kafka

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const SinkType = "kafka"

// Kafka is a sink that writes decorated events to Apache Kafka.
type Kafka struct {
	name   string
	config *Config
	log    logrus.FieldLogger
	proc   *processor.BatchItemProcessor[xatu.DecoratedEvent]
	filter xatu.EventFilter
}

// New creates a new Kafka sink. It builds the Sarama SyncProducer and
// wires it into the ItemExporter used by the batch processor.
func New(name string, config *Config, log logrus.FieldLogger, filterConfig *xatu.EventFilterConfig, shippingMethod processor.ShippingMethod) (*Kafka, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	producer, err := NewSyncProducer(&config.ProducerConfig)
	if err != nil {
		log.
			WithError(err).
			WithField("output_name", name).
			WithField("output_type", SinkType).
			Error("Error while creating the Kafka Client")

		return nil, err
	}

	exporter := NewItemExporter(name, config, log, producer)

	filter, err := xatu.NewEventFilter(filterConfig)
	if err != nil {
		return nil, err
	}

	proc, err := processor.NewBatchItemProcessor[xatu.DecoratedEvent](
		exporter,
		xatu.ImplementationLower()+"_output_"+SinkType+"_"+name,
		log,
		processor.WithMaxQueueSize(config.MaxQueueSize),
		processor.WithBatchTimeout(config.BatchTimeout),
		processor.WithMaxExportBatchSize(config.MaxExportBatchSize),
		processor.WithShippingMethod(shippingMethod),
		processor.WithWorkers(config.Workers),
	)
	if err != nil {
		return nil, err
	}

	return &Kafka{
		name:   name,
		config: config,
		log:    log,
		proc:   proc,
		filter: filter,
	}, nil
}

// Name returns the configured name of this sink.
func (h *Kafka) Name() string {
	return h.name
}

// Type returns the sink type identifier.
func (h *Kafka) Type() string {
	return SinkType
}

// Start begins the batch processor.
func (h *Kafka) Start(ctx context.Context) error {
	h.proc.Start(ctx)

	return nil
}

// Stop shuts down the batch processor.
func (h *Kafka) Stop(ctx context.Context) error {
	return h.proc.Shutdown(ctx)
}

// HandleNewDecoratedEvent processes a single event through the filter and batch processor.
func (h *Kafka) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	shouldBeDropped, err := h.filter.ShouldBeDropped(event)
	if err != nil {
		return err
	}

	if shouldBeDropped {
		return nil
	}

	return h.proc.Write(ctx, []*xatu.DecoratedEvent{event})
}

// HandleNewDecoratedEvents processes a batch of events through the filter and batch processor.
func (h *Kafka) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	filtered := make([]*xatu.DecoratedEvent, 0, len(events))

	for _, event := range events {
		shouldBeDropped, err := h.filter.ShouldBeDropped(event)
		if err != nil {
			return err
		}

		if !shouldBeDropped {
			filtered = append(filtered, event)
		}
	}

	return h.proc.Write(ctx, filtered)
}
