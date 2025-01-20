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

type DiscoveryOverride struct {
	EnvVar     string
	Flag       string
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *discovery.Override) error
}

var DiscoveryOverrides = []DiscoveryOverride{
	{
		EnvVar: "METRICS_ADDR",
		Flag:   "metrics-addr",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(metricsAddrFlag, "", `metrics address (env: METRICS_ADDR). If set, overrides the metrics address in the config file.`)
		},
		Setter: func(cmd *cobra.Command, overrides *discovery.Override) error {
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

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", discoveryCfgFile).Info("Loaded config")

		overrides := &discovery.Override{}
		for _, o := range DiscoveryOverrides {
			if e := o.Setter(cmd, overrides); e != nil {
				log.Fatal(e)
			}
		}

		discovery, err := discovery.New(cmd.Context(), log, config, overrides)
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

	for _, o := range DiscoveryOverrides {
		o.FlagHelper(discoveryCmd)
	}
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
