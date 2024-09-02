package ethereum

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type BeaconNetwork struct {
	log       logrus.FieldLogger
	config    *Config
	wallclock *ethwallclock.EthereumBeaconChain
}

//nolint:tagliatelle // At the mercy of the config spec.
type NetworkConfig struct {
	SecondsPerSlot uint64 `yaml:"SECONDS_PER_SLOT"`
	MinGenesisTime uint64 `yaml:"MIN_GENESIS_TIME"`
	GenesisDelay   uint64 `yaml:"GENESIS_DELAY"`
}

func NewBeaconNetwork(log logrus.FieldLogger, config *Config) (*BeaconNetwork, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &BeaconNetwork{
		log:    log.WithField("component", "ethereum/beacon_network"),
		config: config,
	}, nil
}

func (b *BeaconNetwork) Wallclock() *ethwallclock.EthereumBeaconChain {
	return b.wallclock
}

func (b *BeaconNetwork) Start(ctx context.Context) error {
	// Fetch the network config from the URL
	networkConfig, err := b.fetchNetworkConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch network config")
	}

	b.log.WithFields(logrus.Fields{
		"seconds_per_slot": networkConfig.SecondsPerSlot,
	}).Info("Fetched network config")

	if networkConfig.SecondsPerSlot == 0 {
		return errors.New("invalid seconds_per_slot value found in network config: 0")
	}

	if networkConfig.MinGenesisTime == 0 {
		return errors.New("invalid min_genesis_time value found in network config: 0")
	}

	if networkConfig.GenesisDelay == 0 {
		return errors.New("invalid genesis_delay value found in network config: 0")
	}

	// Calculate the genesis time
	// Convert the genesis time to a Unix timestamp
	genesisTime := time.Unix(int64(networkConfig.MinGenesisTime), 0).Add(time.Duration(networkConfig.GenesisDelay) * time.Second)

	if b.config.OverrideGenesisTime != nil {
		b.log.WithField("override_genesis_time", *b.config.OverrideGenesisTime).Info("Using override genesis time")

		genesisTime = time.Unix(int64(*b.config.OverrideGenesisTime), 0)
	}

	// Create a new EthereumBeaconChain with the calculated genesis time and network config
	b.log.WithFields(logrus.Fields{
		"genesis_time": genesisTime.Unix(),
		"human_time":   genesisTime.Format("2006-01-02 15:04:05"),
	}).Info("Fetched genesis time")

	// Create a new EthereumBeaconChain with the fetched genesis time and network config
	b.wallclock = ethwallclock.NewEthereumBeaconChain(
		genesisTime,
		time.Duration(networkConfig.SecondsPerSlot)*time.Second,
		b.config.SlotsPerEpoch,
	)

	return nil
}

func (b *BeaconNetwork) fetchNetworkConfig(ctx context.Context) (*NetworkConfig, error) {
	resp, err := http.Get(b.config.NetworkConfigURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config NetworkConfig

	err = yaml.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
