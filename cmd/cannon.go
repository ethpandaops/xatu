//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/cannon"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	cannonCfgFile string
)

// cannonCmd represents the cannon command
var cannonCmd = &cobra.Command{
	Use:   "cannon",
	Short: "Runs Xatu in cannon mode.",
	Long: `Runs Xatu in cannon mode, which means it will connect to xatu
	server and process jobs like deriving events from beacon blocks..`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadcannonConfigFromFile(cannonCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLoggerWithOverride(config.LoggingLevel, "")

		log.WithField("location", cannonCfgFile).Info("Loaded config")

		cannon, err := cannon.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := cannon.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu cannon exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(cannonCmd)

	cannonCmd.Flags().StringVar(&cannonCfgFile, "config", "cannon.yaml", "config file (default is cannon.yaml)")
}

func loadcannonConfigFromFile(file string) (*cannon.Config, error) {
	if file == "" {
		file = "cannon.yaml"
	}

	config := &cannon.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain cannon.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
