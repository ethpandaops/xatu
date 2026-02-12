package httpingester

import (
	"encoding/json"
	"testing"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestProtojsonUnmarshalExecutionBlockMetrics(t *testing.T) {
	// This is the EXACT JSON structure that Vector produces in CI
	// Field order: EXECUTION_BLOCK_METRICS, event, meta (alphabetical)
	jsonData := `{"events":[{"EXECUTION_BLOCK_METRICS":{"account_cache":{"hit_rate":81.0,"hits":"1000","misses":"234"},"block_hash":"0x0000000000000000000000000000000000000000000000000000000000000001","block_number":"1","code_cache":{"hit_bytes":"40000","hit_rate":89.9,"hits":"80","miss_bytes":"5000","misses":"9"},"commit_ms":5.4,"execution_ms":45.2,"gas_used":"15000000","mgas_per_sec":211.27,"source":"client-logs","state_hash_ms":8.3,"state_read_ms":12.1,"state_reads":{"accounts":"1234","code":"89","code_bytes":"45000","storage_slots":"5678"},"state_writes":{"accounts":"234","accounts_deleted":"5","code":"3","code_bytes":"15000","storage_slots":"890","storage_slots_deleted":"12"},"storage_cache":{"hit_rate":88.0,"hits":"5000","misses":"678"},"total_ms":71.0,"tx_count":150},"event":{"date_time":"2026-01-30T18:55:50.892527739Z","name":"EXECUTION_BLOCK_METRICS"},"meta":{"client":{"ethereum":{"network":{"id":"0","name":"local-testnet"}},"implementation":"vector","name":"sentry-logs-test","version":"dev"}}}]}`

	req := &xatu.CreateEventsRequest{}
	if err := protojson.Unmarshal([]byte(jsonData), req); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(req.GetEvents()) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(req.GetEvents()))
	}

	event := req.GetEvents()[0]

	// Check Event.Name is set correctly
	if event.GetEvent().GetName() != xatu.Event_EXECUTION_BLOCK_METRICS {
		t.Errorf("Expected event name EXECUTION_BLOCK_METRICS, got %v", event.GetEvent().GetName())
	}

	// Check that Data is populated
	if event.Data == nil {
		t.Error("event.Data is nil - oneof not populated")
	}

	// Check that it can be cast to the correct type
	_, ok := event.Data.(*xatu.DecoratedEvent_ExecutionBlockMetrics)
	if !ok {
		t.Errorf("Failed to cast event.Data to *DecoratedEvent_ExecutionBlockMetrics, actual type: %T", event.Data)
	}

	// Check the actual data
	metrics := event.GetExecutionBlockMetrics()

	if metrics == nil {
		t.Fatal("GetExecutionBlockMetrics() returned nil")
	}

	if metrics.GetSource() != "client-logs" {
		t.Errorf("Expected source 'client-logs', got '%s'", metrics.GetSource())
	}

	if metrics.GetBlockNumber().GetValue() != 1 {
		t.Errorf("Expected block_number 1, got %d", metrics.GetBlockNumber().GetValue())
	}
}

func TestProtojsonUnmarshalBatchedArray(t *testing.T) {
	// Vector may batch events as an array of CreateEventsRequest objects
	jsonData := `[{
		"events": [{
			"event": {
				"name": "EXECUTION_BLOCK_METRICS",
				"date_time": "2024-01-01T00:00:00Z"
			},
			"meta": {
				"client": {
					"name": "test-client"
				}
			},
			"EXECUTION_BLOCK_METRICS": {
				"source": "client-logs",
				"block_number": "1"
			}
		}]
	}, {
		"events": [{
			"event": {
				"name": "EXECUTION_BLOCK_METRICS",
				"date_time": "2024-01-01T00:00:01Z"
			},
			"meta": {
				"client": {
					"name": "test-client"
				}
			},
			"EXECUTION_BLOCK_METRICS": {
				"source": "client-logs",
				"block_number": "2"
			}
		}]
	}]`

	// Simulate the HTTP ingester's array handling logic
	// First, parse as JSON array
	var rawMessages []json.RawMessage
	if err := json.Unmarshal([]byte(jsonData), &rawMessages); err != nil {
		t.Fatalf("Failed to unmarshal outer array: %v", err)
	}

	allEvents := make([]*xatu.DecoratedEvent, 0, len(rawMessages))

	for i, raw := range rawMessages {
		req := &xatu.CreateEventsRequest{}
		if err := protojson.Unmarshal(raw, req); err != nil {
			t.Fatalf("Failed to unmarshal request %d: %v", i, err)
		}

		allEvents = append(allEvents, req.GetEvents()...)
	}

	if len(allEvents) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(allEvents))
	}

	// Verify both events have correct Data field populated
	for i, event := range allEvents {
		if event.Data == nil {
			t.Errorf("Event %d: Data is nil", i)

			continue
		}

		_, ok := event.Data.(*xatu.DecoratedEvent_ExecutionBlockMetrics)
		if !ok {
			t.Errorf("Event %d: Failed to cast Data, actual type: %T", i, event.Data)
		}

		metrics := event.GetExecutionBlockMetrics()
		if metrics == nil {
			t.Errorf("Event %d: GetExecutionBlockMetrics() is nil", i)
		}
	}
}
