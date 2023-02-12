package server

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const Type = "redis-server"

type Server struct {
	config *Config

	log logrus.FieldLogger

	options *redis.Options
	client  *redis.Client

	metrics *Metrics
}

func New(config *Config, log logrus.FieldLogger) (*Server, error) {
	options, err := redis.ParseURL(config.Address)
	if err != nil {
		return nil, err
	}

	return &Server{
		config:  config,
		log:     log.WithField("store/cache", Type),
		options: options,
		metrics: NewMetrics("xatu_server_store_cache"),
	}, nil
}

func (s *Server) Type() string {
	return Type
}

func (s *Server) Start(ctx context.Context) error {
	s.client = redis.NewClient(s.options)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Get(ctx context.Context, key string) (*string, error) {
	cmd := s.client.Get(ctx, key)
	if cmd.Err() != nil {
		if errors.Is(cmd.Err(), redis.Nil) {
			s.metrics.AddGet(1, s.Type(), "miss")

			return nil, nil
		}

		s.metrics.AddGet(1, s.Type(), "error")

		return nil, cmd.Err()
	}

	s.metrics.AddGet(1, s.Type(), "hit")

	item := cmd.Val()

	return &item, nil
}

func (s *Server) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	cmd := s.client.Set(ctx, key, value, ttl)

	if cmd.Err() != nil {
		s.metrics.AddSet(1, s.Type(), "error")
		return cmd.Err()
	}

	s.metrics.AddSet(1, s.Type(), "ok")

	return nil
}

func (s *Server) Delete(ctx context.Context, key string) error {
	cmd := s.client.Del(ctx, key)

	if cmd.Err() != nil {
		s.metrics.AddDelete(1, s.Type(), "error")
		return cmd.Err()
	}

	s.metrics.AddDelete(1, s.Type(), "ok")

	return nil
}
