//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/sentry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	sentryCfgFile string
)

// sentryCmd represents the sentry command
var sentryCmd = &cobra.Command{
	Use:   "sentry",
	Short: "Runs Xatu in Sentry mode.",
	Long: `Runs Xatu in Sentry mode, which means it will listen for events from
	an Ethereum beacon node and forward the data on to 	the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", sentryCfgFile).Info("Loading config")

		config, err := loadSentryConfigFromFile(sentryCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		sentry, err := sentry.New(cmd.Context(), log, config)
		if err != nil {
			log.Fatal(err)
		}

		if err := sentry.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu sentry exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(sentryCmd)

	sentryCmd.Flags().StringVar(&sentryCfgFile, "config", "sentry.yaml", "config file (default is sentry.yaml)")
}

func loadSentryConfigFromFile(file string) (*sentry.Config, error) {
	if file == "" {
		file = "sentry.yaml"
	}

	config := &sentry.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain sentry.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
