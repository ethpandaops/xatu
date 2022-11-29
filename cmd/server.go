package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	serverCfgFile string
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs Xatu in Server mode.",
	Long: `Runs Xatu in Server mode, which means it will listen to gRPC requests from
	Xatu Sentry nodes and forward the data on to the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", serverCfgFile).Info("Loading config")

		config, err := loadServerConfigFromFile(serverCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		server := server.NewXatu(cmd.Context(), log, config)

		if err := server.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu server exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVar(&serverCfgFile, "config", "server.yaml", "config file (default is server.yaml)")
}

func loadServerConfigFromFile(file string) (*server.Config, error) {
	if file == "" {
		file = "server.yaml"
	}

	config := &server.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain server.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
