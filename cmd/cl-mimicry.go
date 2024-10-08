//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/clmimicry"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	clMimicryCfgFile string
)

// clMimicryCmd represents the consensus layer mimicry command
var clMimicryCmd = &cobra.Command{
	Use:   "cl-mimicry",
	Short: "Runs Xatu in CL Mimicry mode.",
	Long: `Runs Xatu in consensus layer Mimicry mode, which means it will connect to 
	the consensus layer p2p network and create events from the data it receives.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadCLMimicryConfigFromFile(clMimicryCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLoggerWithOverride(config.LoggingLevel, "")

		log.WithField("location", clMimicryCfgFile).Info("Loaded config")

		mimicry, err := clmimicry.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := mimicry.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu mimicry exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(clMimicryCmd)

	clMimicryCmd.Flags().StringVar(&clMimicryCfgFile, "config", "cl-mimicry.yaml", "config file (default is cl-mimicry.yaml)")
}

func loadCLMimicryConfigFromFile(file string) (*clmimicry.Config, error) {
	if file == "" {
		file = "cl-mimicry.yaml"
	}

	config := &clmimicry.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain clmimicry.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
