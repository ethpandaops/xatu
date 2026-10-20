package configdecode_test

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/xatu/pkg/cannon"
	"github.com/ethpandaops/xatu/pkg/consumoor"
	"github.com/ethpandaops/xatu/pkg/discovery"
	discoveryStatic "github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	discoveryXatu "github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/ethpandaops/xatu/pkg/internal/configdecode"
	"github.com/ethpandaops/xatu/pkg/mimicry"
	mimicryStatic "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/static"
	mimicryXatu "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu/coordinator"
	chSink "github.com/ethpandaops/xatu/pkg/output/clickhouse"
	"github.com/ethpandaops/xatu/pkg/output/http"
	"github.com/ethpandaops/xatu/pkg/output/kafka"
	"github.com/ethpandaops/xatu/pkg/output/s3blobstore"
	"github.com/ethpandaops/xatu/pkg/output/stdout"
	outputXatu "github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor"
	"github.com/ethpandaops/xatu/pkg/sentry"
	"github.com/ethpandaops/xatu/pkg/server"
	"github.com/ethpandaops/xatu/pkg/server/geoip/maxmind"
	redisCluster "github.com/ethpandaops/xatu/pkg/server/store/redis/cluster"
	redisServer "github.com/ethpandaops/xatu/pkg/server/store/redis/server"
)

type yamlDoc []byte

func (d yamlDoc) Unmarshal(v any) error {
	return yaml.Unmarshal(d, v)
}

type testConfig struct {
	Enabled  bool          `yaml:"enabled" default:"true"`
	Workers  int           `yaml:"workers" default:"4"`
	Interval time.Duration `yaml:"interval" default:"5s"`
	Name     string        `yaml:"name" default:"xatu"`
}

func TestDecodeExplicitValuesWin(t *testing.T) {
	conf := &testConfig{}

	require.NoError(t, configdecode.Decode(yamlDoc("enabled: false\nworkers: 0\ninterval: 0s\nname: \"\"\n"), conf))

	assert.Equal(t, testConfig{}, *conf)
}

func TestDecodeOmittedFieldsGetDefaults(t *testing.T) {
	conf := &testConfig{}

	require.NoError(t, configdecode.Decode(yamlDoc("workers: 8\n"), conf))

	assert.Equal(t, testConfig{Enabled: true, Workers: 8, Interval: 5 * time.Second, Name: "xatu"}, *conf)
}

func TestDecodeNilRawKeepsDefaults(t *testing.T) {
	conf := &testConfig{}

	require.NoError(t, configdecode.Decode(nil, conf))

	assert.Equal(t, testConfig{Enabled: true, Workers: 4, Interval: 5 * time.Second, Name: "xatu"}, *conf)
}

// Yaml-then-defaults is the oracle: a field the YAML omits gets its default, even inside a sub-config the YAML creates.
func TestConfigsDefaultTheFieldsYAMLOmits(t *testing.T) {
	sentryOrders := ordersFor[sentry.Config]()
	sentryOrders.got = func(doc []byte) (any, error) { return sentry.ParseConfig(doc) }

	types := map[string]decodeOrders{
		"cannon":                     ordersFor[cannon.Config](),
		"consumoor":                  ordersFor[consumoor.Config](),
		"discovery":                  ordersFor[discovery.Config](),
		"discovery/p2p/static":       ordersFor[discoveryStatic.Config](),
		"discovery/p2p/xatu":         ordersFor[discoveryXatu.Config](),
		"mimicry":                    ordersFor[mimicry.Config](),
		"mimicry/coordinator/static": ordersFor[mimicryStatic.Config](),
		"mimicry/coordinator/xatu":   ordersFor[mimicryXatu.Config](),
		"output/clickhouse":          ordersFor[chSink.Config](),
		"output/http":                ordersFor[http.Config](),
		"output/kafka":               ordersFor[kafka.Config](),
		"output/s3blobstore":         ordersFor[s3blobstore.Config](),
		"output/stdout":              ordersFor[stdout.Config](),
		"output/xatu":                ordersFor[outputXatu.Config](),
		"relaymonitor":               ordersFor[relaymonitor.Config](),
		"sentry":                     sentryOrders,
		"server":                     ordersFor[server.Config](),
		"server/geoip/maxmind":       ordersFor[maxmind.Config](),
		"server/store/redis/cluster": ordersFor[redisCluster.Config](),
		"server/store/redis/server":  ordersFor[redisServer.Config](),
	}

	for name, orders := range types {
		t.Run(name, func(t *testing.T) {
			docs := sampleDocs(t, orders.typ)
			require.NotEmpty(t, docs)

			for _, doc := range docs {
				want, wantErr := orders.want(doc)
				got, gotErr := orders.got(doc)

				if wantErr != nil && gotErr != nil {
					continue
				}

				require.NoError(t, wantErr, "yaml then defaults:\n%s", doc)
				require.NoError(t, gotErr, "decode of:\n%s", doc)
				assert.Equal(t, want, got, "decode of:\n%s", doc)
			}
		})
	}
}

