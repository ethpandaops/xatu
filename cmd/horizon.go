//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/horizon"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	horizonCfgFile string
)

type HorizonOverride struct {
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *horizon.Override) error
}

type HorizonOverrideConfig struct {
	FlagName     string
	EnvName      string
	Description  string
	OverrideFunc func(val string, overrides *horizon.Override)
}

func createHorizonOverride(config HorizonOverrideConfig) HorizonOverride {
	return HorizonOverride{
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(config.FlagName, "", config.Description+` (env: `+config.EnvName+`)`)
		},
		Setter: func(cmd *cobra.Command, overrides *horizon.Override) error {
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

var HorizonOverrides = []HorizonOverride{
	createHorizonOverride(HorizonOverrideConfig{
		FlagName:    "horizon-xatu-output-authorization",
		EnvName:     "HORIZON_XATU_OUTPUT_AUTHORIZATION",
		Description: "sets the authorization secret for all xatu outputs",
		OverrideFunc: func(val string, overrides *horizon.Override) {
			overrides.XatuOutputAuth.Enabled = true
			overrides.XatuOutputAuth.Value = val
		},
	}),
	createHorizonOverride(HorizonOverrideConfig{
		FlagName:    "metrics-addr",
		EnvName:     "METRICS_ADDR",
		Description: "sets the metrics address",
		OverrideFunc: func(val string, overrides *horizon.Override) {
			overrides.MetricsAddr.Enabled = true
			overrides.MetricsAddr.Value = val
		},
	}),
}

// horizonCmd represents the horizon command
var horizonCmd = &cobra.Command{
	Use:   "horizon",
	Short: "Runs Xatu in horizon mode.",
	Long: `Runs Xatu in horizon mode, which provides real-time head tracking
	with multi-beacon node support and dual-iterator coordination.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadHorizonConfigFromFile(horizonCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", horizonCfgFile).Info("Loaded config")

		overrides := &horizon.Override{}
		for _, override := range HorizonOverrides {
			if errr := override.Setter(cmd, overrides); errr != nil {
				log.Fatal(errr)
			}
		}

		h, err := horizon.New(cmd.Context(), log, config, overrides)
		if err != nil {
			log.Fatal(err)
		}

		if err := h.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu horizon exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(horizonCmd)

	horizonCmd.Flags().StringVar(&horizonCfgFile, "config", "horizon.yaml", "config file (default is horizon.yaml)")

	for _, override := range HorizonOverrides {
		override.FlagHelper(horizonCmd)
	}
}

func loadHorizonConfigFromFile(file string) (*horizon.Config, error) {
	if file == "" {
		file = "horizon.yaml"
	}

	config := &horizon.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	type plain horizon.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
