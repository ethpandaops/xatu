package cannon

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
	}{
		{
			name:      "creates_metrics_with_xatu_cannon_namespace",
			namespace: "xatu_cannon",
		},
		{
			name:      "creates_metrics_with_custom_namespace",
			namespace: "test_namespace",
		},
		{
			name:      "creates_metrics_with_empty_namespace",
			namespace: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new registry for this test to avoid conflicts
			reg := prometheus.NewRegistry()
			
			// Temporarily replace the default registry
			origRegistry := prometheus.DefaultRegisterer
			prometheus.DefaultRegisterer = reg
			defer func() {
				prometheus.DefaultRegisterer = origRegistry
			}()

			metrics := NewMetrics(tt.namespace)

			assert.NotNil(t, metrics)
			assert.NotNil(t, metrics.decoratedEventTotal)

			// Add a test event so the metric appears in the registry
			testEvent := &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
					Id:   "test",
				},
			}
			metrics.AddDecoratedEvent(1, testEvent, "test")

			// Verify the metric is registered
			metricFamilies, err := reg.Gather()
			require.NoError(t, err)
			
			expectedName := tt.namespace + "_decorated_event_total"
			if tt.namespace == "" {
				expectedName = "decorated_event_total"
			}
			
			found := false
			for _, mf := range metricFamilies {
				if mf.GetName() == expectedName {
					found = true
					assert.Equal(t, "Total number of decorated events created by the cannon", mf.GetHelp())
					break
				}
			}
			assert.True(t, found, "Expected metric %s not found in registry", expectedName)
		})
	}
}

func TestMetrics_AddDecoratedEvent(t *testing.T) {
	tests := []struct {
		name           string
		count          int
		eventType      *xatu.DecoratedEvent
		network        string
		expectedLabels map[string]string
		expectedValue  float64
	}{
		{
			name:  "adds_single_beacon_block_event",
			count: 1,
			eventType: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
					DateTime: timestamppb.New(time.Now()),
					Id:       "test-id",
				},
			},
			network: "mainnet",
			expectedLabels: map[string]string{
				"type":    "BEACON_API_ETH_V2_BEACON_BLOCK_V2",
				"network": "mainnet",
			},
			expectedValue: 1.0,
		},
		{
			name:  "adds_multiple_attestation_events",
			count: 5,
			eventType: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
					DateTime: timestamppb.New(time.Now()),
					Id:       "test-id-2",
				},
			},
			network: "sepolia",
			expectedLabels: map[string]string{
				"type":    "BEACON_API_ETH_V1_EVENTS_ATTESTATION",
				"network": "sepolia",
			},
			expectedValue: 5.0,
		},
		{
			name:  "adds_zero_events",
			count: 0,
			eventType: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
					DateTime: timestamppb.New(time.Now()),
					Id:       "test-id-3",
				},
			},
			network: "holesky",
			expectedLabels: map[string]string{
				"type":    "BEACON_API_ETH_V1_EVENTS_BLOCK",
				"network": "holesky",
			},
			expectedValue: 0.0,
		},
		{
			name:  "adds_large_number_of_events",
			count: 1000,
			eventType: &xatu.DecoratedEvent{
				Event: &xatu.Event{
					Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
					DateTime: timestamppb.New(time.Now()),
					Id:       "test-id-4",
				},
			},
			network: "gnosis",
			expectedLabels: map[string]string{
				"type":    "BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT",
				"network": "gnosis",
			},
			expectedValue: 1000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new registry for this test to avoid conflicts
			reg := prometheus.NewRegistry()
			
			// Temporarily replace the default registry
			origRegistry := prometheus.DefaultRegisterer
			prometheus.DefaultRegisterer = reg
			defer func() {
				prometheus.DefaultRegisterer = origRegistry
			}()

			metrics := NewMetrics("test_cannon")
			
			// Add the decorated event
			metrics.AddDecoratedEvent(tt.count, tt.eventType, tt.network)

			// Gather metrics to verify the value was added
			metricFamilies, err := reg.Gather()
			require.NoError(t, err)

			found := false
			for _, mf := range metricFamilies {
				if mf.GetName() == "test_cannon_decorated_event_total" {
					for _, metric := range mf.GetMetric() {
						// Check if this metric has the expected labels
						labels := make(map[string]string)
						for _, label := range metric.GetLabel() {
							labels[label.GetName()] = label.GetValue()
						}

						if labels["type"] == tt.expectedLabels["type"] && 
						   labels["network"] == tt.expectedLabels["network"] {
							found = true
							assert.Equal(t, tt.expectedValue, metric.GetCounter().GetValue())
							break
						}
					}
				}
			}
			assert.True(t, found, "Expected metric with labels %v not found", tt.expectedLabels)
		})
	}
}

