package ethereum

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEthereumMetrics(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		beaconNodeName string
		expectedPrefix string
	}{
		{
			name:           "creates_metrics_with_cannon_namespace",
			namespace:      "xatu_cannon",
			beaconNodeName: "lighthouse",
			expectedPrefix: "xatu_cannon_ethereum",
		},
		{
			name:           "creates_metrics_with_custom_namespace",
			namespace:      "test",
			beaconNodeName: "prysm",
			expectedPrefix: "test_ethereum",
		},
		{
			name:           "creates_metrics_with_empty_namespace",
			namespace:      "",
			beaconNodeName: "nimbus",
			expectedPrefix: "_ethereum",
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

			metrics := NewMetrics(tt.namespace, tt.beaconNodeName)

			assert.NotNil(t, metrics)
			assert.Equal(t, tt.beaconNodeName, metrics.beacon)
			assert.NotNil(t, metrics.blocksFetched)
			assert.NotNil(t, metrics.blocksFetchErrors)
			assert.NotNil(t, metrics.blockCacheHit)
			assert.NotNil(t, metrics.blockCacheMiss)
			assert.NotNil(t, metrics.preloadBlockQueueSize)

			// Add sample values to make metrics appear in the registry
			metrics.IncBlocksFetched("test")
			metrics.IncBlocksFetchErrors("test")
			metrics.IncBlockCacheHit("test")
			metrics.IncBlockCacheMiss("test")
			metrics.SetPreloadBlockQueueSize("test", 1)

			// Verify metrics are registered
			metricFamilies, err := reg.Gather()
			require.NoError(t, err)
			
			expectedMetrics := []string{
				tt.expectedPrefix + "_blocks_fetched_total",
				tt.expectedPrefix + "_blocks_fetch_errors_total",
				tt.expectedPrefix + "_block_cache_hit_total",
				tt.expectedPrefix + "_block_cache_miss_total",
				tt.expectedPrefix + "_preload_block_queue_size",
			}
			
			for _, expectedMetric := range expectedMetrics {
				found := false
				for _, mf := range metricFamilies {
					if mf.GetName() == expectedMetric {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected metric %s not found in registry", expectedMetric)
			}
		})
	}
}

func TestMetrics_IncBlocksFetched(t *testing.T) {
	tests := []struct {
		name     string
		network  string
		beacon   string
		incCount int
	}{
		{
			name:     "increment_mainnet_blocks",
			network:  "mainnet",
			beacon:   "lighthouse",
			incCount: 5,
		},
		{
			name:     "increment_sepolia_blocks",
			network:  "sepolia",
			beacon:   "prysm",
			incCount: 10,
		},
		{
			name:     "single_increment",
			network:  "holesky",
			beacon:   "nimbus",
			incCount: 1,
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

			metrics := NewMetrics("test", tt.beacon)
			
			// Increment the metric
			for i := 0; i < tt.incCount; i++ {
				metrics.IncBlocksFetched(tt.network)
			}

			// Verify the metric value
			metricFamilies, err := reg.Gather()
			require.NoError(t, err)

			found := false
			for _, mf := range metricFamilies {
				if mf.GetName() == "test_ethereum_blocks_fetched_total" {
					for _, metric := range mf.GetMetric() {
						labels := make(map[string]string)
						for _, label := range metric.GetLabel() {
							labels[label.GetName()] = label.GetValue()
						}

						if labels["network"] == tt.network && labels["beacon"] == tt.beacon {
							found = true
							assert.Equal(t, float64(tt.incCount), metric.GetCounter().GetValue())
							break
						}
					}
				}
			}
			assert.True(t, found, "Expected metric with labels network=%s beacon=%s not found", tt.network, tt.beacon)
		})
	}
}

func TestMetrics_IncBlocksFetchErrors(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test", "beacon")
	
	// Increment errors for different networks
	metrics.IncBlocksFetchErrors("mainnet")
	metrics.IncBlocksFetchErrors("mainnet")
	metrics.IncBlocksFetchErrors("sepolia")

	// Verify the metric values
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	mainnetErrors := 0.0
	sepoliaErrors := 0.0

	for _, mf := range metricFamilies {
		if mf.GetName() == "test_ethereum_blocks_fetch_errors_total" {
			for _, metric := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, label := range metric.GetLabel() {
					labels[label.GetName()] = label.GetValue()
				}

				if labels["network"] == "mainnet" && labels["beacon"] == "beacon" {
					mainnetErrors = metric.GetCounter().GetValue()
				} else if labels["network"] == "sepolia" && labels["beacon"] == "beacon" {
					sepoliaErrors = metric.GetCounter().GetValue()
				}
			}
		}
	}

	assert.Equal(t, 2.0, mainnetErrors, "Expected 2 mainnet errors")
	assert.Equal(t, 1.0, sepoliaErrors, "Expected 1 sepolia error")
}

