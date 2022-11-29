package server

import "github.com/ethpandaops/xatu/pkg/output"

type Config struct {
	// The address to listen on.
	Addr string `yaml:"addr" default:":8080"`
	// MetricsAddr is the address to listen on for metrics.
	MetricsAddr string `yaml:"metrics_addr" default:":9090"`
	// LoggingLevel is the logging level to use.
	LoggingLevel string `yaml:"logging_level" default:"info"`
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
}
