package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/output/buffer"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

const SinkType = "http"

type HTTP struct {
	config *Config
	log    logrus.FieldLogger
	buf    *buffer.DecoratedEventBuffer

	client *http.Client
}

func New(config *Config, log logrus.FieldLogger) (*HTTP, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	buf := buffer.NewDecoratedEventBuffer(config.MaxItems)

	return &HTTP{
		config: config,
		log:    log,
		buf:    buf,

		client: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
	}, nil
}

func (h *HTTP) Type() string {
	return SinkType
}

func (h *HTTP) Start(ctx context.Context) error {
	h.buf.OnAdded(func(ctx context.Context, event *xatu.DecoratedEvent) {
		if h.buf.Len() >= h.config.BatchSize {
			if err := h.send(ctx); err != nil {
				h.log.WithError(err).Error("Failed to send batch of events to HTTP sink")
			}
		}
	})

	return nil
}

func (h *HTTP) send(ctx context.Context) error {
	// Pull as many items out of the buffer as we can
	events := []*xatu.DecoratedEvent{}

	for event := range h.buf.Read() {
		events = append(events, event)

		if len(events) == h.config.BatchSize {
			break
		}
	}

	h.log.WithField("events", len(events)).Info("Sending batch of events to HTTP sink")

	if err := h.sendUpstream(ctx, events); err != nil {
		return err
	}

	return nil
}

func (h *HTTP) sendUpstream(ctx context.Context, events []*xatu.DecoratedEvent) error {
	httpMethod := "POST"

	var rsp *http.Response

	body := ""

	for _, event := range events {
		eventAsJSON, err := protojson.Marshal(event)
		if err != nil {
			return err
		}

		body += string(eventAsJSON) + "\n"

		if event.Meta.Client.Event.Name == xatu.ClientMeta_Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION {
			h.log.WithField("event", string(eventAsJSON)).Info("Sending attestation to HTTP sink")
		}
	}

	req, err := http.NewRequest(httpMethod, h.config.Address, bytes.NewBufferString(body))
	if err != nil {
		return err
	}

	for k, v := range h.config.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	rsp, err = h.client.Do(req)
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

func (h *HTTP) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	h.buf.Write(ctx, event)

	return nil
}
