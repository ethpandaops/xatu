package memory

import (
	"context"
	"time"

	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

const Type = "memory"

type Memory struct {
	config *Config

	log logrus.FieldLogger

	client *ttlcache.Cache[string, string]

	metrics *Metrics
}

func New(config *Config, log logrus.FieldLogger) (*Memory, error) {
	return &Memory{
		config:  config,
		log:     log.WithField("store/cache", Type),
		client:  ttlcache.New[string, string](),
		metrics: NewMetrics("xatu_server_store_cache"),
	}, nil
}

func (s *Memory) Type() string {
	return Type
}

func (s *Memory) Start(ctx context.Context) error {
	return nil
}

func (s *Memory) Stop(ctx context.Context) error {
	return nil
}

func (s *Memory) Get(ctx context.Context, key string) (*string, error) {
	item := s.client.Get(key)
	if item == nil {
		s.metrics.AddGet(1, s.Type(), "miss")

		return nil, nil
	}

	s.metrics.AddGet(1, s.Type(), "hit")

	value := item.Value()

	return &value, nil
}

func (s *Memory) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	s.client.Set(key, value, ttl)

	s.metrics.AddSet(1, s.Type(), "ok")

	return nil
}

func (s *Memory) Delete(ctx context.Context, key string) error {
	s.client.Delete(key)

	s.metrics.AddDelete(1, s.Type(), "ok")

	return nil
}
