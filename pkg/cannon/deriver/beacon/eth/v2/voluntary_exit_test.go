package v2

import (
	"context"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestVoluntaryExitDeriver_Name(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	deriver := &VoluntaryExitDeriver{
		log:        logrus.NewEntry(logrus.New()),
		cfg:        config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, VoluntaryExitDeriverName.String(), deriver.Name())
}

func TestVoluntaryExitDeriver_CannonType(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	deriver := &VoluntaryExitDeriver{
		log:        logrus.NewEntry(logrus.New()),
		cfg:        config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, VoluntaryExitDeriverName, deriver.CannonType())
}

func TestVoluntaryExitDeriver_ActivationFork(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	deriver := &VoluntaryExitDeriver{
		log:        logrus.NewEntry(logrus.New()),
		cfg:        config,
		clientMeta: clientMeta,
	}

	// Test that it returns a valid fork version
	fork := deriver.ActivationFork()
	assert.True(t, fork == spec.DataVersionPhase0 ||
		fork == spec.DataVersionAltair ||
		fork == spec.DataVersionBellatrix ||
		fork == spec.DataVersionCapella ||
		fork == spec.DataVersionDeneb)
}

func TestVoluntaryExitDeriver_OnEventsDerived(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	deriver := &VoluntaryExitDeriver{
		log:        logrus.NewEntry(logrus.New()),
		cfg:        config,
		clientMeta: clientMeta,
	}

	// Test callback registration
	callback := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		return nil
	}

	// Initially no callbacks
	assert.Len(t, deriver.onEventsCallbacks, 0)

	// Register callback
	deriver.OnEventsDerived(context.Background(), callback)

	// Verify callback was registered
	assert.Len(t, deriver.onEventsCallbacks, 1)
}

func TestVoluntaryExitDeriverConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config *VoluntaryExitDeriverConfig
		valid  bool
	}{
		{
			name: "valid_enabled_config",
			config: &VoluntaryExitDeriverConfig{
				Enabled: true,
			},
			valid: true,
		},
		{
			name: "valid_disabled_config",
			config: &VoluntaryExitDeriverConfig{
				Enabled: false,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic config properties
			assert.NotNil(t, tt.config)

			// Test that we can create a deriver with this config
			clientMeta := &xatu.ClientMeta{
				Name:           "test-client",
				Version:        "1.0.0",
				Id:             "test-id",
				Implementation: "test-impl",
			}

			deriver := &VoluntaryExitDeriver{
				log:        logrus.NewEntry(logrus.New()),
				cfg:        tt.config,
				clientMeta: clientMeta,
			}

			assert.Equal(t, tt.config.Enabled, deriver.cfg.Enabled)
			assert.NotNil(t, deriver.log)
			assert.NotNil(t, deriver.clientMeta)
		})
	}
}

func TestNewVoluntaryExitDeriver(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	log := logrus.NewEntry(logrus.New())

	// Test constructor (with nil dependencies for unit test)
	deriver := NewVoluntaryExitDeriver(log, config, nil, nil, clientMeta)

	assert.NotNil(t, deriver)
	assert.Equal(t, config, deriver.cfg)
	assert.Equal(t, clientMeta, deriver.clientMeta)
	assert.NotNil(t, deriver.log)
	assert.Len(t, deriver.onEventsCallbacks, 0)
}

func TestVoluntaryExitDeriver_ImplementsEventDeriver(t *testing.T) {
	config := &VoluntaryExitDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:           "test-client",
		Version:        "1.0.0",
		Id:             "test-id",
		Implementation: "test-impl",
	}

	deriver := &VoluntaryExitDeriver{
		log:        logrus.NewEntry(logrus.New()),
		cfg:        config,
		clientMeta: clientMeta,
	}

	// Test interface methods exist and return expected types
	assert.IsType(t, "", deriver.Name())
	assert.IsType(t, VoluntaryExitDeriverName, deriver.CannonType())
	assert.IsType(t, spec.DataVersionPhase0, deriver.ActivationFork())

	// Stop should work even with nil dependencies
	err := deriver.Stop(context.Background())
	assert.NoError(t, err)
}

func TestVoluntaryExitDeriver_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.NotEmpty(t, VoluntaryExitDeriverName.String())

	// Test that the constant is the expected cannon type
	expectedType := xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT
	assert.Equal(t, expectedType, VoluntaryExitDeriverName)
}
