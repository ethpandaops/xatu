package sentry

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// loadConfig mirrors cmd/sentry.go: defaults first, then the YAML file on top.
func loadConfig(t *testing.T, in string) *Config {
	t.Helper()

	config := &Config{}
	require.NoError(t, defaults.Set(config))

	type plain Config

	require.NoError(t, yaml.Unmarshal([]byte(in), (*plain)(config)))

	return config
}

// creasty/defaults drops a struct default it cannot parse as JSON without
// returning an error, which silently zeroes the whole sub-config.
func TestConfigStructDefaultTagsAreValidJSON(t *testing.T) {
	typ := reflect.TypeOf(Config{})

	for i := range typ.NumField() {
		field := typ.Field(i)

		tag, ok := field.Tag.Lookup("default")
		if !ok || !strings.HasPrefix(tag, "{") {
			continue
		}

		assert.True(t, json.Valid([]byte(tag)), "default tag of %s is not valid JSON: %s", field.Name, tag)
	}
}

func TestConfigDefaults(t *testing.T) {
	config := loadConfig(t, "name: sentry\n")

	require.NotNil(t, config.ProposerDuty)
	assert.True(t, config.ProposerDuty.Enabled)

	require.NotNil(t, config.BeaconCommittees)
	assert.False(t, config.BeaconCommittees.Enabled)

	require.NotNil(t, config.ForkChoice)
	assert.False(t, config.ForkChoice.Enabled)
	assert.Equal(t, 12*time.Second, config.ForkChoice.Interval.Every.Duration)

	require.NotNil(t, config.AttestationData)
	assert.False(t, config.AttestationData.Enabled)

	require.NotNil(t, config.Execution)
	assert.False(t, config.Execution.Enabled)
	assert.True(t, config.Execution.TxPoolContentEnabled)
	assert.Equal(t, 15, config.Execution.FetchInterval)
}

func TestConfigFileOverridesDefaults(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		check func(t *testing.T, config *Config)
	}{
		{
			name: "proposer duties can be turned off",
			in:   "proposerDuty:\n  enabled: false\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.False(t, config.ProposerDuty.Enabled)
			},
		},
		{
			name: "beacon committees can be turned on",
			in:   "beaconCommittees:\n  enabled: true\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.True(t, config.BeaconCommittees.Enabled)
				assert.True(t, config.ProposerDuty.Enabled)
			},
		},
		{
			name: "a partial block keeps the remaining defaults",
			in:   "execution:\n  enabled: true\n  wsAddress: ws://localhost:8546\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.True(t, config.Execution.Enabled)
				assert.True(t, config.Execution.TxPoolContentEnabled)
				assert.Equal(t, 15, config.Execution.FetchInterval)
				assert.Equal(t, "ws://localhost:8546", config.Execution.WSAddress)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, loadConfig(t, tt.in))
		})
	}
}
