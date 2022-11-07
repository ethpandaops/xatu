package http

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ethpandaops/xatu/pkg/xatu"
	"github.com/sirupsen/logrus"
)

const SinkType = "http"

type HTTP struct {
	config *Config
	log    logrus.FieldLogger
}

func New(config *Config, log logrus.FieldLogger) (*HTTP, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &HTTP{
		config: config,
		log:    log,
	}, nil
}

func (h *HTTP) Type() string {
	return SinkType
}

func (h *HTTP) HandleNewDecoratedEvent(ctx context.Context, event xatu.DecoratedEvent) error {
	asJson, err := json.Marshal(event)
	if err != nil {
		return err
	}

	h.log.WithField("event", string(asJson)).Info("HTTP sink received event")

	return nil
}
