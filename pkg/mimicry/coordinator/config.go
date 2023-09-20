package coordinator

import (
	"errors"
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/static"
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

func NewCoordinator(name string, coordinatorType Type, config *RawMessage, handlers *handler.Peer, captureDelay time.Duration, log logrus.FieldLogger) (Coordinator, error) {
	if coordinatorType == TypeUnknown {
		return nil, errors.New("coordinator type is required")
	}

	switch coordinatorType {
	case TypeStatic:
		conf := &static.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return static.New(name, conf, handlers, captureDelay, log)
	case TypeXatu:
		conf := &xatuCoordinator.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(name, conf, handlers, captureDelay, log)
	default:
		return nil, fmt.Errorf("coordinator type %s is unknown", coordinatorType)
	}
}
