//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/sage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	sageCfgFile string
)

// sageCmd represents the sage command
var sageCmd = &cobra.Command{
	Use:   "sage",
	Short: "Runs Xatu in sage mode.",
	Long:  `Runs Xatu in sage mode, which means it will connect to an Ethereum beacon p2p network and listen for events (via Armiarma).`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", sageCfgFile).Info("Loading config")

		config, err := loadsageConfigFromFile(sageCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		sage, err := sage.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := sage.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu sage exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(sageCmd)

	sageCmd.Flags().StringVar(&sageCfgFile, "config", "sage.yaml", "config file (default is sage.yaml)")
}

func loadsageConfigFromFile(file string) (*sage.Config, error) {
	if file == "" {
		file = "sage.yaml"
	}

	config := &sage.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain sage.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
