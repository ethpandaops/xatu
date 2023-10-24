package http

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const SinkType = "http"

type HTTP struct {
	name   string
	config *Config
	log    logrus.FieldLogger
	proc   *processor.BatchItemProcessor[xatu.DecoratedEvent]
	filter xatu.EventFilter
}

func New(name string, config *Config, log logrus.FieldLogger, filterConfig *xatu.EventFilterConfig, shippingMethod processor.ShippingMethod) (*HTTP, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	exporter, err := NewItemExporter(name, config, log)
	if err != nil {
		return nil, err
	}

	filter, err := xatu.NewEventFilter(filterConfig)
	if err != nil {
		return nil, err
	}

	proc, err := processor.NewBatchItemProcessor[xatu.DecoratedEvent](exporter,
		xatu.ImplementationLower()+"_output_"+SinkType+"_"+name,
		log,
		processor.WithMaxQueueSize(config.MaxQueueSize),
		processor.WithBatchTimeout(config.BatchTimeout),
		processor.WithExportTimeout(config.ExportTimeout),
		processor.WithMaxExportBatchSize(config.MaxExportBatchSize),
		processor.WithShippingMethod(shippingMethod),
		processor.WithWorkers(config.Workers),
	)
	if err != nil {
		return nil, err
	}

	return &HTTP{
		name:   name,
		config: config,
		log:    log,
		proc:   proc,
		filter: filter,
	}, nil
}

func (h *HTTP) Type() string {
	return SinkType
}

func (h *HTTP) Start(ctx context.Context) error {
	return nil
}

func (h *HTTP) Name() string {
	return h.name
}

func (h *HTTP) Stop(ctx context.Context) error {
	return h.proc.Shutdown(ctx)
}

func (h *HTTP) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	shouldBeDropped, err := h.filter.ShouldBeDropped(event)
	if err != nil {
		return err
	}

	if shouldBeDropped {
		return nil
	}

	return h.proc.Write(ctx, []*xatu.DecoratedEvent{event})
}

func (h *HTTP) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	filtered := []*xatu.DecoratedEvent{}

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
