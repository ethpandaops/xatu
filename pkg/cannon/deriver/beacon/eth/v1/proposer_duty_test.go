package v1

import (
	"context"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestProposerDutyDeriver_Name(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &ProposerDutyDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, ProposerDutyDeriverName.String(), deriver.Name())
}

func TestProposerDutyDeriver_CannonType(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &ProposerDutyDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	assert.Equal(t, ProposerDutyDeriverName, deriver.CannonType())
}

func TestProposerDutyDeriver_ActivationFork(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &ProposerDutyDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
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

func TestProposerDutyDeriver_OnEventsDerived(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &ProposerDutyDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	// Test callback registration
	callbackCount := 0
	callback := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		callbackCount++
		return nil
	}

	// Initially no callbacks
	assert.Len(t, deriver.onEventsCallbacks, 0)

	// Register callback
	deriver.OnEventsDerived(context.Background(), callback)

	// Verify callback was registered
	assert.Len(t, deriver.onEventsCallbacks, 1)

	// Register another callback
	deriver.OnEventsDerived(context.Background(), callback)
	assert.Len(t, deriver.onEventsCallbacks, 2)
}

func TestProposerDutyDeriverConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config *ProposerDutyDeriverConfig
		valid  bool
	}{
		{
			name: "valid_enabled_config",
			config: &ProposerDutyDeriverConfig{
				Enabled: true,
			},
			valid: true,
		},
		{
			name: "valid_disabled_config", 
			config: &ProposerDutyDeriverConfig{
				Enabled: false,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic config validation
			assert.NotNil(t, tt.config)
			
			// Test that we can create a deriver with this config
			clientMeta := &xatu.ClientMeta{
				Name:         "test-client",
				Version:      "1.0.0",
				Id:           "test-id",
				Implementation: "test-impl",
			}

			deriver := &ProposerDutyDeriver{
				log: logrus.NewEntry(logrus.New()),
				cfg: tt.config,
				clientMeta: clientMeta,
			}

			assert.Equal(t, tt.config.Enabled, deriver.cfg.Enabled)
			assert.NotNil(t, deriver.log)
			assert.NotNil(t, deriver.clientMeta)
		})
	}
}

func TestNewProposerDutyDeriver(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	log := logrus.NewEntry(logrus.New())

	// Test constructor (with nil dependencies for unit test)
	deriver := NewProposerDutyDeriver(log, config, nil, nil, clientMeta)

	assert.NotNil(t, deriver)
	assert.Equal(t, config, deriver.cfg)
	assert.Equal(t, clientMeta, deriver.clientMeta)
	assert.NotNil(t, deriver.log)
	assert.Len(t, deriver.onEventsCallbacks, 0)
}

func TestProposerDutyDeriver_ImplementsEventDeriver(t *testing.T) {
	config := &ProposerDutyDeriverConfig{
		Enabled: true,
	}

	clientMeta := &xatu.ClientMeta{
		Name:         "test-client",
		Version:      "1.0.0",
		Id:           "test-id",
		Implementation: "test-impl",
	}

	deriver := &ProposerDutyDeriver{
		log: logrus.NewEntry(logrus.New()),
		cfg: config,
		clientMeta: clientMeta,
	}

	// Test interface methods exist and return expected types
	assert.IsType(t, "", deriver.Name())
	assert.IsType(t, xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY, deriver.CannonType())
	assert.IsType(t, spec.DataVersionPhase0, deriver.ActivationFork())

	// Test that interface methods exist
	// Note: We don't test Start() with nil dependencies as it would panic
	// That's tested in integration tests with proper mocks

	// Stop should work even with nil dependencies
	err := deriver.Stop(context.Background())
	assert.NoError(t, err)
}

func TestProposerDutyDeriver_Constants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY, ProposerDutyDeriverName)
	assert.NotEmpty(t, ProposerDutyDeriverName.String())
}