package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	chwriter "github.com/ethpandaops/xatu/pkg/clickhouse"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	chrouter "github.com/ethpandaops/xatu/pkg/clickhouse/router"
	"github.com/ethpandaops/xatu/pkg/clickhouse/telemetry"
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

// stubWriter is a chWriter implementation for testing HandleNewDecoratedEvents
// without dialing a real ClickHouse instance.
type stubWriter struct {
	flushCalled atomic.Int32
	lastArgs    map[string][]*xatu.DecoratedEvent
	result      *chwriter.FlushResult
}

func (w *stubWriter) Start(_ context.Context) error { return nil }
func (w *stubWriter) Stop(_ context.Context) error  { return nil }
func (w *stubWriter) FlushTableEvents(
	_ context.Context,
	tableEvents map[string][]*xatu.DecoratedEvent,
) *chwriter.FlushResult {
	w.flushCalled.Add(1)
	w.lastArgs = tableEvents

	if w.result != nil {
		return w.result
	}

	return &chwriter.FlushResult{}
}

// makeTestSink builds a Sink wired to a stub writer and a real router with the
// supplied fake routes. Each call uses a unique metrics subsystem to avoid
// promauto registration collisions across subtests.
func makeTestSink(
	t *testing.T,
	writer chWriter,
	routes []route.Route,
	restrictPrefixes []string,
	filterCfg *xatu.EventFilterConfig,
) *Sink {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	subsystem := fmt.Sprintf("sink_test_%d", time.Now().UnixNano())
	metrics := telemetry.NewMetrics("xatu_sink_test", subsystem)

	router := chrouter.New(log.WithField("c", "test_router"), routes, nil, nil, metrics)

	if filterCfg == nil {
		filterCfg = &xatu.EventFilterConfig{}
	}

	filter, err := xatu.NewEventFilter(filterCfg)
	require.NoError(t, err)

	return &Sink{
		name:             "test",
		log:              log.WithField("c", "test_sink"),
		writer:           writer,
		router:           router,
		filter:           filter,
		restrictPrefixes: restrictPrefixes,
	}
}

// makeTestEvent returns a minimal DecoratedEvent with the given Event_Name.
func makeTestEvent(name xatu.Event_Name) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: name,
			Id:   "test-event",
		},
	}
}

func TestHandleNewDecoratedEvents_EmptySliceShortCircuits(t *testing.T) {
	w := &stubWriter{}
	sink := makeTestSink(t, w, nil, nil, nil)

	err := sink.HandleNewDecoratedEvents(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, int32(0), w.flushCalled.Load(), "empty slice should not call flush")

	err = sink.HandleNewDecoratedEvents(context.Background(), []*xatu.DecoratedEvent{})
	require.NoError(t, err)
	assert.Equal(t, int32(0), w.flushCalled.Load(), "zero-length slice should not call flush")
}

func TestHandleNewDecoratedEvents_NilEventsAreSkipped(t *testing.T) {
	w := &stubWriter{}
	sink := makeTestSink(t, w, nil, nil, nil)

	events := []*xatu.DecoratedEvent{nil, nil, nil}
	err := sink.HandleNewDecoratedEvents(context.Background(), events)
	require.NoError(t, err)
	assert.Equal(t, int32(0), w.flushCalled.Load(),
		"all-nil slice should leave tableEvents empty and skip flush")
}

func TestHandleNewDecoratedEvents_FilterDropsAllEvents(t *testing.T) {
	w := &stubWriter{}

	// Build a filter that drops every event we'll send by excluding the
	// event name we use in the fixture.
	const dropName = "BEACON_API_ETH_V2_BEACON_BLOCK"

	filterCfg := &xatu.EventFilterConfig{ExcludeEventNames: []string{dropName}}

	routes := []route.Route{
		&fakeRoute{
			table:  "canonical_beacon_block",
			events: []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK},
		},
	}

	sink := makeTestSink(t, w, routes, nil, filterCfg)

	events := []*xatu.DecoratedEvent{
		makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK),
		makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK),
	}

	err := sink.HandleNewDecoratedEvents(context.Background(), events)
	require.NoError(t, err)
	assert.Equal(t, int32(0), w.flushCalled.Load(),
		"filter-dropped events should not reach the writer")
}

