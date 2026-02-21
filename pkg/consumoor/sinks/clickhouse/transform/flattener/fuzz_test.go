package flattener_test

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// routeIndex maps table names to routes for O(1) lookup in fuzz functions.
var routeIndex = func() map[string]flattener.Route {
	routes := tabledefs.All()
	m := make(map[string]flattener.Route, len(routes))

	for _, r := range routes {
		m[r.TableName()] = r
	}

	return m
}()

// FuzzFlattenColumnAlignment feeds random proto bytes into every flattener and
// verifies safety invariants: no panics and column alignment.
func FuzzFlattenColumnAlignment(f *testing.F) {
	// Seed corpus: one valid serialised DecoratedEvent per route.
	for _, route := range tabledefs.All() {
		seed := &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     route.EventNames()[0],
				DateTime: timestamppb.Now(),
				Id:       "fuzz-seed",
			},
		}

		data, err := proto.Marshal(seed)
		if err != nil {
			f.Fatalf("marshal seed for %s: %v", route.TableName(), err)
		}

		f.Add(data, route.TableName())
	}

	f.Fuzz(func(t *testing.T, data []byte, tableName string) {
		route, ok := routeIndex[tableName]
		if !ok {
			t.Skip()
		}

		var event xatu.DecoratedEvent
		if err := proto.Unmarshal(data, &event); err != nil {
			t.Skip() // Invalid proto — not interesting.
		}

		batch := route.NewBatch()

		// Must not panic.
		_ = batch.FlattenTo(&event, nil)

		// Column alignment invariant: every column must have the same row count.
		expected := batch.Rows()
		for _, col := range batch.Input() {
			if col.Data.Rows() != expected {
				t.Fatalf("column %q has %d rows, expected %d in table %s",
					col.Name, col.Data.Rows(), expected, tableName)
			}
		}

		// Reset must not panic.
		batch.Reset()
		require.Equal(t, 0, batch.Rows(), "batch.Rows() should be 0 after Reset()")
	})
}

// FuzzFlattenNilSafety verifies that partially-constructed events never cause
// panics or column misalignment across all flatteners.
func FuzzFlattenNilSafety(f *testing.F) {
	f.Add(uint8(0), uint8(0))
	f.Add(uint8(0xFF), uint8(0xFF))

	routes := tabledefs.All()

	f.Fuzz(func(t *testing.T, eventBits, metaBits uint8) {
		for _, route := range routes {
			eventName := route.EventNames()[0]
			event := buildEventWithNilVariations(eventName, eventBits, metaBits)

			batch := route.NewBatch()

			// Must not panic.
			_ = batch.FlattenTo(event, nil)

			// Column alignment invariant.
			expected := batch.Rows()
			for _, col := range batch.Input() {
				require.Equalf(t, expected, col.Data.Rows(),
					"column %q misaligned in table %s", col.Name, route.TableName())
			}
		}
	})
}

// buildEventWithNilVariations constructs a DecoratedEvent with selectively nil
// fields driven by the bit patterns. This explores nil-safety of FlattenTo
// without requiring full proto deserialization.
func buildEventWithNilVariations(name xatu.Event_Name, eventBits, metaBits uint8) *xatu.DecoratedEvent {
	event := &xatu.DecoratedEvent{}

	// Bit 0: include Event wrapper.
	if eventBits&0x01 != 0 {
		event.Event = &xatu.Event{
			Name: name,
			Id:   "fuzz-nil",
		}

		// Bit 1: include DateTime.
		if eventBits&0x02 != 0 {
			event.Event.DateTime = timestamppb.Now()
		}
	}

	// Bit 2: include Meta.
	if eventBits&0x04 != 0 {
		event.Meta = &xatu.Meta{}

		// Bit 3: include Client in Meta.
		if eventBits&0x08 != 0 {
			event.Meta.Client = &xatu.ClientMeta{
				Name: "fuzz-client",
			}

			// Bit 4: include Ethereum in Client.
			if eventBits&0x10 != 0 {
				event.Meta.Client.Ethereum = &xatu.ClientMeta_Ethereum{}

				// Bit 5: include Network in Ethereum.
				if eventBits&0x20 != 0 {
					event.Meta.Client.Ethereum.Network = &xatu.ClientMeta_Ethereum_Network{
						Name: "mainnet",
						Id:   1,
					}
				}
			}
		}
	}

	// Bit 6 of metaBits: pre-extract metadata to test the metadata.Extract path.
	// (When metaBits bit 6 is unset, we pass nil metadata which triggers Extract inside FlattenTo.)

	return event
}
