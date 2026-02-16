//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/consumoor"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	consumoorCfgFile string
)

// ConsumoorOverride defines a CLI/env override for consumoor config.
type ConsumoorOverride struct {
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *consumoor.Override) error
}

// ConsumoorOverrideConfig defines how a single override is wired.
type ConsumoorOverrideConfig struct {
	FlagName     string
	EnvName      string
	Description  string
	OverrideFunc func(val string, overrides *consumoor.Override)
}

func createConsumoorOverride(config ConsumoorOverrideConfig) ConsumoorOverride {
	return ConsumoorOverride{
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(config.FlagName, "", config.Description+` (env: `+config.EnvName+`)`)
		},
		Setter: func(cmd *cobra.Command, overrides *consumoor.Override) error {
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

// ConsumoorOverrides is the list of CLI/env overrides for consumoor.
var ConsumoorOverrides = []ConsumoorOverride{
	createConsumoorOverride(ConsumoorOverrideConfig{
		FlagName:    "metrics-addr",
		EnvName:     "METRICS_ADDR",
		Description: "sets the metrics address",
		OverrideFunc: func(val string, overrides *consumoor.Override) {
			overrides.MetricsAddr.Enabled = true
			overrides.MetricsAddr.Value = val
		},
	}),
}

// consumoorCmd represents the consumoor command.
var consumoorCmd = &cobra.Command{
	Use:   "consumoor",
	Short: "Runs Xatu in consumoor mode.",
	Long: `Runs Xatu in consumoor mode, which consumes events from Kafka
	and writes them directly to ClickHouse, replacing Vector.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadConsumoorConfigFromFile(consumoorCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", consumoorCfgFile).Info("Loaded config")

		overrides := &consumoor.Override{}
		for _, override := range ConsumoorOverrides {
			if errr := override.Setter(cmd, overrides); errr != nil {
				log.Fatal(errr)
			}
		}

		c, err := consumoor.New(cmd.Context(), log, config, overrides)
		if err != nil {
			log.Fatal(err)
		}

		if err := c.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu consumoor exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(consumoorCmd)

	consumoorCmd.Flags().StringVar(&consumoorCfgFile, "config", "consumoor.yaml", "config file (default is consumoor.yaml)")

	for _, override := range ConsumoorOverrides {
		override.FlagHelper(consumoorCmd)
	}
}

func loadConsumoorConfigFromFile(file string) (*consumoor.Config, error) {
	if file == "" {
		file = "consumoor.yaml"
	}

	config := &consumoor.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	type plain consumoor.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
