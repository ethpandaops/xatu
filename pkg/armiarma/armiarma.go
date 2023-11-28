package armiarma

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"github.com/r3labs/sse/v2"
	"github.com/sirupsen/logrus"
)

type Armiarma struct {
	log         logrus.FieldLogger
	config      *Config
	eventStream *sse.Client
	sinks       []output.Sink
}

func New(log logrus.FieldLogger, config *Config) (*Armiarma, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	return &Armiarma{
		log:         log.WithField("module", "armiarma"),
		config:      config,
		eventStream: sse.NewClient(config.ArmiarmaURL),
	}, nil
}

func (a *Armiarma) Start(ctx context.Context) error {
	a.log.
		WithField("version", xatu.Full()).
		Info("Starting Xatu in armiarma mode")

	a.subscribeToEvents(ctx)

	return nil
}

func (a *Armiarma) subscribeToEvents(ctx context.Context) error {
	a.eventStream.SubscribeWithContext(ctx, "attestations", func(msg *sse.Event) {
		a.log.WithField("event", "attestations").Debug(msg.Data)
	})

	return nil
}