func TestMetrics_AddDecoratedEvent_Multiple(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test_cannon")

	// Create test events
	blockEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: timestamppb.New(time.Now()),
			Id:       "block-id",
		},
	}

	attestationEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			DateTime: timestamppb.New(time.Now()),
			Id:       "attestation-id",
		},
	}

	// Add multiple events
	metrics.AddDecoratedEvent(3, blockEvent, "mainnet")
	metrics.AddDecoratedEvent(7, blockEvent, "mainnet")      // Same type/network - should accumulate
	metrics.AddDecoratedEvent(2, attestationEvent, "mainnet") // Different type - separate counter
	metrics.AddDecoratedEvent(1, blockEvent, "sepolia")       // Same type, different network - separate counter

	// Gather and verify metrics
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	expectedValues := map[string]map[string]float64{
		"BEACON_API_ETH_V2_BEACON_BLOCK_V2": {
			"mainnet": 10.0, // 3 + 7
			"sepolia": 1.0,
		},
		"BEACON_API_ETH_V1_EVENTS_ATTESTATION": {
			"mainnet": 2.0,
		},
	}

	for _, mf := range metricFamilies {
		if mf.GetName() == "test_cannon_decorated_event_total" {
			for _, metric := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, label := range metric.GetLabel() {
					labels[label.GetName()] = label.GetValue()
				}

				eventType := labels["type"]
				network := labels["network"]
				actualValue := metric.GetCounter().GetValue()

				if expectedNetworks, exists := expectedValues[eventType]; exists {
					if expectedValue, exists := expectedNetworks[network]; exists {
						assert.Equal(t, expectedValue, actualValue, 
							"Unexpected value for type=%s network=%s", eventType, network)
					} else {
						t.Errorf("Unexpected network %s for event type %s", network, eventType)
					}
				} else {
					t.Errorf("Unexpected event type %s", eventType)
				}
			}
		}
	}
}

func TestMetrics_CounterVecLabels(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test_cannon")

	// Add a dummy event to ensure the metric shows up in the registry
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: timestamppb.New(time.Now()),
			Id:       "test-id",
		},
	}
	metrics.AddDecoratedEvent(1, event, "test")

	// Verify that the metric has the expected labels
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_cannon_decorated_event_total" {
			found = true
			assert.Equal(t, dto.MetricType_COUNTER, mf.GetType())
			assert.Equal(t, "Total number of decorated events created by the cannon", mf.GetHelp())
			
			// Should have one metric entry now
			assert.Len(t, mf.GetMetric(), 1)
			break
		}
	}
	assert.True(t, found, "Expected metric family not found")
}

func TestMetrics_ThreadSafety(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test_cannon")

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: timestamppb.New(time.Now()),
			Id:       "concurrent-test",
		},
	}

	// Test concurrent access (basic smoke test)
	done := make(chan bool, 2)
	
	go func() {
		for i := 0; i < 100; i++ {
			metrics.AddDecoratedEvent(1, event, "mainnet")
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			metrics.AddDecoratedEvent(1, event, "sepolia")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify that all increments were recorded
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	mainnetCount := 0.0
	sepoliaCount := 0.0

	for _, mf := range metricFamilies {
		if mf.GetName() == "test_cannon_decorated_event_total" {
			for _, metric := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, label := range metric.GetLabel() {
					labels[label.GetName()] = label.GetValue()
				}

				if labels["network"] == "mainnet" {
					mainnetCount = metric.GetCounter().GetValue()
				} else if labels["network"] == "sepolia" {
					sepoliaCount = metric.GetCounter().GetValue()
				}
			}
		}
	}

	assert.Equal(t, 100.0, mainnetCount, "Expected mainnet count to be 100")
	assert.Equal(t, 100.0, sepoliaCount, "Expected sepolia count to be 100")
}