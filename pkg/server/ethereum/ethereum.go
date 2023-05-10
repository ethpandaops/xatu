package ethereum

import (
	"fmt"
	"time"

	"github.com/ethpandaops/ethwallclock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BeaconChain struct {
	log       logrus.FieldLogger
	wallclock *ethwallclock.EthereumBeaconChain
	config    *Config

	genesisTime time.Time
}

func NewBeaconChain(log logrus.FieldLogger, config *Config) (*BeaconChain, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	secondsPerSlot, err := time.ParseDuration(fmt.Sprintf("%vs", config.Network.Spec.SecondsPerSlot))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse seconds per slot")
	}

	genesisTime := time.Unix(int64(config.Network.Spec.GenesisTime), 0)

	wc := ethwallclock.NewEthereumBeaconChain(genesisTime, secondsPerSlot, config.Network.Spec.SlotsPerEpoch)

	return &BeaconChain{
		log:         log.WithField("component", "ethereum/beaconchain"),
		wallclock:   wc,
		config:      config,
		genesisTime: genesisTime,
	}, nil
}

func (b *BeaconChain) Start() error {
	b.log.WithFields(logrus.Fields{
		"network": b.config.Network.Name,
	}).Info("starting ethereum beacon chain")

	return nil
}

func (b *BeaconChain) Wallclock() *ethwallclock.EthereumBeaconChain {
	return b.wallclock
}

func (b *BeaconChain) Stop() error {
	b.log.WithFields(logrus.Fields{
		"network": b.config.Network.Name,
	}).Info("stopping ethereum beacon chain")

	return nil
}

func (b *BeaconChain) SlotsPerEpoch() uint64 {
	return b.config.Network.Spec.SlotsPerEpoch
}

func (b *BeaconChain) SecondsPerSlot() uint64 {
	return b.config.Network.Spec.SecondsPerSlot
}

func (b *BeaconChain) GenesisTime() time.Time {
	return b.genesisTime
}

func (b *BeaconChain) Spec() *SpecConfig {
	return &b.config.Network.Spec
}

func (b *BeaconChain) NetworkName() string {
	return b.config.Network.Name
}

func (b *BeaconChain) NetworkID() uint64 {
	return b.config.Network.ID
}
