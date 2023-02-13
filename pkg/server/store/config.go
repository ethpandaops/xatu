package store

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/server/store/memory"
	redisCluster "github.com/ethpandaops/xatu/pkg/server/store/redis/cluster"
	redisServer "github.com/ethpandaops/xatu/pkg/server/store/redis/server"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Type Type `yaml:"type" default:"memory"`

	Config *RawMessage `yaml:"config"`
}

func (c *Config) Validate() error {
	if c.Type == TypeUnknown {
		return errors.New("cache type is required")
	}

	return nil
}

func NewCache(cacheType Type, config *RawMessage, log logrus.FieldLogger) (Cache, error) {
	if cacheType == TypeUnknown {
		return nil, errors.New("cache type is required")
	}

	switch cacheType {
	case TypeMemory:
		conf := &memory.Config{}

		return memory.New(conf, log)
	case TypeRedisServer:
		conf := &redisServer.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return redisServer.New(conf, log)
	case TypeRedisCluster:
		conf := &redisCluster.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return redisCluster.New(conf, log)
	default:
		return nil, fmt.Errorf("cache type %s is unknown", cacheType)
	}
}
