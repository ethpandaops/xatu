package blockprint

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlocksPerClientResponse_Structure(t *testing.T) {
	response := BlocksPerClientResponse{
		Uncertain:  100,
		Lighthouse: 1000,
		Lodestar:   200,
		Nimbus:     300,
		Other:      50,
		Prysm:      800,
		Teku:       400,
		Grandine:   150,
	}

	// Verify all fields are accessible
	assert.Equal(t, uint64(100), response.Uncertain)
	assert.Equal(t, uint64(1000), response.Lighthouse)
	assert.Equal(t, uint64(200), response.Lodestar)
	assert.Equal(t, uint64(300), response.Nimbus)
	assert.Equal(t, uint64(50), response.Other)
	assert.Equal(t, uint64(800), response.Prysm)
	assert.Equal(t, uint64(400), response.Teku)
	assert.Equal(t, uint64(150), response.Grandine)
}

func TestBlocksPerClientResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response BlocksPerClientResponse
		expected string
	}{
		{
			name: "all_fields_populated",
			response: BlocksPerClientResponse{
				Uncertain:  10,
				Lighthouse: 100,
				Lodestar:   20,
				Nimbus:     30,
				Other:      5,
				Prysm:      80,
				Teku:       40,
				Grandine:   15,
			},
			expected: `{"Uncertain":10,"Lighthouse":100,"Lodestar":20,"Nimbus":30,"Other":5,"Prysm":80,"Teku":40,"Grandine":15}`,
		},
		{
			name:     "zero_values",
			response: BlocksPerClientResponse{},
			expected: `{"Uncertain":0,"Lighthouse":0,"Lodestar":0,"Nimbus":0,"Other":0,"Prysm":0,"Teku":0,"Grandine":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Test unmarshaling
			var unmarshaled BlocksPerClientResponse
			err = json.Unmarshal([]byte(tt.expected), &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.response, unmarshaled)
		})
	}
}

func TestBlocksPerClientResponse_ZeroValue(t *testing.T) {
	var response BlocksPerClientResponse
	
	// Verify zero values
	assert.Equal(t, uint64(0), response.Uncertain)
	assert.Equal(t, uint64(0), response.Lighthouse)
	assert.Equal(t, uint64(0), response.Lodestar)
	assert.Equal(t, uint64(0), response.Nimbus)
	assert.Equal(t, uint64(0), response.Other)
	assert.Equal(t, uint64(0), response.Prysm)
	assert.Equal(t, uint64(0), response.Teku)
	assert.Equal(t, uint64(0), response.Grandine)
}

func TestSyncStatusResponse_Structure(t *testing.T) {
	response := SyncStatusResponse{
		GreatestBlockSlot: 12345678,
		Synced:           true,
	}

	assert.Equal(t, uint64(12345678), response.GreatestBlockSlot)
	assert.True(t, response.Synced)
}

func TestSyncStatusResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response SyncStatusResponse
		expected string
	}{
		{
			name: "synced_status",
			response: SyncStatusResponse{
				GreatestBlockSlot: 98765432,
				Synced:           true,
			},
			expected: `{"greatest_block_slot":98765432,"synced":true}`,
		},
		{
			name: "not_synced_status",
			response: SyncStatusResponse{
				GreatestBlockSlot: 12345,
				Synced:           false,
			},
			expected: `{"greatest_block_slot":12345,"synced":false}`,
		},
		{
			name:     "zero_values",
			response: SyncStatusResponse{},
			expected: `{"greatest_block_slot":0,"synced":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Test unmarshaling
			var unmarshaled SyncStatusResponse
			err = json.Unmarshal([]byte(tt.expected), &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.response, unmarshaled)
		})
	}
}

func TestProposersBlocksResponse_Structure(t *testing.T) {
	probMap := ProbabilityMap{
		ClientNamePrysm:      0.8,
		ClientNameLighthouse: 0.2,
	}

	response := ProposersBlocksResponse{
		ProposerIndex:   123456,
		Slot:           7890123,
		BestGuessSingle: ClientNamePrysm,
		BestGuessMulti:  "Prysm (80%), Lighthouse (20%)",
		ProbabilityMap:  &probMap,
	}

	assert.Equal(t, uint64(123456), response.ProposerIndex)
	assert.Equal(t, uint64(7890123), response.Slot)
	assert.Equal(t, ClientNamePrysm, response.BestGuessSingle)
	assert.Equal(t, "Prysm (80%), Lighthouse (20%)", response.BestGuessMulti)
	assert.NotNil(t, response.ProbabilityMap)
	assert.Equal(t, 0.8, (*response.ProbabilityMap)[ClientNamePrysm])
	assert.Equal(t, 0.2, (*response.ProbabilityMap)[ClientNameLighthouse])
}

