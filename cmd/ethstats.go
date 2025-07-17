package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethpandaops/xatu/pkg/ethstats"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	ethstatsCfgFile string
	ethstatsCmd     = &cobra.Command{
		Use:   "ethstats",
		Short: "Runs Xatu in Ethstats server mode",
		Long: `Ethstats server mode accepts WebSocket connections from Ethereum execution clients
following the ethstats protocol. It authenticates clients, collects node information,
and handles protocol messages.`,
		Run: func(cmd *cobra.Command, args []string) {
			initCommon()

			// Load configuration
			config, err := loadEthstatsConfigFromFile(ethstatsCfgFile)
			if err != nil {
				log.Fatal(err)
			}

			// Initialize logger
			log = getLogger(config.LoggingLevel, "")

			log.WithField("location", ethstatsCfgFile).Info("Loaded config")

			// Create and start server
			server, err := ethstats.NewServer(log, config)
			if err != nil {
				log.WithError(err).Fatal("Failed to create server")
			}

			// Setup signal handling
			ctx, cancel := context.WithCancel(context.Background())
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigCh
				log.Info("Received shutdown signal")
				cancel()
			}()

			// Start server
			if err := server.Start(ctx); err != nil {
				log.WithError(err).Fatal("Failed to start server")
			}

			log.Info("Server shutdown complete")
		},
	}
)

func init() {
	rootCmd.AddCommand(ethstatsCmd)

	ethstatsCmd.Flags().StringVar(&ethstatsCfgFile, "config", "ethstats.yaml", "config file")
}

func loadEthstatsConfigFromFile(file string) (*ethstats.Config, error) {
	config := ethstats.NewDefaultConfig()

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}
