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

var (
	authorizationFlag = "output-authorization"
	presetFlag        = "preset"
	beaconNodeURLFlag = "beacon-node-url"
)

type SentryOverride struct {
	EnvVar     string
	Flag       string
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *sentry.Override) error
}

var SentryOverrides = []SentryOverride{
	{
		EnvVar: "OUTOUT_AUTHORIZATION",
		Flag:   authorizationFlag,
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(authorizationFlag, "", `event ingester authorization secret (env: OUTPUT_AUTHORIZATION). If set, overrides the secret in the config file, and adds the secret to all Xatu outputs.`)
		},
		Setter: func(cmd *cobra.Command, overrides *sentry.Override) error {
			val := ""

			if cmd.Flags().Changed(authorizationFlag) {
				val = cmd.Flags().Lookup(authorizationFlag).Value.String()
			}

			if os.Getenv("AUTHORIZATION") != "" {
				val = os.Getenv("AUTHORIZATION")
			}

			if val == "" {
				return nil
			}

			overrides.XatuOutputAuth.Enabled = true
			overrides.XatuOutputAuth.Value = val

			return nil
		},
	},
	{
		EnvVar: "PRESET",
		Flag:   "preset",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(presetFlag, "", `configuration preset (env: PRESET). If set, overrides the configuration preset in the config file. See README.md for more information.`)
		},
		Setter: func(cmd *cobra.Command, overrides *sentry.Override) error {
			val := ""

			if cmd.Flags().Changed(presetFlag) {
				val = cmd.Flags().Lookup(presetFlag).Value.String()
			}

			if os.Getenv("PRESET") != "" {
				val = os.Getenv("PRESET")
			}

			if val == "" {
				return nil
			}

			overrides.Preset.Enabled = true
			overrides.Preset.Value = val

			return nil
		},
	},
	{
		EnvVar: "BEACON_NODE_URL",
		Flag:   "beacon-node-url",
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(beaconNodeURLFlag, "", `beacon node URL (env: BEACON_NODE_URL). If set, overrides the beacon node URL in the config file.`)
		},
		Setter: func(cmd *cobra.Command, overrides *sentry.Override) error {
			val := ""

			if cmd.Flags().Changed(beaconNodeURLFlag) {
				val = cmd.Flags().Lookup(beaconNodeURLFlag).Value.String()
			}

			if os.Getenv("BEACON_NODE_URL") != "" {
				val = os.Getenv("BEACON_NODE_URL")
			}

			if val == "" {
				return nil
			}

			overrides.BeaconNodeURL.Enabled = true
			overrides.BeaconNodeURL.Value = val

			return nil
		},
	},
}

// sentryCmd represents the sentry command
var sentryCmd = &cobra.Command{
	Use:   "sentry",
	Short: "Runs Xatu in Sentry mode.",
	Long: `Runs Xatu in Sentry mode, which means it will listen for events from
	an Ethereum beacon node and forward the data on to the configured sinks.`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		log.WithField("location", sentryCfgFile).Info("Loading config")

		overrides := &sentry.Override{}
		for _, o := range SentryOverrides {
			if errr := o.Setter(cmd, overrides); errr != nil {
				log.Fatal(errr)
			}
		}

		allowEmptyConfig := false
		if cmd.Flags().Changed(presetFlag) {
			allowEmptyConfig = true
		}

		config, err := loadSentryConfigFromFile(sentryCfgFile, allowEmptyConfig)
		if err != nil {
			log.Fatal(err)
		}

		// If we have a valid preset configuration, use it to override the config
		if cmd.Flags().Changed(presetFlag) {
			log.Info("Overriding sentry configuration with preset")

			config.Preset = cmd.Flags().Lookup(presetFlag).Value.String()
		}

		log.Info("Config loaded")

		logLevel, err := logrus.ParseLevel(config.LoggingLevel)
		if err != nil {
			log.WithField("logLevel", config.LoggingLevel).Fatal("invalid logging level")
		}

		log.SetLevel(logLevel)

		sentry, err := sentry.New(cmd.Context(), log, config, overrides)
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

	for _, o := range SentryOverrides {
		o.FlagHelper(sentryCmd)
	}
}

func loadSentryConfigFromFile(file string, allowMissingFile bool) (*sentry.Config, error) {
	if file == "" {
		file = "sentry.yaml"
	}

	config := &sentry.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		if os.IsNotExist(err) && allowMissingFile {
			return config, nil
		}

		return nil, err
	}

	type plain sentry.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