func TestHandleNewDecoratedEvents_StatusErroredReturnsError(t *testing.T) {
	w := &stubWriter{}

	// No routes registered for BEACON_API_ETH_V1_EVENTS_HEAD_V2 → router returns
	// StatusErrored (the event is not on the unsupported list and has no route).
	sink := makeTestSink(t, w, nil, nil, nil)

	err := sink.HandleNewDecoratedEvents(
		context.Background(),
		[]*xatu.DecoratedEvent{makeTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2)},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no route registered")
	assert.Contains(t, err.Error(), "BEACON_API_ETH_V1_EVENTS_HEAD_V2")
	assert.NotContains(t, err.Error(), "restricted to prefixes",
		"unrestricted sink should use the simpler error message")
	assert.Equal(t, int32(0), w.flushCalled.Load(),
		"errored routing must not advance the writer")
}

func TestHandleNewDecoratedEvents_StatusErroredIncludesRestrictPrefixes(t *testing.T) {
	w := &stubWriter{}
	prefixes := []string{"canonical_", "mev_"}
	sink := makeTestSink(t, w, nil, prefixes, nil)

	err := sink.HandleNewDecoratedEvents(
		context.Background(),
		[]*xatu.DecoratedEvent{makeTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2)},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restricted to prefixes")
	assert.Contains(t, err.Error(), "canonical_")
	assert.Contains(t, err.Error(), "mev_")
}

func TestHandleNewDecoratedEvents_FlushFailurePropagates(t *testing.T) {
	flushErr := errors.New("clickhouse insert refused")
	w := &stubWriter{
		result: &chwriter.FlushResult{
			TableErrors: map[string]error{
				"canonical_beacon_block": flushErr,
			},
		},
	}

	routes := []route.Route{
		&fakeRoute{
			table:  "canonical_beacon_block",
			events: []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK},
		},
	}

	sink := makeTestSink(t, w, routes, nil, nil)

	err := sink.HandleNewDecoratedEvents(
		context.Background(),
		[]*xatu.DecoratedEvent{makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK)},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clickhouse flush")
	assert.ErrorIs(t, err, flushErr,
		"caller error must wrap the underlying writer error so checkpoint logic can inspect it")
	assert.Equal(t, int32(1), w.flushCalled.Load())
}

func TestHandleNewDecoratedEvents_InvalidEventsLoggedButReturnsNil(t *testing.T) {
	// FlushResult with InvalidEvents but no TableErrors → sink should warn
	// and return nil. Cannon's deriver advances its checkpoint on this.
	w := &stubWriter{
		result: &chwriter.FlushResult{
			InvalidEvents: []*xatu.DecoratedEvent{
				makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK),
			},
		},
	}

	routes := []route.Route{
		&fakeRoute{
			table:  "canonical_beacon_block",
			events: []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK},
		},
	}

	sink := makeTestSink(t, w, routes, nil, nil)

	err := sink.HandleNewDecoratedEvents(
		context.Background(),
		[]*xatu.DecoratedEvent{makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK)},
	)
	require.NoError(t, err, "InvalidEvents alone must not block checkpoint advance")
	assert.Equal(t, int32(1), w.flushCalled.Load())
}

func TestHandleNewDecoratedEvents_RoutesEventsToCorrectTables(t *testing.T) {
	w := &stubWriter{}

	routes := []route.Route{
		&fakeRoute{
			table:  "canonical_beacon_block",
			events: []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK},
		},
		&fakeRoute{
			table:  "canonical_beacon_block_withdrawal",
			events: []xatu.Event_Name{xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL},
		},
	}

	sink := makeTestSink(t, w, routes, nil, nil)

	events := []*xatu.DecoratedEvent{
		makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK),
		makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL),
		makeTestEvent(xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL),
	}

	err := sink.HandleNewDecoratedEvents(context.Background(), events)
	require.NoError(t, err)
	assert.Equal(t, int32(1), w.flushCalled.Load(),
		"a single sink call must produce a single flush regardless of fan-out")

	require.NotNil(t, w.lastArgs)
	assert.Len(t, w.lastArgs["canonical_beacon_block"], 1)
	assert.Len(t, w.lastArgs["canonical_beacon_block_withdrawal"], 2)
}
