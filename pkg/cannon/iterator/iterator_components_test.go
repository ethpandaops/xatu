package iterator

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test BackFillingCheckpointNextResponse struct
func TestBackFillingCheckpointNextResponse_Structure(t *testing.T) {
	tests := []struct {
		name     string
		response *BackFillingCheckpointNextResponse
		validate func(*testing.T, *BackFillingCheckpointNextResponse)
	}{
		{
			name: "head_direction_response",
			response: &BackFillingCheckpointNextResponse{
				Next:       phase0.Epoch(100),
				LookAheads: []phase0.Epoch{100, 101, 102},
				Direction:  BackfillingCheckpointDirectionHead,
			},
			validate: func(t *testing.T, resp *BackFillingCheckpointNextResponse) {
				assert.Equal(t, phase0.Epoch(100), resp.Next)
				assert.Equal(t, []phase0.Epoch{100, 101, 102}, resp.LookAheads)
				assert.Equal(t, BackfillingCheckpointDirectionHead, resp.Direction)
			},
		},
		{
			name: "backfill_direction_response",
			response: &BackFillingCheckpointNextResponse{
				Next:       phase0.Epoch(50),
				LookAheads: []phase0.Epoch{50, 49, 48},
				Direction:  BackfillingCheckpointDirectionBackfill,
			},
			validate: func(t *testing.T, resp *BackFillingCheckpointNextResponse) {
				assert.Equal(t, phase0.Epoch(50), resp.Next)
				assert.Equal(t, []phase0.Epoch{50, 49, 48}, resp.LookAheads)
				assert.Equal(t, BackfillingCheckpointDirectionBackfill, resp.Direction)
			},
		},
		{
			name: "empty_lookaheads",
			response: &BackFillingCheckpointNextResponse{
				Next:       phase0.Epoch(25),
				LookAheads: []phase0.Epoch{},
				Direction:  BackfillingCheckpointDirectionHead,
			},
			validate: func(t *testing.T, resp *BackFillingCheckpointNextResponse) {
				assert.Equal(t, phase0.Epoch(25), resp.Next)
				assert.Empty(t, resp.LookAheads)
				assert.Equal(t, BackfillingCheckpointDirectionHead, resp.Direction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.response)
		})
	}
}

func TestBackfillingCheckpointDirection_Constants(t *testing.T) {
	tests := []struct {
		name      string
		direction BackfillingCheckpointDirection
		expected  string
	}{
		{
			name:      "backfill_direction",
			direction: BackfillingCheckpointDirectionBackfill,
			expected:  "backfill",
		},
		{
			name:      "head_direction",
			direction: BackfillingCheckpointDirectionHead,
			expected:  "head",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.direction))
		})
	}
}

// Test BlockprintIterator structure and basic operations
func TestBlockprintIterator_CreateLocation(t *testing.T) {
	tests := []struct {
		name         string
		cannonType   xatu.CannonType
		slot         phase0.Slot
		target       phase0.Slot
		expectError  bool
		validateFunc func(*testing.T, *xatu.CannonLocation)
	}{
		{
			name:        "valid_blockprint_classification",
			cannonType:  xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION,
			slot:        phase0.Slot(100),
			target:      phase0.Slot(200),
			expectError: false,
			validateFunc: func(t *testing.T, location *xatu.CannonLocation) {
				assert.Equal(t, xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION, location.Type)
				assert.Equal(t, "test_network", location.NetworkId)
				data := location.GetBlockprintBlockClassification()
				require.NotNil(t, data)
				assert.Equal(t, uint64(100), data.GetSlot())
				assert.Equal(t, uint64(200), data.GetTargetEndSlot())
			},
		},
		{
			name:        "zero_values",
			cannonType:  xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION,
			slot:        phase0.Slot(0),
			target:      phase0.Slot(0),
			expectError: false,
			validateFunc: func(t *testing.T, location *xatu.CannonLocation) {
				data := location.GetBlockprintBlockClassification()
				assert.Equal(t, uint64(0), data.GetSlot())
				assert.Equal(t, uint64(0), data.GetTargetEndSlot())
			},
		},
		{
			name:        "non_blockprint_cannon_type",
			cannonType:  xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
			slot:        phase0.Slot(100),
			target:      phase0.Slot(200),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iterator := &BlockprintIterator{
				cannonType: tt.cannonType,
				networkID:  "test_network",
			}

			location, err := iterator.createLocation(tt.slot, tt.target)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown cannon type")
			} else {
				assert.NoError(t, err)
				require.NotNil(t, location)
				assert.Equal(t, "test_network", location.NetworkId)
				assert.Equal(t, tt.cannonType, location.Type)
				if tt.validateFunc != nil {
					tt.validateFunc(t, location)
				}
			}
		})
	}
}

