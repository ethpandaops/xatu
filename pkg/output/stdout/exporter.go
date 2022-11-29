package stdout

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

type EventExporter struct {
	config *Config
	log    logrus.FieldLogger
}

func NewEventExporter(config *Config, log logrus.FieldLogger) (EventExporter, error) {
	return EventExporter{
		config: config,
		log:    log,
	}, nil
}

func (e EventExporter) ExportEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	e.log.WithField("events", len(events)).Info("Sending batch of events to stdout sink")

	if err := e.sendUpstream(ctx, events); err != nil {
		return err
	}

	return nil
}

func (e EventExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *EventExporter) sendUpstream(ctx context.Context, events []*xatu.DecoratedEvent) error {
	for _, event := range events {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		e.log.WithField("event", string(eventAsJSON)).Info("stdout sink event")
	}

	return nil
}
