package relaymonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	relethereum "github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/iterator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
)

// newBeaconNodeWithWallclock constructs a real *ethereum.BeaconNode via its
// real, exported, network-independent constructor and attaches a real,
// offline-computed *ethwallclock.EthereumBeaconChain to its unexported
// wallclock field via reflection. BeaconNode.Start() normally sets this
// field after fetching genesis/spec from a live Beacon API, which isn't
// available in a test; the wallclock math itself is pure and offline, so
// this produces exactly what a real Start() would against this
// genesis/slot-duration.
func newBeaconNodeWithWallclock(t *testing.T, clock *ethwallclock.EthereumBeaconChain) *relethereum.BeaconNode {
	t.Helper()

	node, err := relethereum.NewBeaconNode("test", observability.ContextualLogger(logrus.New()), &relethereum.Config{
		Network:       "mainnet",
		BeaconNodeURL: "http://unused.invalid",
	})
	if err != nil {
		t.Fatalf("NewBeaconNode: %v", err)
	}

	v := reflect.ValueOf(node).Elem().FieldByName("wallclock")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(clock)) //nolint:gosec // test-only field injection

	return node
}

// newFakeDenseRelay is a real HTTP server implementing the real relay
// data-API contract (/relay/v1/data/bidtraces/builder_blocks_received with
// cursor/limit/order=desc pagination, and the exact JSON field shape
// pkg/relaymonitor/relay/client.go's GetBids decodes), backed by a dense
// dataset: many builder submissions per slot, which is normal on mainnet.
func newFakeDenseRelay(t *testing.T, subsPerSlot int, lowSlot, highSlot uint64) *httptest.Server {
	t.Helper()

	type bidJSON struct {
		Slot                 string `json:"slot"`
		ParentHash           string `json:"parent_hash"`
		BlockHash            string `json:"block_hash"`
		BuilderPubkey        string `json:"builder_pubkey"`
		ProposerPubkey       string `json:"proposer_pubkey"`
		ProposerFeeRecipient string `json:"proposer_fee_recipient"`
		GasLimit             string `json:"gas_limit"`
		GasUsed              string `json:"gas_used"`
		Value                string `json:"value"`
		NumTx                string `json:"num_tx"`
		BlockNumber          string `json:"block_number"`
		Timestamp            string `json:"timestamp"`
		TimestampMs          string `json:"timestamp_ms"`
	}

	all := make([]bidJSON, 0, subsPerSlot*int(highSlot-lowSlot+1))

	for slot := highSlot; ; slot-- {
		for i := 0; i < subsPerSlot; i++ {
			all = append(all, bidJSON{
				Slot:                 strconv.FormatUint(slot, 10),
				ParentHash:           "0xparent",
				BlockHash:            fmt.Sprintf("0xhash-%d-%d", slot, i),
				BuilderPubkey:        "0xbuilder",
				ProposerPubkey:       "0xproposer",
				ProposerFeeRecipient: "0xfeerecipient",
				GasLimit:             "30000000",
				GasUsed:              "15000000",
				Value:                "1000000000000000000",
				NumTx:                "100",
				BlockNumber:          strconv.FormatUint(slot*1000, 10),
				Timestamp:            "1700000000",
				TimestampMs:          "1700000000000",
			})
		}

		if slot == lowSlot {
			break
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/relay/v1/data/bidtraces/builder_blocks_received", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		cursor, _ := strconv.ParseUint(q.Get("cursor"), 10, 64) //nolint:errcheck
		limit, _ := strconv.Atoi(q.Get("limit"))                //nolint:errcheck

		// Real relay semantics: slot <= cursor, descending, top `limit`.
		out := make([]bidJSON, 0, limit)

		for _, b := range all {
			slot, _ := strconv.ParseUint(b.Slot, 10, 64) //nolint:errcheck
			if slot > cursor {
				continue
			}

			out = append(out, b)

			if len(out) >= limit {
				break
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

// fakeBatchIterator implements consistency.go's BatchIterator interface,
// standing in only for the coordinator-backed persistence. The batch
// parameters it returns use the same cursor/limit/order formula as the real
// iterator.ForwardFillIterator.NextBatch, so it reproduces the real
// forward-fill request shape without a live gRPC coordinator.
type fakeBatchIterator struct {
	mu             sync.Mutex
	currentSlot    uint64
	targetSlot     uint64
	batchSize      int
	updateLocation []uint64
}

func (f *fakeBatchIterator) Next(_ context.Context) (*phase0.Slot, error) { return nil, nil }

func (f *fakeBatchIterator) UpdateLocation(_ context.Context, slot phase0.Slot) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.updateLocation = append(f.updateLocation, uint64(slot))

	return nil
}

func (f *fakeBatchIterator) NextBatch(_ context.Context) (*iterator.BatchRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Mirrors forward_fill_iterator.go's NextBatch:
	//   cursor = min(currentSlot + batchSize, targetSlot); limit = batchSize; order = desc
	cursor := f.currentSlot + uint64(f.batchSize)
	if cursor > f.targetSlot {
		cursor = f.targetSlot
	}

	return &iterator.BatchRequest{
		Params: map[string][]string{
			"cursor": {strconv.FormatUint(cursor, 10)},
			"limit":  {strconv.Itoa(f.batchSize)},
			"order":  {"desc"},
		},
		CurrentSlot: f.currentSlot,
		TargetSlot:  f.targetSlot,
		BatchSize:   f.batchSize,
	}, nil
}

// fakeSink records the slot number of every event actually delivered.
type fakeSink struct {
	mu    sync.Mutex
	slots map[uint64]bool
}

func (s *fakeSink) Start(context.Context) error { return nil }
func (s *fakeSink) Stop(context.Context) error  { return nil }
func (s *fakeSink) Type() string                { return "test" }
func (s *fakeSink) Name() string                { return "test" }

func (s *fakeSink) HandleNewDecoratedEvent(_ context.Context, event *xatu.DecoratedEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	slot := event.GetMeta().GetClient().GetMevRelayBidTraceBuilderBlockSubmission().GetSlot().GetNumber().GetValue()
	s.slots[slot] = true

	return nil
}

func (s *fakeSink) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	for _, e := range events {
		_ = s.HandleNewDecoratedEvent(ctx, e)
	}

	return nil
}

func newTestRelayMonitor(t *testing.T, namespace string, sink *fakeSink) (*RelayMonitor, observability.ContextualLogger) {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	clock := ethwallclock.NewEthereumBeaconChain(time.Now(), 12*time.Second, 32)

	r := &RelayMonitor{
		Config: &Config{
			Name:     "test",
			Ethereum: relethereum.Config{Network: "mainnet"},
			Consistency: &ConsistencyConfig{
				ForwardFill: &ForwardFillConfig{Enabled: true, TrailDistance: 4},
			},
		},
		id:       uuid.New(),
		log:      observability.ContextualLogger(log),
		metrics:  NewMetrics(namespace, "mainnet"),
		bidCache: NewDuplicateBidCache(13 * time.Minute),
		ethereum: newBeaconNodeWithWallclock(t, clock),
		sinks:    []output.Sink{sink},
	}

	return r, observability.ContextualLogger(log)
}

// When a dense window can be fully covered within the page budget, the
// fetch should page down through it rather than stopping at the first
// (limit-truncated) page, so every slot in the window is delivered before
// the cursor advances.
func TestForwardFill_PagesThroughDenseWindow_NoGap(t *testing.T) {
	const (
		lowSlot     = 105
		highSlot    = 110
		subsPerSlot = 30
		currentSlot = 104
		batchSize   = 50
	)

	srv := newFakeDenseRelay(t, subsPerSlot, lowSlot, highSlot)

	relayClient, err := relay.NewClient("forward_fill_pagination_test_nogap", relay.Config{
		URL:  srv.URL,
		Name: "test-relay",
	}, "mainnet")
	if err != nil {
		t.Fatalf("relay.NewClient: %v", err)
	}

	iter := &fakeBatchIterator{currentSlot: currentSlot, targetSlot: highSlot, batchSize: batchSize}
	sink := &fakeSink{slots: map[uint64]bool{}}
	r, log := newTestRelayMonitor(t, "forward_fill_pagination_test_nogap", sink)

	limiter := rate.NewLimiter(rate.Inf, 1)
	metrics := iterator.NewConsistencyMetrics("forward_fill_pagination_test_nogap")

	batch, err := iter.NextBatch(context.Background())
	if err != nil || batch == nil {
		t.Fatalf("NextBatch: batch=%v err=%v", batch, err)
	}

	r.processBatch(context.Background(), log, relayClient, limiter, metrics, batch,
		xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "forward_fill", iter, "mainnet")

	if len(iter.updateLocation) != 1 {
		t.Fatalf("expected exactly one UpdateLocation call, got %d: %v", len(iter.updateLocation), iter.updateLocation)
	}

	if got := iter.updateLocation[0]; got != highSlot {
		t.Fatalf("expected the cursor to advance to %d, got %d", uint64(highSlot), got)
	}

	sink.mu.Lock()
	defer sink.mu.Unlock()

	for slot := uint64(currentSlot + 1); slot <= highSlot; slot++ {
		if !sink.slots[slot] {
			t.Fatalf("expected slot %d to have been delivered before the cursor advanced past it, it was not", slot)
		}
	}
}

// When a window is too dense to cover within the page budget, the fetch
// should make no location update at all rather than advancing past slots
// that were never fetched. The next tick then retries the same window.
func TestForwardFill_StopsWithoutFalseProgress_WhenTooDenseForPageBudget(t *testing.T) {
	const (
		lowSlot     = 91
		highSlot    = 110
		subsPerSlot = 30
		currentSlot = 90
		batchSize   = 50
	)

	srv := newFakeDenseRelay(t, subsPerSlot, lowSlot, highSlot)

	relayClient, err := relay.NewClient("forward_fill_pagination_test_stall", relay.Config{
		URL:  srv.URL,
		Name: "test-relay",
	}, "mainnet")
	if err != nil {
		t.Fatalf("relay.NewClient: %v", err)
	}

	iter := &fakeBatchIterator{currentSlot: currentSlot, targetSlot: highSlot, batchSize: batchSize}
	sink := &fakeSink{slots: map[uint64]bool{}}
	r, log := newTestRelayMonitor(t, "forward_fill_pagination_test_stall", sink)

	limiter := rate.NewLimiter(rate.Inf, 1)
	metrics := iterator.NewConsistencyMetrics("forward_fill_pagination_test_stall")

	batch, err := iter.NextBatch(context.Background())
	if err != nil || batch == nil {
		t.Fatalf("NextBatch: batch=%v err=%v", batch, err)
	}

	r.processBatch(context.Background(), log, relayClient, limiter, metrics, batch,
		xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "forward_fill", iter, "mainnet")

	if len(iter.updateLocation) != 0 {
		t.Fatalf("expected no UpdateLocation call when the window can't be fully covered, got %d: %v",
			len(iter.updateLocation), iter.updateLocation)
	}
}
