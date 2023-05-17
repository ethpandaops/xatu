package ethereum

import (
	"testing"

	"github.com/ethpandaops/ethwallclock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewBeaconChain_ValidConfig_ReturnsBeaconChain(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{
			Name: "TestNet",
			ID:   1,
			Spec: SpecConfig{
				SlotsPerEpoch:  32,
				SecondsPerSlot: 12,
				GenesisTime:    1590832934,
			},
		},
	}
	log := logrus.New()
	beaconchain, err := NewBeaconChain(log, config)
	assert.NoError(t, err)
	assert.NotNil(t, beaconchain)
}

func TestNewBeaconChain_InvalidConfig_ReturnsError(t *testing.T) {
	config := &Config{}
	log := logrus.New()
	beaconchain, err := NewBeaconChain(log, config)
	assert.Error(t, err)
	assert.Nil(t, beaconchain)
}

func TestBeaconChain_Wallclock_ReturnsEthereumBeaconChain(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{
			Name: "TestNet",
			ID:   1,
			Spec: SpecConfig{
				SlotsPerEpoch:  32,
				SecondsPerSlot: 12,
				GenesisTime:    1590832934,
			},
		},
	}
	log := logrus.New()
	beaconchain, _ := NewBeaconChain(log, config)
	wallclock := beaconchain.Wallclock()
	assert.NotNil(t, wallclock)
	assert.IsType(t, &ethwallclock.EthereumBeaconChain{}, wallclock)
}

func TestBeaconChain_SlotsPerEpoch_ReturnsSlotsPerEpoch(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{
			Name: "TestNet",
			ID:   1,
			Spec: SpecConfig{
				SlotsPerEpoch:  32,
				SecondsPerSlot: 12,
				GenesisTime:    1590832934,
			},
		},
	}
	log := logrus.New()
	beaconchain, _ := NewBeaconChain(log, config)
	slotsPerEpoch := beaconchain.SlotsPerEpoch()
	assert.Equal(t, uint64(32), slotsPerEpoch)
}

func TestBeaconChain_SecondsPerSlot_ReturnsSecondsPerSlot(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{
			Name: "TestNet",
			ID:   1,
			Spec: SpecConfig{
				SlotsPerEpoch:  32,
				SecondsPerSlot: 12,
				GenesisTime:    1590832934,
			},
		},
	}
	log := logrus.New()
	beaconchain, _ := NewBeaconChain(log, config)
	secondsPerSlot := beaconchain.SecondsPerSlot()
	assert.Equal(t, uint64(12), secondsPerSlot)
}
