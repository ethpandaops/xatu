//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/armiarma"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	armiarmaCfgFile string
)

// armiarmaCmd represents the armiarma command
var armiarmaCmd = &cobra.Command{
	Use:   "armiarma",
	Short: "Runs Xatu in armiarma mode.",
	Long:  `Runs Xatu in armiarma mode, which means it will connect to an Ethereum beacon p2p network and listen for events.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", armiarmaCfgFile).Info("Loading config")

		config, err := loadarmiarmaConfigFromFile(armiarmaCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		armiarma, err := armiarma.New(log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := armiarma.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu armiarma exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(armiarmaCmd)

	armiarmaCmd.Flags().StringVar(&armiarmaCfgFile, "config", "armiarma.yaml", "config file (default is armiarma.yaml)")
}

func loadarmiarmaConfigFromFile(file string) (*armiarma.Config, error) {
	if file == "" {
		file = "armiarma.yaml"
	}

	config := &armiarma.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain armiarma.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
