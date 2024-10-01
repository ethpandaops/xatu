package beaconp2p

import (
	"errors"
	"fmt"

	perrors "github.com/pkg/errors"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry"
	"github.com/ethpandaops/xatu/pkg/discovery/crawler"
	"github.com/ethpandaops/xatu/pkg/discovery/ethereum"
	"github.com/ethpandaops/xatu/pkg/discovery/shared/static"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Enabled *bool `yaml:"enabled" default:"false"`

	Discovery struct {
		Type crawler.EnodeProviderType `yaml:"type"`

		Config *RawMessage `yaml:"config"`
	} `yaml:"discovery"`

	Node mimicry.Config `yaml:"node"`

	Ethereum *ethereum.Config `yaml:"ethereum"`
}

func (c *Config) Validate() error {
	if c.Discovery.Type == crawler.EnodeProviderTypeUnknown {
		return errors.New("discovery type is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return perrors.Wrap(err, "ethereum config is invalid")
	}

	return nil
}

func NewNodeDiscoverer(
	p2pType crawler.EnodeProviderType,
	config *RawMessage,
	log logrus.FieldLogger,
) (crawler.EnodeProvider, error) {
	if p2pType == crawler.EnodeProviderTypeUnknown {
		return nil, errors.New("beacon p2p type is required")
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

		// Disable discv4
		conf.DiscV4 = false

		return static.New(conf, log)
	default:
		return nil, fmt.Errorf("beacon p2p type %s is unknown", p2pType)
	}
}
