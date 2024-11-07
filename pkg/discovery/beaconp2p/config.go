package beaconp2p

import (
	"errors"
	"fmt"

	perrors "github.com/pkg/errors"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/crawler"
	"github.com/ethpandaops/xatu/pkg/discovery/provider"
	"github.com/ethpandaops/xatu/pkg/discovery/shared/static"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled *bool `yaml:"enabled" default:"false"`

	Discovery struct {
		Type provider.EnodeProviderType `yaml:"type"`

		Config *RawMessage `yaml:"config"`
	} `yaml:"discovery"`

	Crawler crawler.Config `yaml:"crawler"`

	DialConcurrency int `yaml:"dialConcurrency" default:"10"`
}

func (c *Config) Validate() error {
	if c.Discovery.Type == provider.EnodeProviderTypeUnknown {
		return errors.New("discovery type is required")
	}

	if err := c.Crawler.Validate(); err != nil {
		return perrors.Wrap(err, "crawler config is invalid")
	}

	return nil
}

func NewNodeDiscoverer(
	p2pType provider.EnodeProviderType,
	config *RawMessage,
	log logrus.FieldLogger,
) (provider.EnodeProvider, error) {
	if p2pType == provider.EnodeProviderTypeUnknown {
		return nil, errors.New("beacon p2p type is required")
	}

	switch p2pType {
	case provider.EnodeProviderTypeStatic:
		conf := &static.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		// Disable discv4
		conf.DiscV4 = false

		return static.New(conf, log)
	default:
		return nil, fmt.Errorf("beacon p2p type %s is unknown", p2pType)
	}
}
