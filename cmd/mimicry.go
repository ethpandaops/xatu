//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/mimicry"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	mimicryCfgFile string
)

type MimicryOverride struct {
	EnvVar     string
	Flag       string
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *mimicry.Override) error
}

var MimicryOverrides = []MimicryOverride{
	{
		EnvVar: "METRICS_ADDR",
		Flag:   "metrics-addr",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(metricsAddrFlag, "", `metrics address (env: METRICS_ADDR). If set, overrides the metrics address in the config file.`)
		},
		Setter: func(cmd *cobra.Command, overrides *mimicry.Override) error {
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

// mimicryCmd represents the mimicry command
var mimicryCmd = &cobra.Command{
	Use:   "mimicry",
	Short: "Runs Xatu in Mimicry mode.",
	Long: `Runs Xatu in Mimicry mode, which means it will listen for events from
	an Ethereum beacon node and forward the data on to 	the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadMimicryConfigFromFile(mimicryCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", mimicryCfgFile).Info("Loaded config")

		overrides := &mimicry.Override{}
		for _, o := range MimicryOverrides {
			if e := o.Setter(cmd, overrides); e != nil {
				log.Fatal(e)
			}
		}

		mimicry, err := mimicry.New(cmd.Context(), log, config, overrides)
		if err != nil {
			log.Fatal(err)
		}

		if err := mimicry.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu mimicry exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(mimicryCmd)

	mimicryCmd.Flags().StringVar(&mimicryCfgFile, "config", "mimicry.yaml", "config file (default is mimicry.yaml)")

	for _, o := range MimicryOverrides {
		o.FlagHelper(mimicryCmd)
	}
}

func loadMimicryConfigFromFile(file string) (*mimicry.Config, error) {
	if file == "" {
		file = "mimicry.yaml"
	}

	config := &mimicry.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain mimicry.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
