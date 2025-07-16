package contributoor

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/contributoor/auth"
)

const (
	ServiceType = "contributoor"
)

type Config struct {
	Enabled       bool                     `yaml:"enabled" default:"false"`
	Authorization auth.AuthorizationConfig `yaml:"authorization"`

	// Configuration response settings.
	Version           string           `yaml:"version" default:"1.0"`
	BootConfiguration BootConfigration `yaml:"bootConfiguration"`
}

// BootConfigration holds the YAML-friendly config structure
type BootConfigration struct {
	Global  GlobalConfig             `yaml:"global"`
	User    map[string]UserConfig    `yaml:"user"`
	Network map[string]NetworkConfig `yaml:"network"`
}

// GlobalConfig
type GlobalConfig struct {
	BeaconSubscriptions BeaconSubscriptionsConfig `yaml:"beaconSubscriptions"`
}

// BeaconSubscriptionsConfig
type BeaconSubscriptionsConfig struct {
	Topics            []string          `yaml:"topics"`
	AttestationConfig AttestationConfig `yaml:"attestationConfig"`
}

// AttestationConfig defines how many attestation subnets to subscribe to.
type AttestationConfig struct {
	MaxSubnets int `yaml:"maxSubnets"`
}

// UserConfig defines user-specific beacon subscription overrides
// that take precedence over global configuration settings.
type UserConfig struct {
	BeaconSubscriptions BeaconSubscriptionsConfig `yaml:"beaconSubscriptions"`
}

// NetworkConfig defines network-specific beacon subscription settings
// that can vary based on the target Ethereum network (mainnet, testnet, etc.).
type NetworkConfig struct {
	BeaconSubscriptions BeaconSubscriptionsConfig `yaml:"beaconSubscriptions"`
}

// Validate ensures the contributoor configuration is valid.
// It checks that authorization is properly configured when enabled.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	return c.Authorization.Validate()
}

// ToProtoForUser converts BootConfigration to protobuf message types, filtering user data for the specified user.
// Only the authenticated user's configuration is included in the user section of the response.
// If authenticatedUsername is empty, all user configurations are included.
func (y *BootConfigration) ToProtoForUser(authenticatedUsername string) *xatu.ContributoorConfiguration {
	// Convert user configs - only include the authenticated user's config.
	userConfigs := make(map[string]*xatu.UserConfig)

	if authenticatedUsername != "" {
		if userConfig, exists := y.User[authenticatedUsername]; exists {
			userConfigs[authenticatedUsername] = &xatu.UserConfig{
				BeaconSubscriptions: userConfig.BeaconSubscriptions.ToProto(),
			}
		}
	}

	// Convert network configs.
	networkConfigs := make(map[string]*xatu.NetworkConfig)
	for networkName, networkConfig := range y.Network {
		networkConfigs[networkName] = &xatu.NetworkConfig{
			BeaconSubscriptions: networkConfig.BeaconSubscriptions.ToProto(),
		}
	}

	return &xatu.ContributoorConfiguration{
		Global: &xatu.GlobalConfig{
			BeaconSubscriptions: y.Global.BeaconSubscriptions.ToProto(),
		},
		User:    userConfigs,
		Network: networkConfigs,
	}
}

// ToProto converts BeaconSubscriptionsConfig to protobuf type.
func (y *BeaconSubscriptionsConfig) ToProto() *xatu.BeaconSubscriptionsConfig {
	return &xatu.BeaconSubscriptionsConfig{
		Topics:            y.Topics,
		AttestationConfig: y.AttestationConfig.ToProto(),
	}
}

// ToProto converts AttestationConfig to protobuf type.
func (y *AttestationConfig) ToProto() *xatu.AttestationConfig {
	return &xatu.AttestationConfig{
		MaxSubnets: int32(y.MaxSubnets), //nolint:gosec // int32 conversion fine
	}
}
