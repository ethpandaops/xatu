package iterator

import (
	"context"
	"testing"

	apiv1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// mockCeilingBeaconNode implements CeilingBeaconNode and records every
// identifier passed to GetBeaconBlock.
type mockCeilingBeaconNode struct {
	finality    *apiv1.Finality
	finalityErr error
	blocks      map[string]*spec.VersionedSignedBeaconBlock
	requested   []string
}

func (m *mockCeilingBeaconNode) Finality() (*apiv1.Finality, error) {
	return m.finality, m.finalityErr
}

func (m *mockCeilingBeaconNode) GetBeaconBlock(_ context.Context, identifier string, _ ...bool) (*spec.VersionedSignedBeaconBlock, error) {
	m.requested = append(m.requested, identifier)

	block, ok := m.blocks[identifier]
	if !ok {
		return nil, errors.Errorf("no block for identifier %s", identifier)
	}

	return block, nil
}

func bellatrixBlock(executionBlockNumber uint64) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionBellatrix,
		Bellatrix: &bellatrix.SignedBeaconBlock{
			Message: &bellatrix.BeaconBlock{
				Body: &bellatrix.BeaconBlockBody{
					ExecutionPayload: &bellatrix.ExecutionPayload{
						BlockNumber: executionBlockNumber,
					},
				},
			},
		},
	}
}

func finalityWithRoot(root phase0.Root) *apiv1.Finality {
	return &apiv1.Finality{
		Finalized: &phase0.Checkpoint{Epoch: 1, Root: root},
	}
}

func TestBackfillingBlock_FetchExecutionCeiling_FetchesByImmutableRoot(t *testing.T) {
	root := phase0.Root{0xab, 0xcd}

	mock := &mockCeilingBeaconNode{
		finality: finalityWithRoot(root),
		blocks: map[string]*spec.VersionedSignedBeaconBlock{
			xatuethv1.RootAsString(root): bellatrixBlock(12345),
		},
	}

	b := &BackfillingBlock{beaconNode: mock}

	ceiling, err := b.fetchExecutionCeiling(context.Background())
	require.NoError(t, err)

	assert.Equal(t, uint64(12345), ceiling)
	require.Len(t, mock.requested, 1)
	assert.Equal(t, xatuethv1.RootAsString(root), mock.requested[0],
		"the block must be fetched by its immutable root, never by a moving alias")
}

// Regression test: the ceiling must advance as finality advances. Fetching by
// the "finalized" alias froze it at the value cached at startup, because the
// beacon block cache keys by the raw identifier string.
func TestBackfillingBlock_FetchExecutionCeiling_AdvancesWithFinality(t *testing.T) {
	rootA := phase0.Root{0x0a}
	rootB := phase0.Root{0x0b}

	mock := &mockCeilingBeaconNode{
		finality: finalityWithRoot(rootA),
		blocks: map[string]*spec.VersionedSignedBeaconBlock{
			xatuethv1.RootAsString(rootA): bellatrixBlock(100),
			xatuethv1.RootAsString(rootB): bellatrixBlock(132),
		},
	}

	b := &BackfillingBlock{beaconNode: mock}

	ceiling, err := b.fetchExecutionCeiling(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(100), ceiling)

	// Finality advances to a new checkpoint.
	mock.finality = finalityWithRoot(rootB)

	ceiling, err = b.fetchExecutionCeiling(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(132), ceiling, "ceiling must follow the new finalized checkpoint")
}

func TestBackfillingBlock_FetchExecutionCeiling_ZeroRootFallsBackToGenesis(t *testing.T) {
	mock := &mockCeilingBeaconNode{
		finality: finalityWithRoot(phase0.Root{}),
		blocks: map[string]*spec.VersionedSignedBeaconBlock{
			"0": bellatrixBlock(0),
		},
	}

	b := &BackfillingBlock{beaconNode: mock}

	ceiling, err := b.fetchExecutionCeiling(context.Background())
	require.NoError(t, err)

	assert.Equal(t, uint64(0), ceiling)
	require.Len(t, mock.requested, 1)
	assert.Equal(t, "0", mock.requested[0], "a zero checkpoint root must resolve to the genesis slot")
}

func TestBackfillingBlock_FetchExecutionCeiling_Errors(t *testing.T) {
	tests := []struct {
		name string
		mock *mockCeilingBeaconNode
	}{
		{"finality error", &mockCeilingBeaconNode{finalityErr: errors.New("boom")}},
		{"nil finality", &mockCeilingBeaconNode{}},
		{"nil finalized checkpoint", &mockCeilingBeaconNode{finality: &apiv1.Finality{}}},
		{"block fetch error", &mockCeilingBeaconNode{finality: finalityWithRoot(phase0.Root{0x01})}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BackfillingBlock{beaconNode: tt.mock}

			_, err := b.fetchExecutionCeiling(context.Background())
			assert.Error(t, err)
		})
	}
}

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
