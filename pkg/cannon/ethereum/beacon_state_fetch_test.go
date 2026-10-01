package ethereum

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	beaconapi "github.com/ethpandaops/beacon/pkg/beacon/api"
	client "github.com/ethpandaops/go-eth2-client"
	"github.com/ethpandaops/go-eth2-client/api"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/singleflight"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	dynssz "github.com/pk910/dynamic-ssz"
)

const testMetricText = "test"

var testMetricLabels = []string{"net", "instance"}

// fakeStateService answers the spec request the SSZ codec is built from.
type fakeStateService struct {
	client.Service
}

func (fakeStateService) Spec(_ context.Context, _ *api.SpecOpts) (*api.Response[map[string]any], error) {
	return &api.Response[map[string]any]{Data: map[string]any{}}, nil
}

// fakeStateNode serves a fixed SSZ state and records how it is used. Methods
// of beacon.Node that the state fetch does not call stay unimplemented.
type fakeStateNode struct {
	beacon.Node

	body    []byte
	version string
	openErr error
	delay   time.Duration

	opens       atomic.Int32
	inFlight    atomic.Int32
	maxInFlight atomic.Int32
}

func (f *fakeStateNode) Service() client.Service {
	return fakeStateService{}
}

func (f *fakeStateNode) OpenRawBeaconState(_ context.Context, _ string, contentType string) (*beaconapi.RawResponse, error) {
	f.opens.Add(1)

	if contentType != sszContentType {
		return nil, errors.New("state requested in a format other than ssz")
	}

	current := f.inFlight.Add(1)
	defer f.inFlight.Add(-1)

	for {
		maxSeen := f.maxInFlight.Load()
		if current <= maxSeen || f.maxInFlight.CompareAndSwap(maxSeen, current) {
			break
		}
	}

	time.Sleep(f.delay)

	if f.openErr != nil {
		return nil, f.openErr
	}

	return &beaconapi.RawResponse{
		Body:          io.NopCloser(bytes.NewReader(f.body)),
		ContentLength: int64(len(f.body)),
		Header:        http.Header{"Eth-Consensus-Version": []string{f.version}},
	}, nil
}

func newFetchTestBeaconNode(t *testing.T, node *fakeStateNode) *BeaconNode {
	t.Helper()

	metadata := services.NewMetadataService(logrus.New(), nil)

	return &BeaconNode{
		beacon:       node,
		services:     []services.Service{&metadata},
		stateSfGroup: &singleflight.Group{},
		stateCache: ttlcache.New(
			ttlcache.WithTTL[string, *GloasEpochState](gloasEpochStateCacheTTL),
			ttlcache.WithCapacity[string, *GloasEpochState](gloasEpochStateCacheSize),
		),
		stateSem: make(chan struct{}, 1),
		metrics: &Metrics{
			beacon: testMetricText,
			beaconStateCacheHit: prometheus.NewCounterVec(
				prometheus.CounterOpts{Name: "test_state_cache_hit_total", Help: testMetricText}, testMetricLabels),
			beaconStateFetchDuration: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{Name: "test_state_fetch_duration", Help: testMetricText}, testMetricLabels),
		},
	}
}

func newFakeStateNode(t *testing.T) *fakeStateNode {
	t.Helper()

	raw, err := dynssz.GetGlobalDynSsz().MarshalSSZ(newTestBeaconState(3200))
	require.NoError(t, err)

	return &fakeStateNode{body: raw, version: "gloas"}
}

func TestGetGloasEpochStateSharesOneDownload(t *testing.T) {
	node := newFakeStateNode(t)
	node.delay = 50 * time.Millisecond
	b := newFetchTestBeaconNode(t, node)

	const callers = 8

	states := make([]*GloasEpochState, callers)

	var wg sync.WaitGroup

	for i := range callers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			state, err := b.GetGloasEpochState(context.Background(), "3200")
			assert.NoError(t, err)

			states[i] = state
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), node.opens.Load(), "concurrent callers share a single download")

	for _, state := range states {
		assert.Same(t, states[0], state)
	}

	// A later caller is served from the cache.
	again, err := b.GetGloasEpochState(context.Background(), "3200")
	require.NoError(t, err)
	assert.Same(t, states[0], again)
	assert.Equal(t, int32(1), node.opens.Load())
}

func TestGetGloasEpochStateHoldsOneStateAtATime(t *testing.T) {
	node := newFakeStateNode(t)
	node.delay = 30 * time.Millisecond
	b := newFetchTestBeaconNode(t, node)

	var wg sync.WaitGroup

	for _, id := range []string{"3200", "3232", "3264", "3296"} {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := b.GetGloasEpochState(context.Background(), id)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(4), node.opens.Load())
	assert.Equal(t, int32(1), node.maxInFlight.Load(), "downloads of different states never overlap")
}

func TestGetGloasEpochStateRejectsOtherForks(t *testing.T) {
	node := newFakeStateNode(t)
	node.version = "fulu"
	b := newFetchTestBeaconNode(t, node)

	_, err := b.GetGloasEpochState(context.Background(), "3200")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consensus version")

	// A failure is not cached.
	node.version = "gloas"

	state, err := b.GetGloasEpochState(context.Background(), "3200")
	require.NoError(t, err)
	assert.Equal(t, uint64(3200), uint64(state.Slot))
}

func TestGetGloasEpochStateOpenError(t *testing.T) {
	node := newFakeStateNode(t)
	node.openErr = errors.New("boom")
	b := newFetchTestBeaconNode(t, node)

	_, err := b.GetGloasEpochState(context.Background(), "3200")
	require.ErrorContains(t, err, "boom")
}

func TestGetGloasEpochStateHonoursCancellationWhileQueued(t *testing.T) {
	node := newFakeStateNode(t)
	b := newFetchTestBeaconNode(t, node)

	// Occupy the only download slot.
	b.stateSem <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := b.GetGloasEpochState(ctx, "3200")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(0), node.opens.Load())
}
