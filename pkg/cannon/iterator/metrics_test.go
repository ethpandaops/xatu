package iterator

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackfillingCheckpointMetrics_Creation(t *testing.T) {
	// Use separate registry to avoid conflicts
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	namespace := "test_cannon"
	metrics := NewBackfillingCheckpointMetrics(namespace)

	// Verify metrics are created
	assert.NotNil(t, metrics.BackfillEpoch)
	assert.NotNil(t, metrics.FinalizedEpoch)
	assert.NotNil(t, metrics.FinalizedCheckpointEpoch)
	assert.NotNil(t, metrics.Lag)

	// Set sample values so metrics appear in registry
	metrics.SetBackfillEpoch("test", "test", "test", 1.0)
	metrics.SetFinalizedEpoch("test", "test", "test", 1.0)
	metrics.SetFinalizedCheckpointEpoch("test", 1.0)
	metrics.SetLag("test", "test", BackfillingCheckpointDirectionBackfill, 1.0)

	// Verify metrics are registered by gathering them
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	expectedMetrics := []string{
		"test_cannon_epoch_iterator_backfill_epoch",
		"test_cannon_epoch_iterator_finalized_epoch",
		"test_cannon_epoch_iterator_finalized_checkpoint_epoch",
		"test_cannon_epoch_iterator_lag_epochs",
	}
	
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[*mf.Name] = true
	}
	
	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "Expected metric %s to be registered", expected)
	}
}

func TestBackfillingCheckpointMetrics_SetBackfillEpoch(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBackfillingCheckpointMetrics("test")
	
	// Set a value
	metrics.SetBackfillEpoch("beacon_block", "mainnet", "finalized", 12345.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	// Find the backfill_epoch metric
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_epoch_iterator_backfill_epoch" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 12345.0, *mf.Metric[0].Gauge.Value)
			
			// Verify labels
			labels := mf.Metric[0].Label
			require.Len(t, labels, 3)
			assert.Equal(t, "cannon_type", *labels[0].Name)
			assert.Equal(t, "beacon_block", *labels[0].Value)
			assert.Equal(t, "checkpoint", *labels[1].Name)
			assert.Equal(t, "finalized", *labels[1].Value)
			assert.Equal(t, "network", *labels[2].Name)
			assert.Equal(t, "mainnet", *labels[2].Value)
			
			found = true
			break
		}
	}
	assert.True(t, found, "Backfill epoch metric not found")
}

func TestBackfillingCheckpointMetrics_SetFinalizedEpoch(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBackfillingCheckpointMetrics("test")
	
	// Set a value
	metrics.SetFinalizedEpoch("beacon_block", "mainnet", "finalized", 67890.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_epoch_iterator_finalized_epoch" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 67890.0, *mf.Metric[0].Gauge.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Finalized epoch metric not found")
}

func TestBackfillingCheckpointMetrics_SetFinalizedCheckpointEpoch(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBackfillingCheckpointMetrics("test")
	
	// Set a value
	metrics.SetFinalizedCheckpointEpoch("mainnet", 11111.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_epoch_iterator_finalized_checkpoint_epoch" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 11111.0, *mf.Metric[0].Gauge.Value)
			
			// Should only have network label
			labels := mf.Metric[0].Label
			require.Len(t, labels, 1)
			assert.Equal(t, "network", *labels[0].Name)
			assert.Equal(t, "mainnet", *labels[0].Value)
			
			found = true
			break
		}
	}
	assert.True(t, found, "Finalized checkpoint epoch metric not found")
}

func TestBackfillingCheckpointMetrics_SetLag(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBackfillingCheckpointMetrics("test")
	
	// Set a value
	metrics.SetLag("beacon_block", "mainnet", BackfillingCheckpointDirectionBackfill, 25.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_epoch_iterator_lag_epochs" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 25.0, *mf.Metric[0].Gauge.Value)
			
			// Verify direction label value
			labels := mf.Metric[0].Label
			var directionFound bool
			for _, label := range labels {
				if *label.Name == "direction" {
					assert.Equal(t, string(BackfillingCheckpointDirectionBackfill), *label.Value)
					directionFound = true
					break
				}
			}
			assert.True(t, directionFound, "Direction label not found")
			
			found = true
			break
		}
	}
	assert.True(t, found, "Lag metric not found")
}

