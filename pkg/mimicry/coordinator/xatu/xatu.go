package xatu

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/sirupsen/logrus"
)

const Type = "xatu"

type Xatu struct {
	config *Config

	handlers *handler.Peer

	log logrus.FieldLogger
}

func New(config *Config, handlers *handler.Peer, log logrus.FieldLogger) (*Xatu, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Xatu{
		config:   config,
		handlers: handlers,
		log:      log,
	}, nil
}

func (h *Xatu) Type() string {
	return Type
}

func (h *Xatu) Start(ctx context.Context) error {
	return nil
}

func (h *Xatu) Stop(ctx context.Context) error {
	return nil
}
