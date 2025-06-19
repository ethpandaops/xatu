//go:build integration
// +build integration

package clmimicry

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_FullEventFlow(t *testing.T) {
	// Create configuration
	config := &ConfigV2{
		Name:     "test-mimicry",
		LogLevel: "debug",
		Events: EventConfigV2{
			Enabled: true,
		},
		Sharding: ShardingConfig{
			Topics: map[string]*TopicShardingConfig{
				".*beacon_block.*": {
					ActiveShards: generateShardRange(0, 511), // 100% sampling
				},
				".*attestation.*": {
					ActiveShards: generateShardRange(0, 255), // 50% sampling
				},
				".*": {
					ActiveShards: []uint64{0}, // Minimal sampling for others
				},
			},
			NoShardingKeyEvents: &NoShardingKeyConfig{
				Enabled: true,
			},
		},
	}

	// Create components
	mockSinks := new(MockSinks)
	clientMeta := &xatu.ClientMeta{
		Name:    "test-client",
		Version: "1.0.0",
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: "mainnet",
			},
		},
	}

	processor, err := NewEventProcessorV2(config, mockSinks, clientMeta, logrus.New())
	require.NoError(t, err)

	// Test different event types
	ctx := context.Background()

	t.Run("Group A Events", func(t *testing.T) {
		// Beacon block should always be processed (100% sampling)
		mockSinks.On("HandleNewDecoratedEvent", ctx, mock.AnythingOfType("*xatu.DecoratedEvent")).Return(nil).Once()

		err := processor.ProcessEvent(ctx, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK, &TraceEventData{
			Metadata: map[string]interface{}{
				"msg_id": "block-123",
				"topic":  "beacon_block",
			},
		})
		assert.NoError(t, err)
	})

	t.Run("Group B Events", func(t *testing.T) {
		// JOIN event with attestation topic (50% sampling)
		// May or may not be called depending on shard calculation
		mockSinks.On("HandleNewDecoratedEvent", ctx, mock.AnythingOfType("*xatu.DecoratedEvent")).Return(nil).Maybe()

		err := processor.ProcessEvent(ctx, xatu.Event_LIBP2P_TRACE_JOIN, map[string]interface{}{
			"topic": "beacon_attestation_1",
		})
		assert.NoError(t, err)
	})

	t.Run("Group C Events", func(t *testing.T) {
		// IWANT event (minimal sampling - likely filtered)
		err := processor.ProcessEvent(ctx, xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT, map[string]interface{}{
			"msg_id": "want-456",
		})
		assert.NoError(t, err)
	})

	t.Run("Group D Events", func(t *testing.T) {
		// ADD_PEER should be processed when enabled
		mockSinks.On("HandleNewDecoratedEvent", ctx, mock.AnythingOfType("*xatu.DecoratedEvent")).Return(nil).Once()

		err := processor.ProcessEvent(ctx, xatu.Event_LIBP2P_TRACE_ADD_PEER, map[string]interface{}{
			"peer_id": "12D3KooWTest",
		})
		assert.NoError(t, err)
	})

	mockSinks.AssertExpectations(t)
}

func TestIntegration_RPCMetaEventProcessing(t *testing.T) {
	config := &ConfigV2{
		Events: EventConfigV2{
			Enabled: true,
		},
		Sharding: ShardingConfig{
			Topics: map[string]*TopicShardingConfig{
				".*beacon_block.*": {
					ActiveShards: generateShardRange(0, 511), // 100% sampling
				},
			},
		},
	}

	mockSinks := new(MockSinks)
	clientMeta := &xatu.ClientMeta{
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Name: "mainnet",
			},
		},
	}

	processor, err := NewEventProcessorV2(config, mockSinks, clientMeta, logrus.New())
	require.NoError(t, err)

	// Create RPC meta handler
	rpcHandler := NewRPCMetaHandlerV2()

	// Simulate RPC event with meta events
	peerID, _ := peer.Decode("12D3KooWTest")

	// Create a mock RPC message with various meta events
	metaEvents := []RPCMetaEvent{
		{
			EventType: xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE,
			MsgID:     "msg-001",
			Topic:     "beacon_block",
			Data:      map[string]interface{}{"peer_id": peerID.String()},
		},
		{
			EventType: xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT,
			Topic:     "beacon_block",
			Data:      map[string]interface{}{"peer_id": peerID.String()},
		},
		{
			EventType: xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION,
			Topic:     "unknown_topic",
			Data:      map[string]interface{}{"peer_id": peerID.String()},
		},
	}

	// Expect at least the beacon_block related events to be processed
	mockSinks.On("HandleNewDecoratedEvents", mock.Anything, mock.MatchedBy(func(events []*xatu.DecoratedEvent) bool {
		return len(events) >= 2 // At least the two beacon_block events
	})).Return(nil).Once()

	ctx := context.Background()
	err = processor.ProcessRPCEvent(ctx, xatu.Event_LIBP2P_TRACE_RECV_RPC, nil, metaEvents)
	assert.NoError(t, err)

	mockSinks.AssertExpectations(t)
}

