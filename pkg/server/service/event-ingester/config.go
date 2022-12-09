package eventingester

import "github.com/ethpandaops/xatu/pkg/output"

type Config struct {
	// Outputs is the list of sinks to use.
	Outputs []output.Config `yaml:"outputs"`
}
