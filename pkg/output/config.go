package output

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/output/http"
	"github.com/ethpandaops/xatu/pkg/output/stdout"
	"github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Name     string   `yaml:"name"`
	SinkType SinkType `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}

func (c *Config) Validate() error {
	if c.SinkType == SinkTypeUnknown {
		return errors.New("sink type is required")
	}

	return nil
}

func NewSink(sinkType SinkType, config *RawMessage, log logrus.FieldLogger) (Sink, error) {
	if sinkType == SinkTypeUnknown {
		return nil, errors.New("sink type is required")
	}

	switch sinkType {
	case SinkTypeHTTP:
		conf := &http.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return http.New(conf, log)
	case SinkTypeStdOut:
		conf := &stdout.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return stdout.New(conf, log)
	case SinkTypeXatu:
		conf := &xatu.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(conf, log)
	default:
		return nil, fmt.Errorf("sink type %s is unknown", sinkType)
	}
}
