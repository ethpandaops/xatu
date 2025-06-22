package ethereum

import (
	"context"
	"testing"

	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBeaconNode_Structure(t *testing.T) {
	config := &Config{
		BeaconNodeAddress:   "http://localhost:5052",
		OverrideNetworkName: "testnet",
	}

	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.FatalLevel) // Suppress logs during tests

	// Test the structure can be created (without calling the constructor that requires network access)
	beaconNode := &BeaconNode{
		config: config,
		log:    logger,
	}

	assert.NotNil(t, beaconNode)
	assert.Equal(t, config, beaconNode.config)
	assert.Equal(t, logger, beaconNode.log)
}

func TestBeaconNode_ConfigAccess(t *testing.T) {
	config := &Config{
		BeaconNodeAddress:     "http://localhost:5052",
		OverrideNetworkName:   "mainnet",
		BlockCacheSize:        1000,
		BlockCacheTTL:         human.Duration{Duration: 3600000000000}, // 1 hour in nanoseconds
		BlockPreloadWorkers:   5,
		BlockPreloadQueueSize: 5000,
		BeaconNodeHeaders:     map[string]string{"Authorization": "Bearer token"},
	}

	beaconNode := &BeaconNode{
		config: config,
		log:    logrus.NewEntry(logrus.New()),
	}

	// Test config field access
	assert.Equal(t, "http://localhost:5052", beaconNode.config.BeaconNodeAddress)
	assert.Equal(t, "mainnet", beaconNode.config.OverrideNetworkName)
	assert.Equal(t, uint64(1000), beaconNode.config.BlockCacheSize)
	assert.Equal(t, uint64(5), beaconNode.config.BlockPreloadWorkers)
	assert.Equal(t, uint64(5000), beaconNode.config.BlockPreloadQueueSize)
	assert.Contains(t, beaconNode.config.BeaconNodeHeaders, "Authorization")
	assert.Equal(t, "Bearer token", beaconNode.config.BeaconNodeHeaders["Authorization"])
}

func TestBeaconNode_FieldInitialization(t *testing.T) {
	config := &Config{
		BeaconNodeAddress: "http://localhost:5052",
	}
	logger := logrus.NewEntry(logrus.New())

	beaconNode := &BeaconNode{
		config: config,
		log:    logger,
	}

	// Test that required fields are present
	assert.NotNil(t, beaconNode.config)
	assert.NotNil(t, beaconNode.log)

	// Test that optional/uninitialized fields are nil
	assert.Nil(t, beaconNode.beacon)
	assert.Nil(t, beaconNode.metrics)
	assert.Nil(t, beaconNode.services)
	assert.Nil(t, beaconNode.onReadyCallbacks)
	assert.Nil(t, beaconNode.sfGroup)
	assert.Nil(t, beaconNode.blockCache)
	assert.Nil(t, beaconNode.blockPreloadChan)
	assert.Nil(t, beaconNode.blockPreloadSem)
}

func TestBeaconNode_ServiceManagement(t *testing.T) {
	beaconNode := &BeaconNode{
		config:   &Config{},
		log:      logrus.NewEntry(logrus.New()),
		services: []services.Service{},
	}

	// Test that services slice can be accessed
	assert.NotNil(t, beaconNode.services)
	assert.Empty(t, beaconNode.services)
	assert.Len(t, beaconNode.services, 0)
}

func TestBeaconNode_CallbackManagement(t *testing.T) {
	beaconNode := &BeaconNode{
		config:           &Config{},
		log:              logrus.NewEntry(logrus.New()),
		onReadyCallbacks: []func(ctx context.Context) error{},
	}

	// Test that callback slice can be accessed
	assert.NotNil(t, beaconNode.onReadyCallbacks)
	assert.Empty(t, beaconNode.onReadyCallbacks)
	assert.Len(t, beaconNode.onReadyCallbacks, 0)
}

func TestBeaconNode_CacheFields(t *testing.T) {
	beaconNode := &BeaconNode{
		config: &Config{},
		log:    logrus.NewEntry(logrus.New()),
	}

	// Test cache-related fields are initially nil
	assert.Nil(t, beaconNode.blockCache)
	assert.Nil(t, beaconNode.validatorsCache)
	assert.Nil(t, beaconNode.blockPreloadChan)
	assert.Nil(t, beaconNode.validatorsPreloadChan)
	assert.Nil(t, beaconNode.blockPreloadSem)
	assert.Nil(t, beaconNode.validatorsPreloadSem)
}

func TestBeaconNode_SingleflightFields(t *testing.T) {
	beaconNode := &BeaconNode{
		config: &Config{},
		log:    logrus.NewEntry(logrus.New()),
	}

	// Test singleflight-related fields are initially nil
	assert.Nil(t, beaconNode.sfGroup)
	assert.Nil(t, beaconNode.validatorsSfGroup)
}

func TestBeaconNode_ZeroValue(t *testing.T) {
	var beaconNode BeaconNode

	// Test zero value struct
	assert.Nil(t, beaconNode.config)
	assert.Nil(t, beaconNode.log)
	assert.Nil(t, beaconNode.beacon)
	assert.Nil(t, beaconNode.metrics)
	assert.Nil(t, beaconNode.services)
	assert.Nil(t, beaconNode.onReadyCallbacks)
	assert.Nil(t, beaconNode.sfGroup)
	assert.Nil(t, beaconNode.blockCache)
}

func TestBeaconNode_ConfigPointerSafety(t *testing.T) {
	config1 := &Config{
		BeaconNodeAddress:   "http://localhost:5052",
		OverrideNetworkName: "testnet",
	}
	config2 := &Config{
		BeaconNodeAddress:   "http://localhost:5053",
		OverrideNetworkName: "mainnet",
	}

	beaconNode1 := &BeaconNode{config: config1}
	beaconNode2 := &BeaconNode{config: config2}

	// Test that configs are independent
	assert.Equal(t, "http://localhost:5052", beaconNode1.config.BeaconNodeAddress)
	assert.Equal(t, "http://localhost:5053", beaconNode2.config.BeaconNodeAddress)
	assert.Equal(t, "testnet", beaconNode1.config.OverrideNetworkName)
	assert.Equal(t, "mainnet", beaconNode2.config.OverrideNetworkName)

	// Test that modifying one doesn't affect the other
	beaconNode1.config.BeaconNodeAddress = "http://localhost:5054"
	assert.Equal(t, "http://localhost:5054", beaconNode1.config.BeaconNodeAddress)
	assert.Equal(t, "http://localhost:5053", beaconNode2.config.BeaconNodeAddress)
}
