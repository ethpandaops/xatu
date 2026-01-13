//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/ethstats"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	ethstatsCfgFile string
)

type EthstatsOverride struct {
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *ethstats.Override) error
}

type EthstatsOverrideConfig struct {
	FlagName     string
	EnvName      string
	Description  string
	OverrideFunc func(val string, overrides *ethstats.Override)
}

func createEthstatsOverride(config EthstatsOverrideConfig) EthstatsOverride {
	return EthstatsOverride{
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(config.FlagName, "", config.Description+` (env: `+config.EnvName+`)`)
		},
		Setter: func(cmd *cobra.Command, overrides *ethstats.Override) error {
			val := ""

			if cmd.Flags().Changed(config.FlagName) {
				val = cmd.Flags().Lookup(config.FlagName).Value.String()
			}

			if os.Getenv(config.EnvName) != "" {
				val = os.Getenv(config.EnvName)
			}

			if val == "" {
				return nil
			}

			config.OverrideFunc(val, overrides)

			return nil
		},
	}
}

var EthstatsOverrides = []EthstatsOverride{
	createEthstatsOverride(EthstatsOverrideConfig{
		FlagName:    "ethstats-network-name",
		EnvName:     "ETHSTATS_NETWORK_NAME",
		Description: "sets the network name override",
		OverrideFunc: func(val string, overrides *ethstats.Override) {
			overrides.NetworkName.Enabled = true
			overrides.NetworkName.Value = val
		},
	}),
	createEthstatsOverride(EthstatsOverrideConfig{
		FlagName:    "metrics-addr",
		EnvName:     "METRICS_ADDR",
		Description: "sets the metrics address",
		OverrideFunc: func(val string, overrides *ethstats.Override) {
			overrides.MetricsAddr.Enabled = true
			overrides.MetricsAddr.Value = val
		},
	}),
}

// ethstatsCmd represents the ethstats command
var ethstatsCmd = &cobra.Command{
	Use:   "ethstats",
	Short: "Runs Xatu in ethstats mode.",
	Long: `Runs Xatu in ethstats mode, which means it will accept WebSocket connections
	from ethstats-compatible clients (e.g., geth) and forward the data to configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadEthstatsConfigFromFile(ethstatsCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", ethstatsCfgFile).Info("Loaded config")

		overrides := &ethstats.Override{}
		for _, override := range EthstatsOverrides {
			if errr := override.Setter(cmd, overrides); errr != nil {
				log.Fatal(errr)
			}
		}

		if errr := config.ApplyOverrides(overrides, log); errr != nil {
			log.Fatal(errr)
		}

		e, err := ethstats.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := e.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu ethstats exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(ethstatsCmd)

	ethstatsCmd.Flags().StringVar(&ethstatsCfgFile, "config", "ethstats.yaml", "config file (default is ethstats.yaml)")

	for _, override := range EthstatsOverrides {
		override.FlagHelper(ethstatsCmd)
	}
}

func loadEthstatsConfigFromFile(file string) (*ethstats.Config, error) {
	if file == "" {
		file = "ethstats.yaml"
	}

	config := &ethstats.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	type plain ethstats.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
