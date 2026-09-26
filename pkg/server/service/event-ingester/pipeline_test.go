package eventingester

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/ethpandaops/go-eth2-client/spec"

	"github.com/ethpandaops/xatu/pkg/output"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// testCache is an in-memory store.Cache used by the pipeline tests. It mirrors the
// GetOrSet and Delete semantics the dedup logic relies on, without touching the global
// prometheus registry.
type testCache struct {
	mu sync.Mutex
	m  map[string]string
}

func newTestCache() *testCache { return &testCache{m: make(map[string]string)} }

func (c *testCache) Start(context.Context) error { return nil }
func (c *testCache) Stop(context.Context) error  { return nil }
func (c *testCache) Type() string                { return "test" }

func (c *testCache) Get(_ context.Context, key string) (*string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.m[key]; ok {
		return &v, nil
	}

	return nil, nil
}

func (c *testCache) GetOrSet(_ context.Context, key, value string, _ time.Duration) (*string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.m[key]; ok {
		existing := v

		return &existing, true, nil
	}

	c.m[key] = value
	stored := value

	return &stored, false, nil
}

func (c *testCache) GetAndDelete(_ context.Context, key string) (*string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.m[key]; ok {
		delete(c.m, key)

		return &v, true, nil
	}

	return nil, false, nil
}

func (c *testCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value

	return nil
}

func (c *testCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)

	return nil
}

// mockSink is a minimal output.Sink used to observe writes and inject failures.
type mockSink struct {
	name     string
	fail     bool
	received int
}

func (m *mockSink) Start(context.Context) error { return nil }
func (m *mockSink) Stop(context.Context) error  { return nil }
func (m *mockSink) Type() string                { return "mock" }
func (m *mockSink) Name() string                { return m.name }

func (m *mockSink) HandleNewDecoratedEvent(context.Context, *xatu.DecoratedEvent) error {
	return nil
}

func (m *mockSink) HandleNewDecoratedEvents(_ context.Context, events []*xatu.DecoratedEvent) error {
	if m.fail {
		return errors.New("sink failure")
	}

	m.received += len(events)

	return nil
}

func ingesterTestContext() context.Context {
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1},
	})

	return metadata.NewIncomingContext(ctx, metadata.New(map[string]string{}))
}

func newBeaconBlockEvent(stateRoot string) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlock{
			EthV2BeaconBlock: &ethv2.EventBlock{
				Message: &ethv2.EventBlock_BellatrixBlock{
					BellatrixBlock: &ethv2.BeaconBlockBellatrix{StateRoot: stateRoot},
				},
			},
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				AdditionalData: &xatu.ClientMeta_EthV2BeaconBlock{
					EthV2BeaconBlock: &xatu.ClientMeta_AdditionalEthV2BeaconBlockData{
						Version: spec.DataVersionBellatrix.String(),
					},
				},
			},
		},
	}
}

func newTestPipeline(t *testing.T, sinks ...output.Sink) *Pipeline {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	return &Pipeline{
		log:     log,
		handler: NewHandler(log, nil, nil, newTestCache(), ""),
		sinks:   sinks,
	}
}

const testStateRoot = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

// TestProcessAndSendReleasesDedupOnSinkFailure verifies that when a sink write fails,
// the beacon-block dedup mark is released so the client's retry is not silently dropped
// as a duplicate of a write that never landed.
func TestProcessAndSendReleasesDedupOnSinkFailure(t *testing.T) {
	ctx := ingesterTestContext()

	failing := &mockSink{name: "sink", fail: true}
	p := newTestPipeline(t, failing)

	// First attempt: the sink fails. This marks the block during filtering and must
	// release that mark on failure.
	_, err := p.ProcessAndSend(ctx, []*xatu.DecoratedEvent{newBeaconBlockEvent(testStateRoot)}, nil, nil, "test")
	require.Error(t, err)

	// Retry with a healthy sink: the block must NOT be filtered out.
	recording := &mockSink{name: "sink"}
	p.sinks = []output.Sink{recording}

	count, err := p.ProcessAndSend(ctx, []*xatu.DecoratedEvent{newBeaconBlockEvent(testStateRoot)}, nil, nil, "test")
	require.NoError(t, err)
	require.Equal(t, uint64(1), count, "retry after a sink failure must not be dropped as a duplicate")
	require.Equal(t, 1, recording.received, "the block must reach the sink on retry")
}

// TestProcessAndSendDedupesAfterSuccessfulWrite confirms the dedup behaviour is retained:
// once a block reaches the sink, a subsequent submission of the same block is dropped.
func TestProcessAndSendDedupesAfterSuccessfulWrite(t *testing.T) {
	ctx := ingesterTestContext()

	recording := &mockSink{name: "sink"}
	p := newTestPipeline(t, recording)

	first, err := p.ProcessAndSend(ctx, []*xatu.DecoratedEvent{newBeaconBlockEvent(testStateRoot)}, nil, nil, "test")
	require.NoError(t, err)
	require.Equal(t, uint64(1), first)

	second, err := p.ProcessAndSend(ctx, []*xatu.DecoratedEvent{newBeaconBlockEvent(testStateRoot)}, nil, nil, "test")
	require.NoError(t, err)
	require.Equal(t, uint64(0), second, "a duplicate block after a successful write must be dropped")
	require.Equal(t, 1, recording.received, "the sink must receive the block exactly once")
}