type decodeOrders struct {
	typ  reflect.Type
	want func(doc []byte) (any, error)
	got  func(doc []byte) (any, error)
}

func ordersFor[T any]() decodeOrders {
	return decodeOrders{
		typ: reflect.TypeFor[T](),
		want: func(doc []byte) (any, error) {
			conf := new(T)
			if err := yaml.Unmarshal(doc, conf); err != nil {
				return conf, err
			}

			return conf, defaults.Set(conf)
		},
		got: func(doc []byte) (any, error) {
			conf := new(T)

			return conf, configdecode.Decode(yamlDoc(doc), conf)
		},
	}
}

var (
	yamlUnmarshalerType = reflect.TypeOf((*yaml.Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	durationType        = reflect.TypeOf(time.Duration(0))
)

// sampleDocs returns a document per field path of typ setting it non-zero, plus one per container path creating it empty.
func sampleDocs(t *testing.T, typ reflect.Type) [][]byte {
	t.Helper()

	var docs [][]byte

	add := func(path []any, value any) {
		out, err := yaml.Marshal(nest(path, value))
		require.NoError(t, err)

		docs = append(docs, out)
	}

	var walk func(typ reflect.Type, path []any, stack map[reflect.Type]bool)

	walk = func(typ reflect.Type, path []any, stack map[reflect.Type]bool) {
		for typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}

		custom := reflect.PointerTo(typ).Implements(yamlUnmarshalerType) || reflect.PointerTo(typ).Implements(textUnmarshalerType)

		if custom {
			for _, candidate := range []any{"7s", "x", "7", "true", "localhost:7"} {
				add(path, candidate)
			}
		}

		switch typ.Kind() {
		case reflect.Struct:
			if stack[typ] || !hasExportedField(typ) {
				return
			}

			stack[typ] = true
			defer delete(stack, typ)

			if len(path) > 0 {
				add(path, map[string]any{})
			}

			for i := range typ.NumField() {
				field := typ.Field(i)
				if !field.IsExported() {
					continue
				}

				key, inline := yamlKey(field)
				if key == "-" {
					continue
				}

				if inline {
					walk(field.Type, path, stack)

					continue
				}

				walk(field.Type, appendPath(path, key), stack)
			}
		case reflect.Slice, reflect.Array:
			if typ.Elem().Kind() == reflect.Uint8 {
				add(path, "eA==")

				return
			}

			add(path, []any{})
			walk(typ.Elem(), appendPath(path, 0), stack)
		case reflect.Map:
			if typ.Key().Kind() != reflect.String {
				return
			}

			add(path, map[string]any{})
			walk(typ.Elem(), appendPath(path, "k"), stack)
		case reflect.Bool:
			add(path, true)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if typ == durationType {
				add(path, "7s")

				return
			}

			add(path, 7)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			add(path, 7)
		case reflect.Float32, reflect.Float64:
			add(path, 1.5)
		case reflect.String:
			if !custom {
				add(path, "x")
			}
		default:
		}
	}

	walk(typ, nil, map[reflect.Type]bool{})

	return docs
}

func hasExportedField(typ reflect.Type) bool {
	for i := range typ.NumField() {
		if typ.Field(i).IsExported() {
			return true
		}
	}

	return false
}

func yamlKey(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("yaml")
	name, opts, _ := strings.Cut(tag, ",")

	if strings.Contains(opts, "inline") {
		return "", true
	}

	if name == "" {
		return strings.ToLower(field.Name), false
	}

	return name, false
}

func appendPath(path []any, elem any) []any {
	out := make([]any, 0, len(path)+1)
	out = append(out, path...)

	return append(out, elem)
}

func nest(path []any, value any) any {
	if len(path) == 0 {
		return value
	}

	child := nest(path[1:], value)

	switch key := path[0].(type) {
	case string:
		return map[string]any{key: child}
	case int:
		return []any{child}
	default:
		panic(fmt.Sprintf("unexpected path element %v", key))
	}
}