func TestIntegration_MetricsCollection(t *testing.T) {
	// Create a new registry for testing
	reg := prometheus.NewRegistry()

	// Create metrics with test registry
	metrics := &MetricsV2{
		eventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "test",
				Name:      "events_total",
			},
			[]string{"event_type", "network", "sharding_group"},
		),
		eventsSampled: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "test",
				Name:      "events_sampled",
			},
			[]string{"event_type", "network", "topic_pattern"},
		),
		eventsDropped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "test",
				Name:      "events_dropped",
			},
			[]string{"event_type", "network", "reason"},
		),
	}

	reg.MustRegister(metrics.eventsTotal, metrics.eventsSampled, metrics.eventsDropped)

	// Simulate event processing
	metrics.RecordEvent(
		xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK,
		"mainnet",
		GroupA,
		true,
		"",
		".*beacon_block.*",
	)

	metrics.RecordEvent(
		xatu.Event_LIBP2P_TRACE_JOIN,
		"mainnet",
		GroupB,
		false,
		"group_b_no_config",
		"",
	)

	// Verify metrics
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	// Check events_total
	totalMetric := findMetric(metricFamilies, "test_events_total")
	require.NotNil(t, totalMetric)
	assert.Equal(t, 2, len(totalMetric.Metric))

	// Check events_sampled
	sampledMetric := findMetric(metricFamilies, "test_events_sampled")
	require.NotNil(t, sampledMetric)
	assert.Equal(t, 1, len(sampledMetric.Metric))
	assert.Equal(t, float64(1), sampledMetric.Metric[0].Counter.GetValue())

	// Check events_dropped
	droppedMetric := findMetric(metricFamilies, "test_events_dropped")
	require.NotNil(t, droppedMetric)
	assert.Equal(t, 1, len(droppedMetric.Metric))
	assert.Equal(t, float64(1), droppedMetric.Metric[0].Counter.GetValue())
}

func TestIntegration_ConfigurationMigration(t *testing.T) {
	// Create old configuration
	oldConfig := &Config{
		Name:     "old-mimicry",
		LogLevel: "info",
		Traces: &TracesConfig{
			Enabled:                   true,
			AlwaysRecordRootRpcEvents: true, // Should be dropped
			Topics: map[string]TopicConfig{
				".*duplicate.*": {
					Topics: &TopicsConfig{
						GossipTopics: map[string]GossipTopicConfig{
							".*beacon_block.*": {
								TotalShards:  512,
								ActiveShards: generateShardRange(0, 511),
								ShardingKey:  "MsgID",
							},
							".*attestation.*": {
								TotalShards:  512,
								ActiveShards: generateShardRange(0, 255),
								ShardingKey:  "PeerID", // Should be dropped
							},
						},
					},
				},
			},
		},
	}

	// Migrate configuration
	newConfig := MigrateFromOldConfig(oldConfig)

	// Verify migration
	assert.Equal(t, "old-mimicry", newConfig.Name)
	assert.Equal(t, "info", newConfig.LogLevel)

	// Check sharding was migrated correctly
	assert.Len(t, newConfig.Sharding.Topics, 1) // Only MsgID config migrated

	beaconConfig, exists := newConfig.Sharding.Topics[".*beacon_block.*"]
	assert.True(t, exists)
	assert.Len(t, beaconConfig.ActiveShards, 512)

	// PeerID sharding should not be migrated
	_, exists = newConfig.Sharding.Topics[".*attestation.*"]
	assert.False(t, exists)

	// AlwaysRecordRootRpcEvents should be gone
	assert.True(t, newConfig.Sharding.NoShardingKeyEvents.Enabled)
}

// Helper function to find a metric by name
func findMetric(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	return nil
}
