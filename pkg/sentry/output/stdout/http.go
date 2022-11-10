package stdout

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/output/buffer"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

const SinkType = "StdOut"

type StdOut struct {
	config *Config
	log    logrus.FieldLogger
	buf    *buffer.DecoratedEventBuffer
}

func New(config *Config, log logrus.FieldLogger) (*StdOut, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	buf := buffer.NewDecoratedEventBuffer(100000)

	return &StdOut{
		config: config,
		log:    log,
		buf:    buf,
	}, nil
}

func (h *StdOut) Type() string {
	return SinkType
}

func (h *StdOut) Start(ctx context.Context) error {
	h.buf.OnAdded(func(ctx context.Context, event *xatu.DecoratedEvent) {
		if h.buf.Len() >= 1 {
			if err := h.send(ctx); err != nil {
				h.log.WithError(err).Error("Failed to send batch of events to StdOut sink")
			}
		}
	})

	return nil
}

func (h *StdOut) send(ctx context.Context) error {
	// Pull as many items out of the buffer as we can
	events := []*xatu.DecoratedEvent{}

	for event := range h.buf.Read() {
		events = append(events, event)

		if len(events) == 1 {
			break
		}
	}

	h.log.WithField("events", len(events)).Info("Sending batch of events to StdOut sink")

	if err := h.sendUpstream(ctx, events); err != nil {
		return err
	}

	return nil
}

func (h *StdOut) sendUpstream(ctx context.Context, events []*xatu.DecoratedEvent) error {
	for _, event := range events {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		h.log.WithField("event", string(eventAsJSON)).Info("stdout sink event")
	}

	return nil
}

func (h *StdOut) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	h.buf.Write(ctx, event)

	return nil
}
