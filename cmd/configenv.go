package cmd

import (
	"os"

	"github.com/ethpandaops/xatu/pkg/configenv"
)

// loadConfigFile reads a YAML config file into out, expanding ${VAR} and
// ${VAR:-default} placeholders in its scalar values.
func loadConfigFile(file string, out any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return configenv.Decode(data, out)
}
