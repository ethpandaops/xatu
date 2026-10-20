package sentry

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// creasty/defaults silently zeroes a pointer sub-config whose default tag is not valid JSON.
func TestConfigStructDefaultTagsAreValidJSON(t *testing.T) {
	seen := map[reflect.Type]bool{}

	var walk func(typ reflect.Type, path string)

	walk = func(typ reflect.Type, path string) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
			typ = typ.Elem()
		}

		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}

		seen[typ] = true

		for i := range typ.NumField() {
			field := typ.Field(i)

			if tag, ok := field.Tag.Lookup("default"); ok && (strings.HasPrefix(tag, "{") || strings.HasPrefix(tag, "[")) {
				assert.True(t, json.Valid([]byte(tag)), "default tag of %s.%s is not valid JSON: %s", path, field.Name, tag)
			}

			walk(field.Type, path+"."+field.Name)
		}
	}

	walk(reflect.TypeOf(Config{}), "Config")
}

func TestExampleConfigHasNoUnknownKeys(t *testing.T) {
	data, err := os.ReadFile("../../example_sentry.yaml")
	require.NoError(t, err)

	type plain Config

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	require.NoError(t, decoder.Decode(&plain{}))
}

func TestConfigDefaults(t *testing.T) {
	config, err := ParseConfig([]byte("name: sentry\n"))
	require.NoError(t, err)

	assert.True(t, config.ProposerDuty.Enabled)
	assert.False(t, config.BeaconCommittees.Enabled)
	assert.False(t, config.AttestationData.Enabled)
	assert.False(t, config.ForkChoice.Enabled)
	assert.Equal(t, 12*time.Second, config.ForkChoice.Interval.Every.Duration)
	assert.Equal(t, 12*time.Second, config.AttestationData.Interval.Every.Duration)

	execution := config.Execution
	assert.False(t, execution.Enabled)
	assert.True(t, execution.TxPoolContentEnabled)
	assert.Equal(t, 45, execution.FetchInterval)
	assert.Equal(t, 3000, execution.ProcessingInterval)
	assert.Equal(t, 15, execution.RpcBatchSize)
	assert.Equal(t, 3, execution.MaxConcurrency)
	assert.Equal(t, 10000, execution.QueueSize)
	assert.Equal(t, 40, execution.ProcessorWorkerCount)
	assert.Equal(t, 384, execution.PruneDuration)

	require.NotNil(t, execution.StateSize)
	assert.False(t, execution.StateSize.Enabled)
	assert.Equal(t, "head", execution.StateSize.TriggerMode)
	assert.Equal(t, 12, execution.StateSize.IntervalSeconds)
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
			name: "a partial execution block keeps the remaining defaults",
			in:   "execution:\n  enabled: true\n  rpcAddress: http://localhost:8545\n  fetchInterval: 30\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.True(t, config.Execution.Enabled)
				assert.True(t, config.Execution.TxPoolContentEnabled)
				assert.Equal(t, 30, config.Execution.FetchInterval)
				assert.Equal(t, 3000, config.Execution.ProcessingInterval)
				assert.Equal(t, "http://localhost:8545", config.Execution.RPCAddress)
			},
		},
		{
			name: "txpool_content polling can be turned off",
			in:   "execution:\n  enabled: true\n  txPoolContentEnabled: false\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.False(t, config.Execution.TxPoolContentEnabled)
			},
		},
		{
			name: "an enabled state size block polls on head by default",
			in:   "execution:\n  stateSize:\n    enabled: true\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.True(t, config.Execution.StateSize.Enabled)
				assert.Equal(t, "head", config.Execution.StateSize.TriggerMode)
				assert.Equal(t, 12, config.Execution.StateSize.IntervalSeconds)
			},
		},
		{
			name: "interval state size polling gets a default interval",
			in:   "execution:\n  stateSize:\n    enabled: true\n    triggerMode: interval\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				assert.Equal(t, "interval", config.Execution.StateSize.TriggerMode)
				assert.Equal(t, 12, config.Execution.StateSize.IntervalSeconds)
			},
		},
		{
			name: "null sub-config blocks fall back to their defaults",
			in:   "proposerDuty:\nforkChoice:\nbeaconCommittees: null\nexecution:\n",
			check: func(t *testing.T, config *Config) {
				t.Helper()
				require.NotNil(t, config.ProposerDuty)
				assert.True(t, config.ProposerDuty.Enabled)
				require.NotNil(t, config.ForkChoice)
				assert.Equal(t, 12*time.Second, config.ForkChoice.Interval.Every.Duration)
				require.NotNil(t, config.BeaconCommittees)
				require.NotNil(t, config.Execution)
				assert.True(t, config.Execution.TxPoolContentEnabled)
				assert.NoError(t, config.ForkChoice.Validate())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseConfig([]byte(tt.in))
			require.NoError(t, err)

			tt.check(t, config)
		})
	}
}
