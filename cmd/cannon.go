//nolint:dupl // disable duplicate code warning for cmds
package cmd

import (
	"os"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/cannon"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	cannonCfgFile string
)

type CannonOverride struct {
	FlagHelper func(cmd *cobra.Command)
	Setter     func(cmd *cobra.Command, overrides *cannon.Override) error
}

type CannonOverrideConfig struct {
	FlagName     string
	EnvName      string
	Description  string
	OverrideFunc func(val string, overrides *cannon.Override)
}

func createCannonOverride(config CannonOverrideConfig) CannonOverride {
	return CannonOverride{
		FlagHelper: func(cmd *cobra.Command) {
			cmd.Flags().String(config.FlagName, "", config.Description+` (env: `+config.EnvName+`)`)
		},
		Setter: func(cmd *cobra.Command, overrides *cannon.Override) error {
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

var CannonOverrides = []CannonOverride{
	createCannonOverride(CannonOverrideConfig{
		FlagName:    "cannon-xatu-output-authorization",
		EnvName:     "CANNON_XATU_OUTPUT_AUTHORIZATION",
		Description: "sets the authorization secret for all xatu outputs",
		OverrideFunc: func(val string, overrides *cannon.Override) {
			overrides.XatuOutputAuth.Enabled = true
			overrides.XatuOutputAuth.Value = val
		},
	}),
	createCannonOverride(CannonOverrideConfig{
		FlagName:    "cannon-xatu-coordinator-authorization",
		EnvName:     "CANNON_XATU_COORDINATOR_AUTHORIZATION",
		Description: "sets the authorization secret for the xatu coordinator",
		OverrideFunc: func(val string, overrides *cannon.Override) {
			overrides.XatuCoordinatorAuth.Enabled = true
			overrides.XatuCoordinatorAuth.Value = val
		},
	}),
	createCannonOverride(CannonOverrideConfig{
		FlagName:    "cannon-beacon-node-url",
		EnvName:     "CANNON_BEACON_NODE_URL",
		Description: "sets the beacon node url",
		OverrideFunc: func(val string, overrides *cannon.Override) {
			overrides.BeaconNodeURL.Enabled = true
			overrides.BeaconNodeURL.Value = val
		},
	}),
	createCannonOverride(CannonOverrideConfig{
		FlagName:    "cannon-beacon-node-authorization-header",
		EnvName:     "CANNON_BEACON_NODE_AUTHORIZATION_HEADER",
		Description: "sets the beacon node authorization header",
		OverrideFunc: func(val string, overrides *cannon.Override) {
			overrides.BeaconNodeAuthorizationHeader.Enabled = true
			overrides.BeaconNodeAuthorizationHeader.Value = val
		},
	}),
	createCannonOverride(CannonOverrideConfig{
		FlagName:    "cannon-network-name",
		EnvName:     "CANNON_NETWORK_NAME",
		Description: "sets the network name",
		OverrideFunc: func(val string, overrides *cannon.Override) {
			overrides.NetworkName.Enabled = true
			overrides.NetworkName.Value = val
		},
	}),
}

// cannonCmd represents the cannon command
var cannonCmd = &cobra.Command{
	Use:   "cannon",
	Short: "Runs Xatu in cannon mode.",
	Long: `Runs Xatu in cannon mode, which means it will connect to xatu
	server and process jobs like deriving events from beacon blocks..`,
	Run: func(cmd *cobra.Command, args []string) {
		initCommon()

		config, err := loadcannonConfigFromFile(cannonCfgFile)
		if err != nil {
			log.Fatal(err)
		}

		log = getLogger(config.LoggingLevel, "")

		log.WithField("location", cannonCfgFile).Info("Loaded config")

		overrides := &cannon.Override{}

		for _, override := range CannonOverrides {
			if errr := override.Setter(cmd, overrides); errr != nil {
				log.Fatal(errr)
			}
		}

		cannon, err := cannon.New(cmd.Context(), log, config, overrides)
		if err != nil {
			log.Fatal(err)
		}

		if err := cannon.Start(cmd.Context()); err != nil {
			log.Fatal(err)
		}

		log.Info("Xatu cannon exited - cya!")
	},
}

func init() {
	rootCmd.AddCommand(cannonCmd)

	cannonCmd.Flags().StringVar(&cannonCfgFile, "config", "cannon.yaml", "config file (default is cannon.yaml)")

	for _, override := range CannonOverrides {
		override.FlagHelper(cannonCmd)
	}
}

func loadcannonConfigFromFile(file string) (*cannon.Config, error) {
	if file == "" {
		file = "cannon.yaml"
	}

	config := &cannon.Config{}

	if err := defaults.Set(config); err != nil {
		return nil, err
	}

	yamlFile, err := os.ReadFile(file)

	if err != nil {
		return nil, err
	}

	type plain cannon.Config

	if err := yaml.Unmarshal(yamlFile, (*plain)(config)); err != nil {
		return nil, err
	}

	return config, nil
}
