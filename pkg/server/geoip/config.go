package geoip

import (
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/internal/configdecode"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/server/geoip/maxmind"
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

func NewProvider(providerType Type, config *RawMessage, log observability.ContextualLogger) (Provider, error) {
	if providerType == TypeUnknown {
		return nil, errors.New("geoip provider type is required")
	}

	switch providerType {
	case TypeMaxmind:
		conf := &maxmind.Config{}

		if err := configdecode.Decode(config, conf); err != nil {
			return nil, err
		}

		return maxmind.New(conf, log)
	default:
		return nil, fmt.Errorf("geoip provider type %s is unknown", providerType)
	}
}
