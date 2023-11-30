//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/seer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	seerCfgFile string
)

// seerCmd represents the seer command
var seerCmd = &cobra.Command{
	Use:   "seer",
	Short: "Runs Xatu in seer mode.",
	Long:  `Runs Xatu in seer mode, which means it will connect to an Ethereum beacon p2p network and listen for events (via Armiarma).`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", seerCfgFile).Info("Loading config")

		config, err := loadseerConfigFromFile(seerCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		seer, err := seer.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := seer.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu seer exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(seerCmd)

	seerCmd.Flags().StringVar(&seerCfgFile, "config", "seer.yaml", "config file (default is seer.yaml)")
}

func loadseerConfigFromFile(file string) (*seer.Config, error) {
	if file == "" {
		file = "seer.yaml"
	}

	config := &seer.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain seer.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
