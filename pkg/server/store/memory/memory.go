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

func (m *Memory) Type() string {
	return Type
}

func (m *Memory) Start(ctx context.Context) error {
	return nil
}

func (m *Memory) Stop(ctx context.Context) error {
	return nil
}

func (m *Memory) Get(ctx context.Context, key string) (*string, error) {
	item := m.client.Get(key)
	if item == nil {
		m.metrics.AddGet(1, m.Type(), "miss")

		//nolint:nilnil // expected behavior
		return nil, nil
	}

	m.metrics.AddGet(1, m.Type(), "hit")

	value := item.Value()

	return &value, nil
}

func (m *Memory) GetOrSet(ctx context.Context, key, value string, ttl time.Duration) (storedValue *string, retrieved bool, err error) {
	item, retrieved := m.client.GetOrSet(key, value, ttl)

	if retrieved {
		m.metrics.AddGet(1, m.Type(), "hit")
	} else {
		m.metrics.AddGet(1, m.Type(), "miss")
		m.metrics.AddSet(1, m.Type(), "ok")
	}

	v := item.Value()
	storedValue = &v

	return
}

func (m *Memory) GetAndDelete(ctx context.Context, key string) (deletedValue *string, exists bool, err error) {
	item, exists := m.client.GetAndDelete(key)

	if !exists {
		m.metrics.AddGet(1, m.Type(), "miss")

		return
	}

	m.metrics.AddGet(1, m.Type(), "hit")
	m.metrics.AddDelete(1, m.Type(), "ok")

	value := item.Value()
	deletedValue = &value

	return
}

func (m *Memory) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	m.client.Set(key, value, ttl)

	m.metrics.AddSet(1, m.Type(), "ok")

	return nil
}

func (m *Memory) Delete(ctx context.Context, key string) error {
	m.client.Delete(key)

	m.metrics.AddDelete(1, m.Type(), "ok")

	return nil
}
