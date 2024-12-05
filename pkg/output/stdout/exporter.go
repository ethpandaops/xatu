package stdout

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (*ItemExporter, error) {
	return &ItemExporter{
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),
	}, nil
}

func (e *ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx, "StdOutItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to stdout sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		return err
	}

	return nil
}

func (e *ItemExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*xatu.DecoratedEvent) error {
	for _, event := range items {
		if err := e.logEvent(event); err != nil {
			return err
		}
	}

	return nil
}

func (e *ItemExporter) logEvent(event *xatu.DecoratedEvent) error {
	eventAsJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	entry := e.log.WithField("event", string(eventAsJSON))
	msg := "stdout sink event"

	switch e.config.LoggingLevel {
	case "debug":
		entry.Debug(msg)
	case "info":
		entry.Info(msg)
	case "warn":
		entry.Warn(msg)
	case "error":
		entry.Error(msg)
	default:
		entry.Info(msg)
	}

	return nil
}
