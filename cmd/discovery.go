//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/discovery"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	discoveryCfgFile string
)

// discoveryCmd represents the discovery command
var discoveryCmd = &cobra.Command{
	Use:   "discovery",
	Short: "Runs Xatu in Discovery mode.",
	Long: `Runs Xatu in Discovery mode, which means it will use Ethereum Node
	Discovery Protocol v5 to discover nodes.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadDiscoveryConfigFromFile(discoveryCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLoggerWithOverride(config.LoggingLevel, "")

		log.WithField("location", discoveryCfgFile).Info("Loaded config")
		discovery, err := discovery.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := discovery.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu discovery exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(discoveryCmd)

	discoveryCmd.Flags().StringVar(&discoveryCfgFile, "config", "discovery.yaml", "config file (default is discovery.yaml)")
}

func loadDiscoveryConfigFromFile(file string) (*discovery.Config, error) {
	if file == "" {
		file = "discovery.yaml"
	}

	config := &discovery.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain discovery.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
