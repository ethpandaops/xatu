package contributoor

var (
	// DefaultSubscriptionTopics defines the standard beacon chain topics
	// that clients should subscribe to for comprehensive network monitoring.
	DefaultSubscriptionTopics = []string{
		"block",
		"block_gossip",
		"head",
		"finalized_checkpoint",
		"blob_sidecar",
		"chain_reorg",
		"beacon_attestation",
	}
	// MaxAttestationSubnetSubscriptions is the default maximum number of
	// attestation subnets a client should subscribe to.
	MaxAttestationSubnetSubscriptions = 2
)

// DefaultBootConfiguration returns a default contributoor configuration
// with standard beacon subscription topics and reasonable default values.
func DefaultBootConfiguration() *BootConfigration {
	return &BootConfigration{
		Global:  defaultGlobalConfig(),
		User:    defaultUserConfig(),
		Network: defaultNetworkConfig(),
	}
}

// defaultGlobalConfig returns the default global beacon subscription configuration
// that applies to all users when no specific configuration is provided.
func defaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		BeaconSubscriptions: BeaconSubscriptionsConfig{
			Topics: DefaultSubscriptionTopics,
			AttestationConfig: AttestationConfig{
				MaxSubnets: MaxAttestationSubnetSubscriptions,
			},
		},
	}
}

// defaultUserConfig returns an empty map of user configurations.
// User-specific configurations are typically loaded from YAML configuration.
func defaultUserConfig() map[string]UserConfig {
	return make(map[string]UserConfig)
}

// defaultNetworkConfig returns an empty map of network configurations.
// Network-specific configurations are typically loaded from YAML configuration.
func defaultNetworkConfig() map[string]NetworkConfig {
	return make(map[string]NetworkConfig)
}
