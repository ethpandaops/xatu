package p2p

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/discovery/crawler"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/ethpandaops/xatu/pkg/discovery/shared/static"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled *bool `yaml:"enabled" default:"true"`

	Type crawler.EnodeProviderType `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}

func (c *Config) Validate() error {
	if c.Type == crawler.EnodeProviderTypeUnknown {
		return errors.New("p2p type is required")
	}

	return nil
}

func NewEnodeProvider(
	p2pType crawler.EnodeProviderType,
	config *RawMessage,
	log logrus.FieldLogger,
) (crawler.EnodeProvider, error) {
	if p2pType == crawler.EnodeProviderTypeUnknown {
		return nil, errors.New("p2p type is required")
	}

	switch p2pType {
	case crawler.EnodeProviderTypeStatic:
		conf := &static.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return static.New(conf, log)
	case crawler.EnodeProviderTypeXatu:
		conf := &xatu.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(conf, log)
	default:
		return nil, fmt.Errorf("p2p type %s is unknown", p2pType)
	}
}
