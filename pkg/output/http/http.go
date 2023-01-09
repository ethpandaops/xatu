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
	config *Config
	log    logrus.FieldLogger
	proc   *processor.BatchItemProcessor[xatu.DecoratedEvent]
}

func New(config *Config, log logrus.FieldLogger) (*HTTP, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	exporter, err := NewItemExporter(config, log)
	if err != nil {
		return nil, err
	}

	proc := processor.NewBatchItemProcessor[xatu.DecoratedEvent](exporter,
		log,
		processor.WithMaxQueueSize(config.MaxQueueSize),
		processor.WithBatchTimeout(config.BatchTimeout),
		processor.WithExportTimeout(config.ExportTimeout),
		processor.WithMaxExportBatchSize(config.MaxExportBatchSize),
	)

	return &HTTP{
		config: config,
		log:    log,
		proc:   proc,
	}, nil
}

func (h *HTTP) Type() string {
	return SinkType
}

func (h *HTTP) Start(ctx context.Context) error {
	return nil
}

func (h *HTTP) Stop(ctx context.Context) error {
	return h.proc.Shutdown(ctx)
}

func (h *HTTP) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	h.proc.Write(event)

	return nil
}
