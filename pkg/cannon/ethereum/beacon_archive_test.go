package ethereum

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	eth2client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/all"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
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

	mu        sync.Mutex
	fetched   []string
	envelopes []string
}

// fakeEnvelopeService answers envelope requests the way the beacon API does for a revealed payload.
type fakeEnvelopeService struct {
	eth2client.Service

	node *fakeBeaconNode
}

func (f *fakeEnvelopeService) SignedExecutionPayloadEnvelope(_ context.Context, opts *api.SignedExecutionPayloadEnvelopeOpts) (*api.Response[*spec.VersionedSignedExecutionPayloadEnvelope], error) {
	f.node.mu.Lock()
	defer f.node.mu.Unlock()

	f.node.envelopes = append(f.node.envelopes, opts.Block)

	return &api.Response[*spec.VersionedSignedExecutionPayloadEnvelope]{
		Data: &spec.VersionedSignedExecutionPayloadEnvelope{Version: spec.DataVersionGloas, Gloas: testEnvelope(999)},
	}, nil
}

func (f *fakeEnvelopeService) AgnosticSignedExecutionPayloadEnvelope(context.Context, *api.SignedExecutionPayloadEnvelopeOpts) (*api.Response[*all.SignedExecutionPayloadEnvelope], error) {
	return nil, errors.New("not implemented")
}

func (f *fakeBeaconNode) Service() eth2client.Service {
	return &fakeEnvelopeService{node: f}
}

func (f *fakeBeaconNode) FetchBlock(_ context.Context, id string) (*spec.VersionedSignedBeaconBlock, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.fetched = append(f.fetched, id)

	return testVersionedBlock(999), nil
}

type fakeBlockArchive struct {
	slots blockarchive.Slots
	err   error
	calls int
}

func (f *fakeBlockArchive) Slots(_ context.Context, slot phase0.Slot) (slots blockarchive.Slots, found bool, err error) {
	f.calls++

	if f.err != nil {
		return nil, false, f.err
	}

	if _, ok := f.slots[slot]; !ok {
		return nil, false, nil
	}

	return f.slots, true, nil
}

func blocksOnly(blocks map[phase0.Slot]*spec.VersionedSignedBeaconBlock) blockarchive.Slots {
	out := make(blockarchive.Slots, len(blocks))
	for slot, block := range blocks {
		out[slot] = blockarchive.Slot{Block: block}
	}

	return out
}

func testVersionedBlock(slot phase0.Slot) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionPhase0,
		Phase0:  &phase0.SignedBeaconBlock{Message: &phase0.BeaconBlock{Slot: slot}},
	}
}

func testGloasBlock(slot phase0.Slot) *spec.VersionedSignedBeaconBlock {
	return &spec.VersionedSignedBeaconBlock{
		Version: spec.DataVersionGloas,
		Gloas:   &gloas.SignedBeaconBlock{Message: &gloas.BeaconBlock{Slot: slot}},
	}
}

func testEnvelope(slot uint64) *gloas.SignedExecutionPayloadEnvelope {
	return &gloas.SignedExecutionPayloadEnvelope{Message: &gloas.ExecutionPayloadEnvelope{
		Payload: &gloas.ExecutionPayload{SlotNumber: slot},
	}}
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
		envelopeSfGroup: &singleflight.Group{},
		envelopeCache: ttlcache.New(
			ttlcache.WithTTL[string, *gloas.SignedExecutionPayloadEnvelope](time.Hour),
			ttlcache.WithCapacity[string, *gloas.SignedExecutionPayloadEnvelope](100),
		),
		metrics: testBeaconMetrics(),
	}, node
}

