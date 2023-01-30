package coordinator

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/manual"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu"
	xatuCoordinator "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu/coordinator"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Type Type `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}

func (c *Config) Validate() error {
	if c.Type == TypeUnknown {
		return errors.New("coordinator type is required")
	}

	return nil
}

func NewCoordinator(name string, coordinatorType Type, config *RawMessage, handlers *handler.Peer, log logrus.FieldLogger) (Coordinator, error) {
	if coordinatorType == TypeUnknown {
		return nil, errors.New("coordinator type is required")
	}

	switch coordinatorType {
	case TypeManual:
		conf := &manual.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return manual.New(name, conf, handlers, log)
	case TypeXatu:
		conf := &xatuCoordinator.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(name, conf, handlers, log)
	default:
		return nil, fmt.Errorf("coordinator type %s is unknown", coordinatorType)
	}
}
