package static

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/discovery"
	"github.com/sirupsen/logrus"
)

const Type = "static"

type Static struct {
	config *Config

	discV4  *discovery.DiscV4
	discV5  *discovery.DiscV5
	handler func(ctx context.Context, node *enode.Node, source string) error

	log logrus.FieldLogger
}

func New(config *Config, handler func(ctx context.Context, node *enode.Node, source string) error, log logrus.FieldLogger) (*Static, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Static{
		config:  config,
		log:     log,
		handler: handler,
	}, nil
}

func (s *Static) Type() string {
	return Type
}

func (s *Static) Start(ctx context.Context) error {
	if s.config.DiscV4 {
		s.discV4 = discovery.NewDiscV4(ctx, s.config.Restart, s.log)

		if err := s.discV4.UpdateBootNodes(s.config.BootNodes); err != nil {
			return err
		}

		if err := s.discV4.Start(ctx); err != nil {
			return err
		}

		s.discV4.OnNodeRecord(ctx, func(ctx context.Context, node *enode.Node) error {
			return s.handler(ctx, node, "discV4")
		})
	}

	if s.config.DiscV5 {
		s.discV5 = discovery.NewDiscV5(ctx, s.config.Restart, s.log)

		if err := s.discV5.UpdateBootNodes(s.config.BootNodes); err != nil {
			return err
		}

		if err := s.discV5.Start(ctx); err != nil {
			return err
		}

		s.discV5.OnNodeRecord(ctx, func(ctx context.Context, node *enode.Node) error {
			return s.handler(ctx, node, "discV5")
		})
	}

	return nil
}

func (s *Static) Stop(ctx context.Context) error {
	if s.config.DiscV4 {
		if err := s.discV4.Stop(ctx); err != nil {
			return err
		}
	}

	if s.config.DiscV5 {
		if err := s.discV5.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Static) GetNetworkIds() []uint64 {
	return []uint64{}
}

func (s *Static) GetForkIdHashes() []string {
	return []string{}
}

func (s *Static) GetForkDigests() []string {
	return []string{}
}