func TestGetBeaconBlockServesBundleAndCachesNeighbours(t *testing.T) {
	archive := &fakeBlockArchive{slots: blocksOnly(map[phase0.Slot]*spec.VersionedSignedBeaconBlock{
		100: testVersionedBlock(100),
		101: nil,
		102: testVersionedBlock(102),
	})}
	b, node := newArchiveTestNode(archive)

	block, err := b.GetBeaconBlock(context.Background(), "100")
	require.NoError(t, err)
	assert.Same(t, archive.slots[100].Block, block)

	missed, err := b.GetBeaconBlock(context.Background(), "101")
	require.NoError(t, err)
	assert.Nil(t, missed)

	block, err = b.GetBeaconBlock(context.Background(), "102")
	require.NoError(t, err)
	assert.Same(t, archive.slots[102].Block, block)

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
			archive:     &fakeBlockArchive{slots: blockarchive.Slots{100: {Block: testVersionedBlock(100)}}},
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
			archive:     &fakeBlockArchive{slots: blockarchive.Slots{100: {Block: testVersionedBlock(100)}}},
			identifier:  "0x0000000000000000000000000000000000000000000000000000000000000001",
			wantArchive: 0,
		},
		{
			name:        "named identifier",
			archive:     &fakeBlockArchive{slots: blockarchive.Slots{100: {Block: testVersionedBlock(100)}}},
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

func gloasArchive() *fakeBlockArchive {
	return &fakeBlockArchive{slots: blockarchive.Slots{
		100: {Block: testGloasBlock(100), Envelope: testEnvelope(100)},
		101: {Block: testGloasBlock(101)},
		102: {},
		103: {Block: testVersionedBlock(103)},
	}}
}

func TestGetExecutionPayloadEnvelopeServesBundle(t *testing.T) {
	archive := gloasArchive()
	b, node := newArchiveTestNode(archive)

	envelope, err := b.GetExecutionPayloadEnvelope(context.Background(), "100")
	require.NoError(t, err)
	assert.Same(t, archive.slots[100].Envelope, envelope)

	withheld, err := b.GetExecutionPayloadEnvelope(context.Background(), "101")
	require.NoError(t, err)
	assert.Nil(t, withheld, "a withheld payload has no envelope")

	block, err := b.GetBeaconBlock(context.Background(), "101")
	require.NoError(t, err)
	assert.Same(t, archive.slots[101].Block, block)

	assert.Equal(t, 1, archive.calls, "the range's blocks and envelopes come from one fetch")
	assert.Empty(t, node.envelopes)
	assert.Empty(t, node.fetched)
}

func TestGetBeaconBlockCachesEnvelopes(t *testing.T) {
	archive := gloasArchive()
	b, node := newArchiveTestNode(archive)

	_, err := b.GetBeaconBlock(context.Background(), "101")
	require.NoError(t, err)

	envelope, err := b.GetExecutionPayloadEnvelope(context.Background(), "100")
	require.NoError(t, err)
	assert.Same(t, archive.slots[100].Envelope, envelope)
	assert.Equal(t, 1, archive.calls)
	assert.Empty(t, node.envelopes)
}

func TestGetExecutionPayloadEnvelopeFallsBackToBeacon(t *testing.T) {
	tests := []struct {
		name        string
		archive     blockArchive
		identifier  string
		wantArchive int
	}{
		{name: "missed slot", archive: gloasArchive(), identifier: "102", wantArchive: 1},
		{name: "pre-gloas block", archive: gloasArchive(), identifier: "103", wantArchive: 1},
		{name: "slot not bundled", archive: gloasArchive(), identifier: "5000", wantArchive: 1},
		{name: "archive error", archive: &fakeBlockArchive{err: errors.New("archive down")}, identifier: "100", wantArchive: 1},
		{name: "root identifier", archive: gloasArchive(), identifier: "0x0000000000000000000000000000000000000000000000000000000000000001"},
		{name: "archive disabled", identifier: "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, node := newArchiveTestNode(tt.archive)

			envelope, err := b.GetExecutionPayloadEnvelope(context.Background(), tt.identifier)
			require.NoError(t, err)
			require.NotNil(t, envelope)
			assert.Equal(t, uint64(999), envelope.Message.Payload.SlotNumber)
			assert.Equal(t, []string{tt.identifier}, node.envelopes)

			if fake, ok := tt.archive.(*fakeBlockArchive); ok {
				assert.Equal(t, tt.wantArchive, fake.calls)
			}
		})
	}
}
