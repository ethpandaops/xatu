package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/output"
	outputXatu "github.com/ethpandaops/xatu/pkg/output/xatu"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestConfigLoadersExpandEnv(t *testing.T) {
	t.Setenv("XATU_TEST_INSTANCE", "from-env")
	t.Setenv("XATU_TEST_DSN", "postgres://xatu:s3\"c #r\tet@db:5432/xatu")

	t.Run("server", func(t *testing.T) {
		config, err := loadServerConfigFromFile(writeConfig(t, "persistence:\n  connectionString: ${XATU_TEST_DSN}\n"))
		require.NoError(t, err)
		assert.Equal(t, "postgres://xatu:s3\"c #r\tet@db:5432/xatu", config.Persistence.ConnectionString)
		assert.Equal(t, ":8080", config.Addr)
	})

	t.Run("mimicry", func(t *testing.T) {
		config, err := loadMimicryConfigFromFile(writeConfig(t, "name: ${XATU_TEST_INSTANCE}\n"))
		require.NoError(t, err)
		assert.Equal(t, "from-env", config.Name)
	})

	t.Run("discovery", func(t *testing.T) {
		config, err := loadDiscoveryConfigFromFile(writeConfig(t, "metricsAddr: \"${XATU_TEST_METRICS_ADDR:-:9191}\"\n"))
		require.NoError(t, err)
		assert.Equal(t, ":9191", config.MetricsAddr)
	})

	t.Run("relay-monitor", func(t *testing.T) {
		config, err := loadRelayMonitorConfigFromFile(writeConfig(t, "name: ${XATU_TEST_INSTANCE}\n"))
		require.NoError(t, err)
		assert.Equal(t, "from-env", config.Name)
	})

	t.Run("unset variable", func(t *testing.T) {
		_, err := loadServerConfigFromFile(writeConfig(t, "persistence:\n  connectionString: \"${XATU_TEST_UNSET_DSN}\"\n"))
		require.EqualError(t, err, "config references unset or empty environment variables: XATU_TEST_UNSET_DSN")
	})
}

func TestSentryCannonConsumoorLoadersExpandEnv(t *testing.T) {
	loaders := map[string]func(file string) (metricsAddr string, err error){
		"sentry": func(file string) (string, error) {
			config, err := loadSentryConfigFromFile(file, false)
			if err != nil {
				return "", err
			}

			return config.MetricsAddr, nil
		},
		"cannon": func(file string) (string, error) {
			config, err := loadcannonConfigFromFile(file)
			if err != nil {
				return "", err
			}

			return config.MetricsAddr, nil
		},
		"consumoor": func(file string) (string, error) {
			config, err := loadConsumoorConfigFromFile(file)
			if err != nil {
				return "", err
			}

			return config.MetricsAddr, nil
		},
	}

	values := []string{"true", "0x1F", "1e3", "007", "null", "~", "yes", "a: b", "- x", "#x"}

	for name, load := range loaders {
		t.Run(name, func(t *testing.T) {
			for _, value := range values {
				t.Run(value, func(t *testing.T) {
					t.Setenv("XATU_TEST_VALUE", value)

					got, err := load(writeConfig(t, "metricsAddr: ${XATU_TEST_VALUE}\n"))
					require.NoError(t, err)
					assert.Equal(t, value, got)
				})
			}

			t.Run("default", func(t *testing.T) {
				got, err := load(writeConfig(t, "metricsAddr: ${XATU_TEST_UNSET_ADDR:-:9191}\n"))
				require.NoError(t, err)
				assert.Equal(t, ":9191", got)
			})

			t.Run("unset variables", func(t *testing.T) {
				_, err := load(writeConfig(t, "metricsAddr: ${XATU_TEST_UNSET_A}\nlogging: ${XATU_TEST_UNSET_B}\n"))
				require.EqualError(t, err, "config references unset or empty environment variables: XATU_TEST_UNSET_A, XATU_TEST_UNSET_B")
			})

			t.Run("example config without placeholders", func(t *testing.T) {
				got, err := load(filepath.Join("..", "example_"+name+".yaml"))
				require.NoError(t, err)
				assert.Equal(t, ":9090", got)
			})
		})
	}
}

func TestSentryOutputHeadersExpandEnv(t *testing.T) {
	t.Setenv("XATU_AUTH", "dXNlcjpwYXNz")

	config, err := loadSentryConfigFromFile(writeConfig(t, `name: sentry
outputs:
  - name: xatu
    type: xatu
    config:
      address: xatu:8080
      headers:
        authorization: Basic ${XATU_AUTH}
`), false)
	require.NoError(t, err)
	require.Len(t, config.Outputs, 1)
	assert.Equal(t, output.SinkTypeXatu, config.Outputs[0].SinkType)

	var sinkConfig outputXatu.Config

	require.NoError(t, config.Outputs[0].Config.Unmarshal(&sinkConfig))
	assert.Equal(t, map[string]string{"authorization": "Basic dXNlcjpwYXNz"}, sinkConfig.Headers)
}

func TestSentryLoaderAllowsMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")

	config, err := loadSentryConfigFromFile(missing, true)
	require.NoError(t, err)
	assert.Equal(t, ":9090", config.MetricsAddr)

	_, err = loadSentryConfigFromFile(missing, false)
	require.ErrorIs(t, err, os.ErrNotExist)
}
