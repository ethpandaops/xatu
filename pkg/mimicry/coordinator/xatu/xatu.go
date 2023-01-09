package xatu

import (
	"context"
	"errors"
	"time"

	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	xatupb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

const Type = "xatu"

type Xatu struct {
	config   *Config
	handlers *handler.Peer
	name     string
	log      logrus.FieldLogger

	cache       *cache.SharedCache
	coordinator *Coordinator
	peers       map[string]*Peer
}

func New(name string, config *Config, handlers *handler.Peer, log logrus.FieldLogger) (*Xatu, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	coordinator, err := NewCoordinator(name, config, log)
	if err != nil {
		return nil, err
	}

	handlers.ExecutionStatus = coordinator.HandleExecutionNodeRecordStatus

	return &Xatu{
		config:      config,
		handlers:    handlers,
		log:         log,
		cache:       cache.NewSharedCache(),
		coordinator: coordinator,
		peers:       make(map[string]*Peer),
	}, nil
}

func (h *Xatu) Type() string {
	return Type
}

func (h *Xatu) Start(ctx context.Context) error {
	if err := h.coordinator.Start(ctx); err != nil {
		return err
	}

	if err := h.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (h *Xatu) Stop(ctx context.Context) error {
	return nil
}

func (h *Xatu) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("15s").Do(func() {
		var records []*xatupb.CoordinatedNodeRecord
		for _, peer := range h.peers {
			records = append(records, peer.Record)
		}

		res, err := h.coordinator.CoordinateExecutionNodeRecords(ctx, records)
		if err != nil {
			h.log.WithError(err).Error("failed to coordinate execution node records")
			return
		}

		if res == nil {
			h.log.Error("failed to coordinate execution node records: nil response")
			return
		}

		retryDelay := time.Duration(res.RetryDelay) * time.Second

		for i, peer := range h.peers {
			found := false
			for _, record := range res.NodeRecords {
				if record == i {
					found = true
					peer.RetryDelay(retryDelay)
					break
				}
			}

			// remove peer
			if !found {
				if err := peer.Stop(); err != nil {
					h.log.WithError(err).Error("failed to stop peer")
				}
				delete(h.peers, i)
			}
		}

		for _, record := range res.NodeRecords {
			if _, ok := h.peers[record]; !ok {
				h.peers[record] = NewPeer(h.log, h.handlers, h.cache, record, retryDelay)
				if err := h.peers[record].Start(ctx); err != nil {
					h.log.WithError(err).Error("failed to start peer")
					delete(h.peers, record)
				}
			}
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}
