package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

type envTestConfig struct {
	Name     string            `yaml:"name"`
	Password string            `yaml:"password"`
	Port     int               `yaml:"port"`
	Enabled  bool              `yaml:"enabled"`
	Topic    string            `yaml:"topic"`
	Labels   map[string]string `yaml:"labels"`
	Items    []string          `yaml:"items"`
}

func TestDecodeConfigExpandsEnv(t *testing.T) {
	tests := []struct {
		name    string
		content string
		env     map[string]string
		want    envTestConfig
		wantErr string
	}{
		{
			name:    "variable is replaced by its value",
			content: "name: xatu\npassword: \"${XATU_TEST_PASSWORD}\"\n",
			env:     map[string]string{"XATU_TEST_PASSWORD": "s3cret"},
			want:    envTestConfig{Name: "xatu", Password: "s3cret"},
		},
		{
			name:    "several variables in one value",
			content: "password: postgres://${XATU_TEST_USER}:${XATU_TEST_DB_PASSWORD}@db:5432/xatu",
			env:     map[string]string{"XATU_TEST_USER": "alice", "XATU_TEST_DB_PASSWORD": "pw"},
			want:    envTestConfig{Password: "postgres://alice:pw@db:5432/xatu"},
		},
		{
			name:    "variables in nested mappings and sequences",
			content: "labels:\n  region: ${XATU_TEST_REGION}\nitems:\n  - ${XATU_TEST_REGION}\n  - fixed\n",
			env:     map[string]string{"XATU_TEST_REGION": "x"},
			want:    envTestConfig{Labels: map[string]string{"region": "x"}, Items: []string{"x", "fixed"}},
		},
		{
			name:    "aliases receive the expanded anchor value",
			content: "password: &pw ${XATU_TEST_ANCHOR}\nname: *pw\n",
			env:     map[string]string{"XATU_TEST_ANCHOR": "a"},
			want:    envTestConfig{Password: "a", Name: "a"},
		},
		{
			name:    "variable set to empty is an error",
			content: "password: \"${XATU_TEST_EMPTY}\"",
			env:     map[string]string{"XATU_TEST_EMPTY": ""},
			wantErr: "config references unset or empty environment variables: XATU_TEST_EMPTY",
		},
		{
			name:    "unset variable is an error naming each missing variable once",
			content: "name: ${XATU_TEST_MISSING_A}\npassword: ${XATU_TEST_PRESENT}\ntopic: ${XATU_TEST_MISSING_B}\nitems: [\"${XATU_TEST_MISSING_A}\"]\n",
			env:     map[string]string{"XATU_TEST_PRESENT": "x"},
			wantErr: "config references unset or empty environment variables: XATU_TEST_MISSING_A, XATU_TEST_MISSING_B",
		},
		{
			name:    "default used when variable unset",
			content: "name: ${XATU_TEST_DEFAULT_UNSET:-unset}",
			want:    envTestConfig{Name: "unset"},
		},
		{
			name:    "default used when variable empty",
			content: "name: ${XATU_TEST_DEFAULT_EMPTY:-fallback}",
			env:     map[string]string{"XATU_TEST_DEFAULT_EMPTY": ""},
			want:    envTestConfig{Name: "fallback"},
		},
		{
			name:    "default ignored when variable set",
			content: "name: ${XATU_TEST_DEFAULT_SET:-fallback}",
			env:     map[string]string{"XATU_TEST_DEFAULT_SET": "set"},
			want:    envTestConfig{Name: "set"},
		},
		{
			name:    "empty default is an empty string",
			content: "name: ${XATU_TEST_EMPTY_DEFAULT:-}",
			want:    envTestConfig{Name: ""},
		},
		{
			name:    "unquoted variables fill numbers and booleans",
			content: "port: ${XATU_TEST_PORT}\nenabled: ${XATU_TEST_ENABLED}\n",
			env:     map[string]string{"XATU_TEST_PORT": "8080", "XATU_TEST_ENABLED": "true"},
			want:    envTestConfig{Port: 8080, Enabled: true},
		},
		{
			name:    "unquoted value that looks like null or a number stays a string",
			content: "name: ${XATU_TEST_NULL}\npassword: ${XATU_TEST_NUMBER}\n",
			env:     map[string]string{"XATU_TEST_NULL": "null", "XATU_TEST_NUMBER": "0123"},
			want:    envTestConfig{Name: "null", Password: "0123"},
		},
		{
			name:    "literal dollar signs and bare $VAR are unchanged",
			content: "password: pa$$word$XATU_TEST_DOLLAR\nname: $5 $ {XATU_TEST_DOLLAR}\ntopic: ${XATU_TEST_DOLLAR}$\n",
			env:     map[string]string{"XATU_TEST_DOLLAR": "x"},
			want:    envTestConfig{Password: "pa$$word$XATU_TEST_DOLLAR", Name: "$5 $ {XATU_TEST_DOLLAR}", Topic: "x$"},
		},
		{
			name:    "comments are not expanded",
			content: "# password: ${XATU_TEST_MISSING}\nname: ${XATU_TEST_COMMENT} # ${XATU_TEST_MISSING}\n",
			env:     map[string]string{"XATU_TEST_COMMENT": "x"},
			want:    envTestConfig{Name: "x"},
		},
		{
			name:    "mapping keys are not expanded",
			content: "labels:\n  ${XATU_TEST_KEY}: ${XATU_TEST_KEY}\n",
			env:     map[string]string{"XATU_TEST_KEY": "x"},
			want:    envTestConfig{Labels: map[string]string{"${XATU_TEST_KEY}": "x"}},
		},
		{
			name:    "kafka topic pattern variables are unchanged",
			content: "topic: \"xatu-${EVENT_NAME}-${event-name}\"",
			env:     map[string]string{"EVENT_NAME": "x"},
			want:    envTestConfig{Topic: "xatu-${EVENT_NAME}-${event-name}"},
		},
		{
			name:    "unclosed and empty placeholders are unchanged",
			content: "items: [\"${}\", \"${XATU_TEST_UNCLOSED\", \"$${XATU_TEST_UNCLOSED\", \"${XATU_TEST_UNCLOSED:x}\"]",
			env:     map[string]string{"XATU_TEST_UNCLOSED": "x"},
			want:    envTestConfig{Items: []string{"${}", "${XATU_TEST_UNCLOSED", "$${XATU_TEST_UNCLOSED", "${XATU_TEST_UNCLOSED:x}"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			var got envTestConfig

			err := decodeConfig([]byte(tt.content), &got)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeConfigKeepsSpecialCharactersInValues(t *testing.T) {
	values := map[string]string{
		"tab":            "a\tb\\tc",
		"comment marker": "secret #not-a-comment",
		"newline":        "line1\nport: 9999",
		"double quote":   `pa"ss`,
		"single quote":   "it's",
		"yaml syntax":    "{a: [b, c]}: - &x *y",
		"leading space":  "  padded  ",
	}

	placeholders := map[string]string{
		"unquoted":      "password: ${XATU_TEST_SPECIAL}\n",
		"double quoted": "password: \"${XATU_TEST_SPECIAL}\"\n",
		"single quoted": "password: '${XATU_TEST_SPECIAL}'\n",
		"block":         "password: |-\n  ${XATU_TEST_SPECIAL}\n",
	}

	for valueName, value := range values {
		for styleName, content := range placeholders {
			t.Run(valueName+"/"+styleName, func(t *testing.T) {
				t.Setenv("XATU_TEST_SPECIAL", value)

				var got envTestConfig

				require.NoError(t, decodeConfig([]byte(content), &got))
				assert.Equal(t, envTestConfig{Password: value}, got)
			})
		}
	}
}

func TestDecodeConfigWithoutPlaceholdersIsUnchanged(t *testing.T) {
	t.Setenv("HOME", "/root")

	content := []byte("name: $HOME\r\n# comment ${HOME}\r\npassword: \"a$b{c}\"\r\nport: 8080\r\nlabels:\n  $: $\n")

	var want, got envTestConfig

	require.NoError(t, yaml.Unmarshal(content, &want))
	require.NoError(t, decodeConfig(content, &got))
	assert.Equal(t, want, got)
	assert.Equal(t, "$HOME", got.Name)
}

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