func TestProposersBlocksResponse_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response ProposersBlocksResponse
		jsonStr  string
	}{
		{
			name: "complete_response",
			response: ProposersBlocksResponse{
				ProposerIndex:   555,
				Slot:           777,
				BestGuessSingle: ClientNameTeku,
				BestGuessMulti:  "Teku (90%), Other (10%)",
				ProbabilityMap: &ProbabilityMap{
					ClientNameTeku: 0.9,
					"Other":        0.1,
				},
			},
			jsonStr: `{
				"proposer_index": 555,
				"slot": 777,
				"best_guess_single": "Teku",
				"best_guess_multi": "Teku (90%), Other (10%)",
				"probability_map": {
					"Teku": 0.9,
					"Other": 0.1
				}
			}`,
		},
		{
			name: "minimal_response",
			response: ProposersBlocksResponse{
				ProposerIndex:   1,
				Slot:           2,
				BestGuessSingle: ClientNameUnknown,
				BestGuessMulti:  "",
				ProbabilityMap:  nil,
			},
			jsonStr: `{
				"proposer_index": 1,
				"slot": 2,
				"best_guess_single": "Unknown",
				"best_guess_multi": "",
				"probability_map": null
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)
			assert.JSONEq(t, tt.jsonStr, string(data))

			// Test unmarshaling
			var unmarshaled ProposersBlocksResponse
			err = json.Unmarshal([]byte(tt.jsonStr), &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.response.ProposerIndex, unmarshaled.ProposerIndex)
			assert.Equal(t, tt.response.Slot, unmarshaled.Slot)
			assert.Equal(t, tt.response.BestGuessSingle, unmarshaled.BestGuessSingle)
			assert.Equal(t, tt.response.BestGuessMulti, unmarshaled.BestGuessMulti)
			
			if tt.response.ProbabilityMap != nil {
				require.NotNil(t, unmarshaled.ProbabilityMap)
				assert.Equal(t, *tt.response.ProbabilityMap, *unmarshaled.ProbabilityMap)
			} else {
				assert.Nil(t, unmarshaled.ProbabilityMap)
			}
		})
	}
}

func TestProposersBlocksResponse_ZeroValue(t *testing.T) {
	var response ProposersBlocksResponse
	
	assert.Equal(t, uint64(0), response.ProposerIndex)
	assert.Equal(t, uint64(0), response.Slot)
	assert.Equal(t, ClientName(""), response.BestGuessSingle)
	assert.Equal(t, "", response.BestGuessMulti)
	assert.Nil(t, response.ProbabilityMap)
}

func TestClientName_Values(t *testing.T) {
	// Test defined client name constants
	assert.Equal(t, ClientName("Unknown"), ClientNameUnknown)
	assert.Equal(t, ClientName("Uncertain"), ClientNameUncertain)
	assert.Equal(t, ClientName("Prysm"), ClientNamePrysm)
	assert.Equal(t, ClientName("Lighthouse"), ClientNameLighthouse)
	assert.Equal(t, ClientName("Lodestar"), ClientNameLodestar)
	assert.Equal(t, ClientName("Nimbus"), ClientNameNimbus)
	assert.Equal(t, ClientName("Teku"), ClientNameTeku)
	assert.Equal(t, ClientName("Grandine"), ClientNameGrandine)
}

func TestClientName_StringConversion(t *testing.T) {
	name := ClientNamePrysm
	assert.Equal(t, "Prysm", string(name))
	
	// Test custom client name
	custom := ClientName("CustomClient")
	assert.Equal(t, "CustomClient", string(custom))
}

func TestProbabilityMap_Type(t *testing.T) {
	probMap := ProbabilityMap{
		ClientNamePrysm:      0.6,
		ClientNameLighthouse: 0.3,
		ClientNameTeku:       0.1,
	}

	// Verify it's a map with correct key-value types
	assert.Equal(t, 0.6, probMap[ClientNamePrysm])
	assert.Equal(t, 0.3, probMap[ClientNameLighthouse])
	assert.Equal(t, 0.1, probMap[ClientNameTeku])
	assert.Equal(t, 0.0, probMap[ClientNameNimbus]) // Non-existent key returns zero
}

func TestProbabilityMap_JSONSerialization(t *testing.T) {
	probMap := ProbabilityMap{
		ClientNamePrysm:  0.75,
		ClientNameNimbus: 0.25,
	}

	// Test marshaling
	data, err := json.Marshal(probMap)
	require.NoError(t, err)
	assert.JSONEq(t, `{"Prysm":0.75,"Nimbus":0.25}`, string(data))

	// Test unmarshaling
	var unmarshaled ProbabilityMap
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, probMap, unmarshaled)
}

func TestProbabilityMap_EmptyMap(t *testing.T) {
	var probMap ProbabilityMap
	
	// Empty map should handle non-existent keys gracefully
	assert.Equal(t, 0.0, probMap[ClientNamePrysm])
	assert.Equal(t, 0.0, probMap[ClientNameLighthouse])
	
	// JSON serialization of empty map
	data, err := json.Marshal(probMap)
	require.NoError(t, err)
	assert.Equal(t, "null", string(data))
}