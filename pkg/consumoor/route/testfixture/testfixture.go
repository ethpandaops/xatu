// Package testfixture provides shared test helpers for per-table snapshot
// tests across the flattener domain packages.
package testfixture

import (
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// BaseMeta returns a reusable Meta with known deterministic values.
func BaseMeta() *xatu.Meta {
	return &xatu.Meta{
		Client: &xatu.ClientMeta{
			Name:           "test-client",
			Version:        "0.1.0",
			Id:             "client-id-1",
			Implementation: "xatu",
			Os:             "linux",
			Ethereum: &xatu.ClientMeta_Ethereum{
				Network: &xatu.ClientMeta_Ethereum_Network{
					Name: "mainnet",
					Id:   1,
				},
				Consensus: &xatu.ClientMeta_Ethereum_Consensus{
					Implementation: "lighthouse",
					Version:        "v4.5.6",
				},
				Execution: &xatu.ClientMeta_Ethereum_Execution{
					Implementation: "geth",
					Version:        "v1.13.0",
				},
			},
		},
	}
}

// TS returns a deterministic timestamp for tests.
func TS() *timestamppb.Timestamp {
	return timestamppb.New(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
}

// MetaWithAdditional returns BaseMeta with the given additional_data set.
// The provided ClientMeta's AdditionalData field is preserved while all
// standard fields are overwritten with deterministic test values.
func MetaWithAdditional(ad *xatu.ClientMeta) *xatu.Meta {
	m := BaseMeta()
	m.Client = ad
	m.Client.Name = "test-client"
	m.Client.Version = "0.1.0"
	m.Client.Id = "client-id-1"
	m.Client.Implementation = "xatu"
	m.Client.Os = "linux"
	m.Client.Ethereum = &xatu.ClientMeta_Ethereum{
		Network: &xatu.ClientMeta_Ethereum_Network{Name: "mainnet", Id: 1},
		Consensus: &xatu.ClientMeta_Ethereum_Consensus{
			Implementation: "lighthouse",
			Version:        "v4.5.6",
		},
		Execution: &xatu.ClientMeta_Ethereum_Execution{
			Implementation: "geth",
			Version:        "v1.13.0",
		},
	}

	return m
}

// SlotEpochAdditional returns common slot additional data for beacon tests.
func SlotEpochAdditional() *xatu.SlotV2 {
	return &xatu.SlotV2{
		Number:        wrapperspb.UInt64(100),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
	}
}

// EpochAdditional returns common epoch additional data for beacon tests.
func EpochAdditional() *xatu.EpochV2 {
	return &xatu.EpochV2{
		Number:        wrapperspb.UInt64(3),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// WallclockSlotAdditional returns common wallclock slot additional data.
func WallclockSlotAdditional() *xatu.SlotV2 {
	return &xatu.SlotV2{
		Number:        wrapperspb.UInt64(100),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
	}
}

// WallclockEpochAdditional returns common wallclock epoch additional data.
func WallclockEpochAdditional() *xatu.EpochV2 {
	return &xatu.EpochV2{
		Number:        wrapperspb.UInt64(3),
		StartDateTime: timestamppb.New(time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)),
	}
}

// PropagationAdditional returns common propagation additional data.
func PropagationAdditional() *xatu.PropagationV2 {
	return &xatu.PropagationV2{
		SlotStartDiff: wrapperspb.UInt64(500),
	}
}

// AssertSnapshot flattens the event into the batch and asserts row count,
// column values, and column alignment. FixedString null-byte padding is
// trimmed before string comparisons.
func AssertSnapshot(
	t *testing.T,
	batch route.ColumnarBatch,
	event *xatu.DecoratedEvent,
	expectedRows int,
	checks map[string]any,
) {
	t.Helper()

	err := batch.FlattenTo(event)
	require.NoError(t, err)

	require.Equal(t, expectedRows, batch.Rows(),
		"unexpected row count")

	snapper, ok := batch.(route.Snapshotter)
	require.True(t, ok, "batch must implement Snapshotter")

	snap := snapper.Snapshot()
	require.Len(t, snap, expectedRows)

	for col, want := range checks {
		got, exists := snap[0][col]
		if !assert.True(t, exists, "column %q not in snapshot", col) {
			continue
		}

		// Trim null-byte padding from FixedString columns.
		if gotStr, ok := got.(string); ok {
			if wantStr, ok := want.(string); ok {
				assert.Equal(t, wantStr, strings.TrimRight(gotStr, "\x00"),
					"column %q", col)

				continue
			}
		}

		assert.Equal(t, want, got, "column %q", col)
	}

	// Verify column alignment.
	for _, col := range batch.Input() {
		assert.Equalf(t, expectedRows, col.Data.Rows(),
			"column %q misaligned", col.Name)
	}
}
