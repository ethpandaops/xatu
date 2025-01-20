//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/relaymonitor"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	relayMonitorCfgFile string
)

type RelayMonitorOverride struct {
	EnvVar     string
	Flag       string
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *relaymonitor.Override) error
}

var RelayMonitorOverrides = []RelayMonitorOverride{
	{
		EnvVar: "METRICS_ADDR",
		Flag:   "metrics-addr",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(metricsAddrFlag, "", `metrics address (env: METRICS_ADDR). If set, overrides the metrics address in the config file.`)
		},
		Setter: func(cmd *cobra.Command, overrides *relaymonitor.Override) error {
			val := ""

			if cmd.Flags().Changed(metricsAddrFlag) {
				val = cmd.Flags().Lookup(metricsAddrFlag).Value.String()
			}

			if os.Getenv("METRICS_ADDR") != "" {
				val = os.Getenv("METRICS_ADDR")
			}

			if val == "" {
				return nil
			}

			overrides.MetricsAddr.Enabled = true
			overrides.MetricsAddr.Value = val

			return nil
		},
	},
}

// relayMonitorCmd represents the relay monitor command
var relayMonitorCmd = &cobra.Command{
	Use:   "relay-monitor",
	Short: "Runs Xatu in Relay Monitor mode.",
	Long: `Runs Xatu in Relay Monitor mode, which means it will monitor relay networks
	and create events from the data it receives.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadRelayMonitorConfigFromFile(relayMonitorCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", relayMonitorCfgFile).Info("Loaded config")

		overrides := &relaymonitor.Override{}
		for _, o := range RelayMonitorOverrides {
			if e := o.Setter(cmd, overrides); e != nil {
				log.Fatal(e)
			}
		}

		monitor, err := relaymonitor.New(cmd.Context(), log, config, overrides)
		if err != nil {
			log.Fatal(err)
		}

		if err := monitor.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu relay monitor exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(relayMonitorCmd)

	relayMonitorCmd.Flags().StringVar(&relayMonitorCfgFile, "config", "relay-monitor.yaml", "config file (default is relay-monitor.yaml)")

	for _, o := range RelayMonitorOverrides {
		o.FlagHelper(relayMonitorCmd)
	}
}

func loadRelayMonitorConfigFromFile(file string) (*relaymonitor.Config, error) {
	if file == "" {
		file = "relay-monitor.yaml"
	}

	config := &relaymonitor.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain relaymonitor.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
