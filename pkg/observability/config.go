package observability

import (
	"crypto/tls"

	"github.com/ethpandaops/beacon/pkg/human"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

type TracingConfig struct {
	Enabled     bool                       `yaml:"enabled" default:"false"`
	Endpoint    string                     `yaml:"endpoint" default:""`
	URLPath     string                     `yaml:"urlPath" default:""`
	Timeout     human.Duration             `yaml:"timeout" default:"15s"`
	Compression bool                       `yaml:"compression" default:"true"`
	Headers     map[string]string          `yaml:"headers"`
	Insecure    bool                       `yaml:"insecure" default:"false"`
	Retry       *otlptracehttp.RetryConfig `yaml:"retry"`
	TLS         *tls.Config                `yaml:"tls"`
}

func (t *TracingConfig) Validate() error {
	return nil
}

func (t *TracingConfig) AsOTelOpts() []otlptracehttp.Option {
	var opts []otlptracehttp.Option

	if t.Endpoint != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(t.Endpoint))
	}

	if t.URLPath != "" {
		opts = append(opts, otlptracehttp.WithURLPath(t.URLPath))
	}

	if t.Timeout.Duration != 0 {
		opts = append(opts, otlptracehttp.WithTimeout(t.Timeout.Duration))
	}

	if t.Compression {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
	} else {
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.NoCompression))
	}

	if len(t.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(t.Headers))
	}

	if t.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if t.Retry != nil && t.Retry.Enabled {
		opts = append(opts, otlptracehttp.WithRetry(*t.Retry))
	}

	if t.TLS != nil {
		opts = append(opts, otlptracehttp.WithTLSClientConfig(t.TLS))
	}

	return opts
}
