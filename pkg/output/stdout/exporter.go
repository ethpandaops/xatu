package stdout

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger
}

func NewItemExporter(config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	return ItemExporter{
		config: config,
		log:    log,
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	e.log.WithField("events", len(items)).Info("Sending batch of events to stdout sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		return err
	}

	return nil
}

func (e ItemExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*xatu.DecoratedEvent) error {
	for _, event := range items {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		e.log.WithField("event", string(eventAsJSON)).Info("stdout sink event")
	}

	return nil
}
