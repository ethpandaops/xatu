package horizon

import (
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
)

type Override struct {
	MetricsAddr struct {
		Enabled bool
		Value   string
	}
	XatuOutputAuth struct {
		Enabled bool
		Value   string
	}
	// BeaconNodeURLs allows overriding beacon node URLs via environment variables.
	// When enabled, it replaces all configured beacon nodes with a single node.
	BeaconNodeURLs struct {
		Enabled bool
		Value   string
	}
	// BeaconNodeHeaders allows overriding beacon node authorization headers.
	BeaconNodeHeaders struct {
		Enabled bool
		Value   string
	}
	// NetworkName allows overriding the network name.
	NetworkName struct {
		Enabled bool
		Value   string
	}
}

// ApplyBeaconNodeOverrides applies beacon node overrides to the config.
func (o *Override) ApplyBeaconNodeOverrides(cfg *ethereum.Config) {
	if o == nil {
		return
	}

	if o.BeaconNodeURLs.Enabled && o.BeaconNodeURLs.Value != "" {
		// Replace all beacon nodes with the override
		cfg.BeaconNodes = []ethereum.BeaconNodeConfig{
			{
				Name:    "override-node",
				Address: o.BeaconNodeURLs.Value,
				Headers: make(map[string]string),
			},
		}
	}

	if o.BeaconNodeHeaders.Enabled && o.BeaconNodeHeaders.Value != "" {
		for i := range cfg.BeaconNodes {
			if cfg.BeaconNodes[i].Headers == nil {
				cfg.BeaconNodes[i].Headers = make(map[string]string)
			}

			cfg.BeaconNodes[i].Headers["Authorization"] = o.BeaconNodeHeaders.Value
		}
	}

	if o.NetworkName.Enabled && o.NetworkName.Value != "" {
		cfg.OverrideNetworkName = o.NetworkName.Value
	}
}
