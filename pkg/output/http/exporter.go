package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

type EventExporter struct {
	config *Config
	log    logrus.FieldLogger

	client *http.Client
}

func NewEventExporter(config *Config, log logrus.FieldLogger) (EventExporter, error) {
	return EventExporter{
		config: config,
		log:    log,

		client: &http.Client{
			Timeout: config.ExportTimeout,
		},
	}, nil
}

func (e EventExporter) ExportEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	e.log.WithField("events", len(events)).Info("Sending batch of events to HTTP sink")

	if err := e.sendUpstream(ctx, events); err != nil {
		return err
	}

	return nil
}

func (e EventExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *EventExporter) sendUpstream(ctx context.Context, events []*xatu.DecoratedEvent) error {
	httpMethod := "POST"

	var rsp *http.Response

	body := ""

	for _, event := range events {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		body += string(eventAsJSON) + "\n"
	}

	buf := bytes.NewBufferString(body)
	if e.config.GzipCompression {
		compressed, err := e.gzip(buf)
		if err != nil {
			return err
		}

		buf = compressed
	}

	// TODO: check that this also handles processor timeout
	req, err := http.NewRequestWithContext(ctx, httpMethod, e.config.Address, buf)
	if err != nil {
		return err
	}

	for k, v := range e.config.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	if e.config.GzipCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}

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

func (e *EventExporter) gzip(in *bytes.Buffer) (*bytes.Buffer, error) {
	out := &bytes.Buffer{}
	g := gzip.NewWriter(out)

	_, err := g.Write(in.Bytes())
	if err != nil {
		return out, err
	}

	if err := g.Close(); err != nil {
		return out, err
	}

	return out, nil
}
