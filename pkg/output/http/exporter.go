package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
)

type ItemExporter struct {
	config     *Config
	log        logrus.FieldLogger
	compressor *Compressor
	client     *http.Client
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	log = log.WithField("output_name", name).WithField("output_type", SinkType)

	t := http.DefaultTransport.(*http.Transport).Clone()

	if config.KeepAlive != nil && !*config.KeepAlive {
		log.WithField("keep_alive", *config.KeepAlive).Warn("Disabling keep-alives")

		t.DisableKeepAlives = true
	}

	return ItemExporter{
		config: config,
		log:    log,

		client: &http.Client{
			Transport: t,
			Timeout:   config.ExportTimeout,
		},
		compressor: &Compressor{Strategy: config.Compression},
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx, "HTTPItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to HTTP sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func (e ItemExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*xatu.DecoratedEvent) error {
	httpMethod := "POST"

	var rsp *http.Response

	var body strings.Builder

	for _, event := range items {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		body.WriteString(string(eventAsJSON) + "\n")
	}

	buf := bytes.NewBufferString(body.String())

	compressed, err := e.compressor.Compress(buf)
	if err != nil {
		return err
	}

	buf = compressed

	// TODO: check that this also handles processor timeout
	req, err := http.NewRequestWithContext(ctx, httpMethod, e.config.Address, buf)
	if err != nil {
		return err
	}

	for k, v := range e.config.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	e.compressor.AddHeaders(req)

	rsp, err = e.client.Do(req)
	if err != nil {
		return err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", rsp.StatusCode)
	}

	_, err = io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	return nil
}
