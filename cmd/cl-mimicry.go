//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/clmimicry"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	clMimicryCfgFile string
)

type CLMimicryOverride struct {
	EnvVar     string
	Flag       string
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *clmimicry.Override) error
}

var CLMimicryOverrides = []CLMimicryOverride{
	{
		EnvVar: "METRICS_ADDR",
		Flag:   "metrics-addr",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(metricsAddrFlag, "", `metrics address (env: METRICS_ADDR). If set, overrides the metrics address in the config file.`)
		},
		Setter: func(cmd *cobra.Command, overrides *clmimicry.Override) error {
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

// clMimicryCmd represents the consensus layer mimicry command
var clMimicryCmd = &cobra.Command{
	Use:   "cl-mimicry",
	Short: "Runs Xatu in CL Mimicry mode.",
	Long: `Runs Xatu in consensus layer Mimicry mode, which means it will connect to 
	the consensus layer p2p network and create events from the data it receives.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadCLMimicryConfigFromFile(clMimicryCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", clMimicryCfgFile).Info("Loaded config")

		overrides := &clmimicry.Override{}
		for _, o := range CLMimicryOverrides {
			if e := o.Setter(cmd, overrides); e != nil {
				log.Fatal(e)
			}
		}

		mimicry, err := clmimicry.New(cmd.Context(), log, config, overrides)
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
	rootCmd.AddCommand(clMimicryCmd)

	clMimicryCmd.Flags().StringVar(&clMimicryCfgFile, "config", "cl-mimicry.yaml", "config file (default is cl-mimicry.yaml)")

	for _, o := range CLMimicryOverrides {
		o.FlagHelper(clMimicryCmd)
	}
}

func loadCLMimicryConfigFromFile(file string) (*clmimicry.Config, error) {
	if file == "" {
		file = "cl-mimicry.yaml"
	}

	config := &clmimicry.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain clmimicry.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
