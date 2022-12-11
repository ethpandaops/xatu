package coordinator

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/sirupsen/logrus"
)

const SinkType = "xatu"

type Client struct {
	config *Config
	log    logrus.FieldLogger
	proc   *processor.BatchItemProcessor[string]
}

func New(config *Config, log logrus.FieldLogger) (*Client, error) {
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

	proc := processor.NewBatchItemProcessor[string](exporter,
		log,
		processor.WithMaxQueueSize(config.MaxQueueSize),
		processor.WithBatchTimeout(config.BatchTimeout),
		processor.WithExportTimeout(config.ExportTimeout),
		processor.WithMaxExportBatchSize(config.MaxExportBatchSize),
	)

	return &Client{
		config: config,
		log:    log,
		proc:   proc,
	}, nil
}

func (h *Client) Type() string {
	return SinkType
}

func (h *Client) Start(ctx context.Context) error {
	return nil
}

func (h *Client) Stop(ctx context.Context) error {
	return h.proc.Shutdown(ctx)
}

func (h *Client) HandleNewNodeRecord(ctx context.Context, record *string) error {
	h.proc.Write(record)

	return nil
}