func TestBlockprintMetrics_Creation(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	namespace := "test_blockprint"
	metrics := NewBlockprintMetrics(namespace)

	// Verify metrics are created
	assert.NotNil(t, metrics.Targetslot)
	assert.NotNil(t, metrics.Currentslot)

	// Set sample values so metrics appear in registry
	metrics.SetTargetSlot("test", "test", 1.0)
	metrics.SetCurrentSlot("test", "test", 1.0)

	// Verify metrics are registered
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	expectedMetrics := []string{
		"test_blockprint_slot_iterator_target_slot",
		"test_blockprint_slot_iterator_current_slot",
	}
	
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[*mf.Name] = true
	}
	
	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "Expected metric %s to be registered", expected)
	}
}

func TestBlockprintMetrics_SetTargetSlot(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBlockprintMetrics("test")
	
	// Set a value
	metrics.SetTargetSlot("blockprint", "mainnet", 999999.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_slot_iterator_target_slot" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 999999.0, *mf.Metric[0].Gauge.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Target slot metric not found")
}

func TestBlockprintMetrics_SetCurrentSlot(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewBlockprintMetrics("test")
	
	// Set a value
	metrics.SetCurrentSlot("blockprint", "mainnet", 888888.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_slot_iterator_current_slot" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 888888.0, *mf.Metric[0].Gauge.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Current slot metric not found")
}

func TestSlotMetrics_Creation(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	namespace := "test_slot"
	metrics := NewSlotMetrics(namespace)

	// Verify metrics are created
	assert.NotNil(t, metrics.TrailingSlots)
	assert.NotNil(t, metrics.CurrentSlot)

	// Set sample values so metrics appear in registry
	metrics.SetTrailingSlots("test", "test", 1.0)
	metrics.SetCurrentSlot("test", "test", 1.0)

	// Verify metrics are registered
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	expectedMetrics := []string{
		"test_slot_slot_iterator_trailing_slots",
		"test_slot_slot_iterator_current_slot",
	}
	
	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[*mf.Name] = true
	}
	
	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "Expected metric %s to be registered", expected)
	}
}

func TestSlotMetrics_SetTrailingSlots(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewSlotMetrics("test")
	
	// Set a value
	metrics.SetTrailingSlots("beacon_block", "mainnet", 42.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_slot_iterator_trailing_slots" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 42.0, *mf.Metric[0].Gauge.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Trailing slots metric not found")
}

func TestSlotMetrics_SetCurrentSlot(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewSlotMetrics("test")
	
	// Set a value
	metrics.SetCurrentSlot("beacon_block", "mainnet", 777777.0)
	
	// Verify the value was set
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "test_slot_iterator_current_slot" {
			require.Len(t, mf.Metric, 1)
			assert.Equal(t, 777777.0, *mf.Metric[0].Gauge.Value)
			found = true
			break
		}
	}
	assert.True(t, found, "Current slot metric not found")
}

func TestSlotMetrics_MultipleValues(t *testing.T) {
	reg := prometheus.NewRegistry()
	origRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = reg
	defer func() {
		prometheus.DefaultRegisterer = origRegisterer
	}()

	metrics := NewSlotMetrics("test")
	
	// Set multiple values with different labels
	metrics.SetCurrentSlot("beacon_block", "mainnet", 100.0)
	metrics.SetCurrentSlot("beacon_block", "sepolia", 200.0)
	metrics.SetCurrentSlot("blockprint", "mainnet", 300.0)
	
	// Verify all values are present
	metricFamilies, err := reg.Gather()
	require.NoError(t, err)
	
	var currentSlotMetrics int
	for _, mf := range metricFamilies {
		if *mf.Name == "test_slot_iterator_current_slot" {
			currentSlotMetrics = len(mf.Metric)
			assert.Equal(t, 3, len(mf.Metric), "Should have 3 metrics with different label combinations")
			break
		}
	}
	assert.Equal(t, 3, currentSlotMetrics, "Should find current slot metrics")
}