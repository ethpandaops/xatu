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

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger

	client *http.Client
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	if config.KeepAlive != nil && !*config.KeepAlive {
		t.DisableKeepAlives = true
	}

	return ItemExporter{
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),

		client: &http.Client{
			Transport: t,
			Timeout:   config.ExportTimeout,
		},
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	e.log.WithField("events", len(items)).Debug("Sending batch of events to HTTP sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

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

	body := ""

	for _, event := range items {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		body += string(eventAsJSON) + "\n"
	}

	buf := bytes.NewBufferString(body)
	if e.config.Compression == CompressionStrategyGzip {
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

	if e.config.Compression == CompressionStrategyGzip {
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

func (e *ItemExporter) gzip(in *bytes.Buffer) (*bytes.Buffer, error) {
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
