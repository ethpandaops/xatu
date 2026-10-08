package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
)

// A self-built Gloas block's bid carries the BUILDER_INDEX_SELF_BUILD sentinel
// (UINT64_MAX); it must be stored as NULL, while real indices (including 0)
// are kept.
func TestCanonicalBeaconBlockBuilderIndexSelfBuild(t *testing.T) {
	tests := []struct {
		name    string
		index   uint64
		want    uint64
		wantSet bool
	}{
		{name: "self_build", index: route.BuilderIndexSelfBuilt},
		{name: "builder_zero", index: 0, want: 0, wantSet: true},
		{name: "builder", index: 42, want: 42, wantSet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newcanonicalBeaconBlockBatch()
			b.appendEpbsFromBid(&ethv1.SignedExecutionPayloadBid{
				Message: &ethv1.ExecutionPayloadBid{BuilderIndex: wrapperspb.UInt64(tt.index)},
			})

			require.Equal(t, 1, b.BuilderIndex.Rows())

			got := b.BuilderIndex.Row(0)
			assert.Equal(t, tt.wantSet, got.Set)

			if tt.wantSet {
				assert.Equal(t, tt.want, got.Value)
			}
		})
	}
}
