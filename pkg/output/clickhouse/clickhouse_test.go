package clickhouse

import (
	"testing"
	"time"

	chwriter "github.com/ethpandaops/xatu/pkg/clickhouse"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

// fakeRoute is a minimal route.Route implementation usable as test fixture.
type fakeRoute struct {
	table  string
	events []xatu.Event_Name
}

func (r *fakeRoute) TableName() string                         { return r.table }
func (r *fakeRoute) EventNames() []xatu.Event_Name             { return r.events }
func (r *fakeRoute) ShouldProcess(_ *xatu.DecoratedEvent) bool { return true }
func (r *fakeRoute) NewBatch() route.ColumnarBatch             { return nil }

func TestFilterRoutesByTablePrefix(t *testing.T) {
	all := []route.Route{
		&fakeRoute{table: "canonical_beacon_block"},
		&fakeRoute{table: "canonical_beacon_blob_sidecar"},
		&fakeRoute{table: "libp2p_gossip_block"},
		&fakeRoute{table: "mev_relay_bid_trace"},
		&fakeRoute{table: "execution_block_metrics"},
	}

	tests := []struct {
		name     string
		prefixes []string
		expected []string
	}{
		{
			name:     "single prefix matches subset",
			prefixes: []string{"canonical_"},
			expected: []string{"canonical_beacon_block", "canonical_beacon_blob_sidecar"},
		},
		{
			name:     "multiple prefixes union",
			prefixes: []string{"canonical_", "mev_"},
			expected: []string{"canonical_beacon_block", "canonical_beacon_blob_sidecar", "mev_relay_bid_trace"},
		},
		{
			name:     "non-matching prefix returns empty",
			prefixes: []string{"nonexistent_"},
			expected: []string{},
		},
		{
			name:     "exact prefix match",
			prefixes: []string{"libp2p_gossip_block"},
			expected: []string{"libp2p_gossip_block"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterRoutesByTablePrefix(all, tt.prefixes)
			gotTables := make([]string, 0, len(got))

			for _, r := range got {
				gotTables = append(gotTables, r.TableName())
			}

			assert.ElementsMatch(t, tt.expected, gotTables)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	t.Run("nil config errors", func(t *testing.T) {
		var c *Config

		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config is required")
	})

	t.Run("missing dsn errors via embedded writer config", func(t *testing.T) {
		c := &Config{}

		err := c.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dsn")
	})
}

func TestNewRequiresConfig(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	_, err := New("test", nil, log, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config is required")
}

func TestNewRejectsEmptyMatchPrefix(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	cfg := &Config{
		Config: chwriter.Config{
			DSN: "clickhouse://localhost:9000/default",
			ChGo: chwriter.ChGoConfig{
				DialTimeout:       5 * time.Second,
				ReadTimeout:       5 * time.Second,
				QueryTimeout:      5 * time.Second,
				RetryBaseDelay:    100 * time.Millisecond,
				RetryMaxDelay:     1 * time.Second,
				MaxConns:          4,
				MinConns:          1,
				ConnMaxLifetime:   1 * time.Hour,
				ConnMaxIdleTime:   10 * time.Minute,
				HealthCheckPeriod: 30 * time.Second,
				AdaptiveLimiter: chwriter.AdaptiveLimiterConfig{
					Enabled: boolPtr(false),
				},
			},
		},
		MetricsSubsystem:        "test_no_match",
		RestrictToTablePrefixes: []string{"this_prefix_matches_nothing_"},
	}

	_, err := New("test", cfg, log, nil, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matched no routes")
}

func TestNewSinkSatisfiesOutputSinkInterface(t *testing.T) {
	// Compile-time guarantee at clickhouse.go:37; this test simply
	// keeps the assertion from being silently removed in future edits.
	var _ outputSink = (*Sink)(nil)
}
