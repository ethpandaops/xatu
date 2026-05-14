package output

import (
	"errors"
	"fmt"

	"github.com/creasty/defaults"

	"github.com/ethpandaops/xatu/pkg/observability"
	chSink "github.com/ethpandaops/xatu/pkg/output/clickhouse"
	"github.com/ethpandaops/xatu/pkg/output/http"
	"github.com/ethpandaops/xatu/pkg/output/kafka"
	"github.com/ethpandaops/xatu/pkg/output/s3blobstore"
	"github.com/ethpandaops/xatu/pkg/output/stdout"
	"github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/processor"
	pxatu "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Config struct {
	Name     string   `yaml:"name"`
	SinkType SinkType `yaml:"type"`

	Config *RawMessage `yaml:"config"`

	FilterConfig pxatu.EventFilterConfig `yaml:"filter"`

	ShippingMethod *processor.ShippingMethod `yaml:"shippingMethod"`
}

func (c *Config) Validate() error {
	if c.SinkType == SinkTypeUnknown {
		return errors.New("sink type is required")
	}

	return nil
}

func NewSink(name string, sinkType SinkType, config *RawMessage, log observability.ContextualLogger, filterConfig *pxatu.EventFilterConfig, shippingMethod processor.ShippingMethod) (Sink, error) {
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

		return http.New(name, conf, log, filterConfig, shippingMethod)
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

		return stdout.New(name, conf, log, filterConfig, shippingMethod)
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

		return xatu.New(name, conf, log, filterConfig, shippingMethod)
	case SinkTypeKafka:
		conf := &kafka.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return kafka.New(name, conf, log, filterConfig, shippingMethod)
	case SinkTypeClickhouse:
		conf := &chSink.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return chSink.New(name, conf, log, filterConfig, shippingMethod)
	case SinkTypeS3BlobStore:
		conf := &s3blobstore.Config{}

		if config != nil {
			if err := config.Unmarshal(conf); err != nil {
				return nil, err
			}
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return s3blobstore.New(name, conf, log, filterConfig, shippingMethod)
	default:
		return nil, fmt.Errorf("sink type %s is unknown", sinkType)
	}
}