func TestMetrics_CacheMetrics(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test", "beacon")
	
	// Increment cache hits and misses
	metrics.IncBlockCacheHit("mainnet")
	metrics.IncBlockCacheHit("mainnet")
	metrics.IncBlockCacheHit("mainnet")
	metrics.IncBlockCacheMiss("mainnet")
	metrics.IncBlockCacheMiss("mainnet")

	// Verify the metric values
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	cacheHits := 0.0
	cacheMisses := 0.0

	for _, mf := range metricFamilies {
		if mf.GetName() == "test_ethereum_block_cache_hit_total" {
			for _, metric := range mf.GetMetric() {
				cacheHits = metric.GetCounter().GetValue()
			}
		} else if mf.GetName() == "test_ethereum_block_cache_miss_total" {
			for _, metric := range mf.GetMetric() {
				cacheMisses = metric.GetCounter().GetValue()
			}
		}
	}

	assert.Equal(t, 3.0, cacheHits, "Expected 3 cache hits")
	assert.Equal(t, 2.0, cacheMisses, "Expected 2 cache misses")
}

func TestMetrics_SetPreloadBlockQueueSize(t *testing.T) {
	tests := []struct {
		name     string
		network  string
		beacon   string
		queueSize int
	}{
		{
			name:      "set_mainnet_queue_size",
			network:   "mainnet",
			beacon:    "lighthouse",
			queueSize: 100,
		},
		{
			name:      "set_sepolia_queue_size",
			network:   "sepolia", 
			beacon:    "prysm",
			queueSize: 50,
		},
		{
			name:      "set_zero_queue_size",
			network:   "holesky",
			beacon:    "nimbus",
			queueSize: 0,
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

			metrics := NewMetrics("test", tt.beacon)
			
			// Set the queue size
			metrics.SetPreloadBlockQueueSize(tt.network, tt.queueSize)

			// Verify the metric value
			metricFamilies, err := reg.Gather()
			require.NoError(t, err)

			found := false
			for _, mf := range metricFamilies {
				if mf.GetName() == "test_ethereum_preload_block_queue_size" {
					for _, metric := range mf.GetMetric() {
						labels := make(map[string]string)
						for _, label := range metric.GetLabel() {
							labels[label.GetName()] = label.GetValue()
						}

						if labels["network"] == tt.network && labels["beacon"] == tt.beacon {
							found = true
							assert.Equal(t, float64(tt.queueSize), metric.GetGauge().GetValue())
							break
						}
					}
				}
			}
			assert.True(t, found, "Expected metric with labels network=%s beacon=%s not found", tt.network, tt.beacon)
		})
	}
}

func TestMetrics_MultipleOperations(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	metrics := NewMetrics("test", "multi-beacon")
	
	// Perform multiple operations
	metrics.IncBlocksFetched("mainnet")
	metrics.IncBlocksFetched("mainnet")
	metrics.IncBlocksFetchErrors("mainnet")
	metrics.IncBlockCacheHit("mainnet")
	metrics.IncBlockCacheMiss("mainnet")
	metrics.SetPreloadBlockQueueSize("mainnet", 25)

	// Verify all metrics were updated
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	metricsFound := make(map[string]float64)
	
	for _, mf := range metricFamilies {
		for _, metric := range mf.GetMetric() {
			metricName := mf.GetName()
			if metric.GetCounter() != nil {
				metricsFound[metricName] = metric.GetCounter().GetValue()
			} else if metric.GetGauge() != nil {
				metricsFound[metricName] = metric.GetGauge().GetValue()
			}
		}
	}

	assert.Equal(t, 2.0, metricsFound["test_ethereum_blocks_fetched_total"], "Blocks fetched should be 2")
	assert.Equal(t, 1.0, metricsFound["test_ethereum_blocks_fetch_errors_total"], "Fetch errors should be 1")
	assert.Equal(t, 1.0, metricsFound["test_ethereum_block_cache_hit_total"], "Cache hits should be 1")
	assert.Equal(t, 1.0, metricsFound["test_ethereum_block_cache_miss_total"], "Cache misses should be 1")
	assert.Equal(t, 25.0, metricsFound["test_ethereum_preload_block_queue_size"], "Queue size should be 25")
}

func TestMetrics_LabelValues(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()
	
	// Temporarily replace the default registry
	origRegistry := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegistry
	}()

	beaconName := "test-beacon-node"
	metrics := NewMetrics("test", beaconName)
	
	// Increment a metric
	metrics.IncBlocksFetched("custom-network")

	// Verify the labels are set correctly
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metricFamilies {
		if mf.GetName() == "test_ethereum_blocks_fetched_total" {
			for _, metric := range mf.GetMetric() {
				labels := make(map[string]string)
				for _, label := range metric.GetLabel() {
					labels[label.GetName()] = label.GetValue()
				}

				if labels["network"] == "custom-network" && labels["beacon"] == beaconName {
					found = true
					assert.Equal(t, 1.0, metric.GetCounter().GetValue())
					assert.Len(t, labels, 2, "Should have exactly 2 labels")
					break
				}
			}
		}
	}
	assert.True(t, found, "Expected metric with correct labels not found")
}