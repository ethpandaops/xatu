package stdout

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/output/processor"
	"github.com/sirupsen/logrus"
)

const SinkType = "stdout"

type StdOut struct {
	config *Config
	log    logrus.FieldLogger
	proc   *processor.BatchDecoratedEventProcessor
}

func New(config *Config, log logrus.FieldLogger) (*StdOut, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	exporter, err := NewEventExporter(config, log)
	if err != nil {
		return nil, err
	}

	proc := processor.NewBatchDecoratedEventProcessor(exporter, log)

	return &StdOut{
		config: config,
		log:    log,
		proc:   proc,
	}, nil
}

func (h *StdOut) Type() string {
	return SinkType
}

func (h *StdOut) Start(ctx context.Context) error {
	return nil
}

func (h *StdOut) Stop(ctx context.Context) error {
	return nil
}

func (h *StdOut) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	h.proc.Write(event)

	return nil
}
