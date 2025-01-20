package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/server"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	serverCfgFile string
)

type ServerOverride struct {
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *server.Override) error
}

type ServerOverrideConfig struct {
	FlagName     string
	EnvName      string
	Description  string
	OverrideFunc func(val string, overrides *server.Override)
}

func createServerOverride(config ServerOverrideConfig) ServerOverride {
	return ServerOverride{
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(config.FlagName, "", config.Description+` (env: `+config.EnvName+`)`)
		},
		Setter: func(cmd *cobra.Command, overrides *server.Override) error {
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

var ServerOverrides = []ServerOverride{
	createServerOverride(ServerOverrideConfig{
		FlagName:    "server-event-ingester-basic-auth-username",
		EnvName:     "SERVER_EVENT_INGESTER_BASIC_AUTH_USERNAME",
		Description: "sets the basic auth username for the event ingester",
		OverrideFunc: func(val string, overrides *server.Override) {
			overrides.EventIngesterBasicAuth.Username = val
		},
	}),
	createServerOverride(ServerOverrideConfig{
		FlagName:    "server-event-ingester-basic-auth-password",
		EnvName:     "SERVER_EVENT_INGESTER_BASIC_AUTH_PASSWORD",
		Description: "sets the basic auth password for the event ingester",
		OverrideFunc: func(val string, overrides *server.Override) {
			overrides.EventIngesterBasicAuth.Password = val
		},
	}),
	createServerOverride(ServerOverrideConfig{
		FlagName:    "server-coordinator-auth-secret",
		EnvName:     "SERVER_COORDINATOR_AUTH_SECRET",
		Description: "sets the auth secret for the coordinator",
		OverrideFunc: func(val string, overrides *server.Override) {
			overrides.CoordinatorAuth.AuthSecret = val
		},
	}),
	createServerOverride(ServerOverrideConfig{
		FlagName:    "metrics-addr",
		EnvName:     "METRICS_ADDR",
		Description: "sets the metrics address",
		OverrideFunc: func(val string, overrides *server.Override) {
			overrides.MetricsAddr.Enabled = true
			overrides.MetricsAddr.Value = val
		},
	}),
}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs Xatu in Server mode.",
	Long: `Runs Xatu in Server mode, which means it will listen to gRPC requests from
	Xatu modules and forward the data on to the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadServerConfigFromFile(serverCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", serverCfgFile).Info("Loaded config")

		o := &server.Override{}

		for _, override := range ServerOverrides {
			if errr := override.Setter(cmd, o); errr != nil {
				log.Fatal(errr)
			}
		}

		server, err := server.NewXatu(cmd.Context(), log, config, o)
		if err != nil {
			log.Fatal(err)
		}

		if err := server.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu server exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVar(&serverCfgFile, "config", "server.yaml", "config file (default is server.yaml)")

	for _, override := range ServerOverrides {
		override.FlagHelper(serverCmd)
	}
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