func TestBlockprintIterator_GetSlotsFromLocation(t *testing.T) {
	tests := []struct {
		name           string
		location       *xatu.CannonLocation
		expectedSlot   phase0.Slot
		expectedTarget phase0.Slot
		expectedError  string
	}{
		{
			name: "valid_blockprint_location",
			location: &xatu.CannonLocation{
				Type: xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION,
				Data: &xatu.CannonLocation_BlockprintBlockClassification{
					BlockprintBlockClassification: &xatu.CannonLocationBlockprintBlockClassification{
						Slot:          1500,
						TargetEndSlot: 2500,
					},
				},
			},
			expectedSlot:   phase0.Slot(1500),
			expectedTarget: phase0.Slot(2500),
			expectedError:  "",
		},
		{
			name: "zero_values",
			location: &xatu.CannonLocation{
				Type: xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION,
				Data: &xatu.CannonLocation_BlockprintBlockClassification{
					BlockprintBlockClassification: &xatu.CannonLocationBlockprintBlockClassification{
						Slot:          0,
						TargetEndSlot: 0,
					},
				},
			},
			expectedSlot:   phase0.Slot(0),
			expectedTarget: phase0.Slot(0),
			expectedError:  "",
		},
		{
			name: "non_blockprint_cannon_type",
			location: &xatu.CannonLocation{
				Type: xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
			},
			expectedSlot:   phase0.Slot(0),
			expectedTarget: phase0.Slot(0),
			expectedError:  "unknown cannon type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iterator := &BlockprintIterator{}

			slot, target, err := iterator.getSlotsFromLocation(tt.location)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Equal(t, phase0.Slot(0), slot)
				assert.Equal(t, phase0.Slot(0), target)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSlot, slot)
				assert.Equal(t, tt.expectedTarget, target)
			}
		})
	}
}

// Test basic BackfillingCheckpoint structure
func TestBackfillingCheckpoint_DirectionConstants(t *testing.T) {
	assert.Equal(t, "backfill", string(BackfillingCheckpointDirectionBackfill))
	assert.Equal(t, "head", string(BackfillingCheckpointDirectionHead))
}

func TestBackfillingCheckpoint_LookAheadCalculations(t *testing.T) {
	// Create a minimal BackfillingCheckpoint for testing lookahead methods
	checkpoint := &BackfillingCheckpoint{
		lookAheadDistance: 3,
	}

	t.Run("calculateBackfillingLookAheads", func(t *testing.T) {
		tests := []struct {
			name     string
			epoch    phase0.Epoch
			expected []phase0.Epoch
		}{
			{
				name:     "basic_lookaheads",
				epoch:    phase0.Epoch(100),
				expected: []phase0.Epoch{100, 99, 98},
			},
			{
				name:     "zero_epoch",
				epoch:    phase0.Epoch(0),
				expected: []phase0.Epoch{0, 18446744073709551615, 18446744073709551614}, // Underflow behavior
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkpoint.calculateBackfillingLookAheads(tt.epoch)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("calculateFinalizedLookAheads", func(t *testing.T) {
		tests := []struct {
			name           string
			epoch          phase0.Epoch
			finalizedEpoch phase0.Epoch
			expected       []phase0.Epoch
		}{
			{
				name:           "basic_lookaheads",
				epoch:          phase0.Epoch(100),
				finalizedEpoch: phase0.Epoch(105),
				expected:       []phase0.Epoch{100, 101, 102},
			},
			{
				name:           "limited_by_finalized",
				epoch:          phase0.Epoch(100),
				finalizedEpoch: phase0.Epoch(101),
				expected:       []phase0.Epoch{100},
			},
			{
				name:           "epoch_equals_finalized",
				epoch:          phase0.Epoch(100),
				finalizedEpoch: phase0.Epoch(100),
				expected:       []phase0.Epoch{},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := checkpoint.calculateFinalizedLookAheads(tt.epoch, tt.finalizedEpoch)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestIterator_StructureFields(t *testing.T) {
	t.Run("BlockprintIterator", func(t *testing.T) {
		iterator := &BlockprintIterator{
			cannonType:  xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION,
			networkID:   "test_network",
			networkName: "testnet",
		}

		assert.Equal(t, xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION, iterator.cannonType)
		assert.Equal(t, "test_network", iterator.networkID)
		assert.Equal(t, "testnet", iterator.networkName)
	})

	t.Run("BackfillingCheckpoint", func(t *testing.T) {
		checkpoint := &BackfillingCheckpoint{
			cannonType:        xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
			networkID:         "test_network",
			networkName:       "testnet",
			checkpointName:    "finalized",
			lookAheadDistance: 5,
		}

		assert.Equal(t, xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, checkpoint.cannonType)
		assert.Equal(t, "test_network", checkpoint.networkID)
		assert.Equal(t, "testnet", checkpoint.networkName)
		assert.Equal(t, "finalized", checkpoint.checkpointName)
		assert.Equal(t, 5, checkpoint.lookAheadDistance)
	})
}
