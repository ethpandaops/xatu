package ethereum

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/blockarchive"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
)

var testBeaconMetrics = sync.OnceValue(func() *Metrics { return NewMetrics("xatu_cannon_archive_test", "test") })

type fakeBeaconNode struct {
	beacon.Node

	mu      sync.Mutex
	fetched []string
}

func (f *fakeBeaconNode) FetchBlock(_ context.Context, id string) (*spec.VersionedSignedBeaconBlock, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fetched = append(f.fetched, id)

	return testVersionedBlock(999), nil
}

type fakeBlockArchive struct {
	blocks blockarchive.Blocks
	err    error
	calls  int
}

func (f *fakeBlockArchive) Blocks(_ context.Context, slot phase0.Slot) (blocks blockarchive.Blocks, found bool, err error) {
	f.calls++

	if f.err != nil {
		return nil, false, f.err
	}

	if _, ok := f.blocks[slot]; !ok {
		return nil, false, nil
	}

	return f.blocks, true, nil
}

func testVersionedBlock(slot phase0.Slot) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionPhase0,
		Phase0:  &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{Slot: slot}},
	}
}

func newArchiveTestNode(archive blockArchive) (*BeaconNode, *fakeBeaconNode) {
	node := &fakeBeaconNode{}
	log := logrus.New()
	metadata := services.NewMetadataService(log, node)

	return &BeaconNode{
		config:   &Config{},
		log:      log,
		beacon:   node,
		archive:  archive,
		services: []services.Service{&metadata},
		sfGroup:  &singleflight.Group{},
		blockCache: ttlcache.New(
			ttlcache.WithTTL[string, *spec.VersionedSignedBeaconBlock](time.Hour),
			ttlcache.WithCapacity[string, *spec.VersionedSignedBeaconBlock](100),
		),
		blockPreloadSem: make(chan struct{}, 1),
		metrics:         testBeaconMetrics(),
	}, node
}

func TestGetBeaconBlockServesBundleAndCachesNeighbours(t *testing.T) {
	archive := &fakeBlockArchive{blocks: blockarchive.Blocks{
		100: testVersionedBlock(100),
		101: nil,
		102: testVersionedBlock(102),
	}}
	b, node := newArchiveTestNode(archive)

	block, err := b.GetBeaconBlock(context.Background(), "100")
	require.NoError(t, err)
	assert.Same(t, archive.blocks[100], block)

	missed, err := b.GetBeaconBlock(context.Background(), "101")
	require.NoError(t, err)
	assert.Nil(t, missed)

	block, err = b.GetBeaconBlock(context.Background(), "102")
	require.NoError(t, err)
	assert.Same(t, archive.blocks[102], block)

	assert.Equal(t, 1, archive.calls, "neighbouring slots come from the block cache")
	assert.Empty(t, node.fetched)
}

func TestGetBeaconBlockFallsBackToBeacon(t *testing.T) {
	tests := []struct {
		name        string
		archive     *fakeBlockArchive
		identifier  string
		wantArchive int
	}{
		{
			name:        "slot not bundled",
			archive:     &fakeBlockArchive{blocks: blockarchive.Blocks{100: testVersionedBlock(100)}},
			identifier:  "5000",
			wantArchive: 1,
		},
		{
			name:        "archive error",
			archive:     &fakeBlockArchive{err: errors.New("archive down")},
			identifier:  "100",
			wantArchive: 1,
		},
		{
			name:        "root identifier",
			archive:     &fakeBlockArchive{blocks: blockarchive.Blocks{100: testVersionedBlock(100)}},
			identifier:  "0x0000000000000000000000000000000000000000000000000000000000000001",
			wantArchive: 0,
		},
		{
			name:        "named identifier",
			archive:     &fakeBlockArchive{blocks: blockarchive.Blocks{100: testVersionedBlock(100)}},
			identifier:  "finalized",
			wantArchive: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, node := newArchiveTestNode(tt.archive)

			block, err := b.GetBeaconBlock(context.Background(), tt.identifier)
			require.NoError(t, err)
			require.NotNil(t, block)
			assert.Equal(t, phase0.Slot(999), block.Phase0.Message.Slot)
			assert.Equal(t, []string{tt.identifier}, node.fetched)
			assert.Equal(t, tt.wantArchive, tt.archive.calls)
		})
	}
}

func TestGetBeaconBlockWithoutArchive(t *testing.T) {
	b, node := newArchiveTestNode(nil)

	block, err := b.GetBeaconBlock(context.Background(), "100")
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, []string{"100"}, node.fetched)
}
