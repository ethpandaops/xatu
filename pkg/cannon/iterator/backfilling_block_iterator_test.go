package iterator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestBackfillingBlock_LocationRoundTrip(t *testing.T) {
	b := &BackfillingBlock{
		cannonType: xatu.CannonType_EXECUTION_CANONICAL_BLOCK,
		networkID:  "1",
	}

	location, err := b.createLocation(100, 50)
	require.NoError(t, err)

	assert.Equal(t, xatu.CannonType_EXECUTION_CANONICAL_BLOCK, location.GetType())
	assert.Equal(t, "1", location.GetNetworkId())

	marker, err := b.GetMarker(location)
	require.NoError(t, err)

	assert.Equal(t, uint64(100), marker.GetFinalizedBlock())
	assert.Equal(t, int64(50), marker.GetBackfillBlock())
}

func TestBackfillingBlock_GetMarkerNilDefaultsToMinusOne(t *testing.T) {
	b := &BackfillingBlock{
		cannonType: xatu.CannonType_EXECUTION_CANONICAL_BLOCK,
	}

	// A location of the right type but with no marker data set.
	location := &xatu.CannonLocation{
		Type: xatu.CannonType_EXECUTION_CANONICAL_BLOCK,
	}

	marker, err := b.GetMarker(location)
	require.NoError(t, err)

	assert.Equal(t, int64(-1), marker.GetBackfillBlock(), "uninitialised backfill marker should default to -1")
}

func TestBackfillingBlock_MaxRange(t *testing.T) {
	b := &BackfillingBlock{config: &BackfillingBlockConfig{}}
	assert.Equal(t, uint64(1), b.maxRange(), "zero config max range should clamp to 1")

	b.config.MaxRangeSize = 50
	assert.Equal(t, uint64(50), b.maxRange())
}

func TestMinBackfillBlock(t *testing.T) {
	tests := []struct {
		name       string
		cannonType xatu.CannonType
		want       uint64
	}{
		{"balance_reads floors at 1 (genesis untraceable)", xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS, 1},
		{"storage_reads floors at 1 (genesis untraceable)", xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS, 1},
		{"nonce_reads floors at 1 (genesis untraceable)", xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS, 1},
		{"block floors at 0", xatu.CannonType_EXECUTION_CANONICAL_BLOCK, 0},
		{"balance_diffs floors at 0 (genesis premine is valid)", xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS, 0},
		{"transaction floors at 0", xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION, 0},
		{"traces floors at 0", xatu.CannonType_EXECUTION_CANONICAL_TRACES, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, minBackfillBlock(tt.cannonType))
		})
	}
}
