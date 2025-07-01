package p2p

import (
	"context"
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Type Type `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}

// GetNetworkIDs extracts network IDs from the p2p config based on its type.
func (c *Config) GetNetworkIDs() []uint64 {
	if c.Config == nil {
		return nil
	}

	switch c.Type {
	case TypeXatu:
		conf := &xatu.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return nil
		}

		return conf.NetworkIds
	default:
		// Static and other types don't have network IDs configured
		return nil
	}
}

func (c *Config) Validate() error {
	if c.Type == TypeUnknown {
		return errors.New("p2p type is required")
	}

	return nil
}

func NewP2P(p2pType Type, config *RawMessage, handler func(ctx context.Context, node *enode.Node, source string) error, log logrus.FieldLogger) (P2P, error) {
	if p2pType == TypeUnknown {
		return nil, errors.New("p2p type is required")
	}

	switch p2pType {
	case TypeStatic:
		conf := &static.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return static.New(conf, handler, log)
	case TypeXatu:
		conf := &xatu.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(conf, handler, log)
	default:
		return nil, fmt.Errorf("p2p type %s is unknown", p2pType)
	}
}
