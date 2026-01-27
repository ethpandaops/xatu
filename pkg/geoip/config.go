package geoip

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/geoip/maxmind"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled bool `yaml:"enabled" default:"false"`
	Type    Type `yaml:"type" default:"maxmind"`

	Config *RawMessage `yaml:"config"`
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.Type == TypeUnknown {
		return errors.New("geoip provider type is required")
	}

	return nil
}

func NewProvider(providerType Type, config *RawMessage, log logrus.FieldLogger) (Provider, error) {
	if providerType == TypeUnknown {
		return nil, errors.New("geoip provider type is required")
	}

	switch providerType {
	case TypeMaxmind:
		conf := &maxmind.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return maxmind.New(conf, log)
	default:
		return nil, fmt.Errorf("geoip provider type %s is unknown", providerType)
	}
}
