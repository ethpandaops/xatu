package ethereum

import (
	"context"
	"encoding/binary"
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

	genesisTime, err := b.fetchGenesisTime(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch genesis time")
	}

	b.log.WithFields(logrus.Fields{
		"genesis_time": genesisTime,
		"human_time":   time.Unix(int64(genesisTime), 0).Format("2006-01-02 15:04:05"),
	}).Info("Fetched genesis time")

	// Create a new EthereumBeaconChain with the fetched genesis time and network config
	b.wallclock = ethwallclock.NewEthereumBeaconChain(
		time.Unix(int64(genesisTime), 0),
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

// FetchGenesisTime fetches the genesis time from a given URL.
func (b *BeaconNetwork) fetchGenesisTime(ctx context.Context) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", b.config.GenesisSSZURL, nil)
	if err != nil {
		return 0, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	// Read only the first 8 bytes for GenesisTime
	data := make([]byte, 8)

	_, err = io.ReadFull(response.Body, data)
	if err != nil {
		return 0, err
	}

	genesisTime := binary.LittleEndian.Uint64(data)

	return genesisTime, nil
}
