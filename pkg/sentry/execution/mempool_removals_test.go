package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chainStub answers the JSON-RPC calls the pool tracker makes from an in-memory chain.
type chainStub struct {
	ClientProvider

	latest uint64
	blocks map[uint64][]string
	nonces map[common.Address]uint64
	// mined holds transactions the node has in a block that the blocks map does not show, such as after a reorg.
	mined map[string]uint64
}

func (c *chainStub) CallContext(_ context.Context, result any, method string, _ ...any) error {
	if method != "eth_blockNumber" {
		return fmt.Errorf("unexpected method %s", method)
	}

	latest, ok := result.(*hexutil.Uint64)
	if !ok {
		return fmt.Errorf("unexpected result type %T", result)
	}

	*latest = hexutil.Uint64(c.latest)

	return nil
}

func (c *chainStub) BatchCall(_ context.Context, reqs []rpc.BatchElem) error {
	for i := range reqs {
		var value any

		switch reqs[i].Method {
		case "eth_getBlockByNumber":
			number, ok := reqs[i].Args[0].(hexutil.Uint64)
			if !ok {
				return fmt.Errorf("unexpected block number %v", reqs[i].Args[0])
			}

			txs, ok := c.blocks[uint64(number)]
			if !ok {
				value = nil

				break
			}

			value = map[string]any{"transactions": txs}
		case "eth_getTransactionCount":
			sender, ok := reqs[i].Args[0].(common.Address)
			if !ok {
				return fmt.Errorf("unexpected sender %v", reqs[i].Args[0])
			}

			value = hexutil.Uint64(c.nonces[sender])
		case "eth_getTransactionByHash":
			hash, ok := reqs[i].Args[0].(string)
			if !ok {
				return fmt.Errorf("unexpected hash %v", reqs[i].Args[0])
			}

			value = map[string]any{"hash": hash}

			if block, ok := c.mined[hash]; ok {
				value = map[string]any{"hash": hash, "blockNumber": hexutil.Uint64(block)}
			}
		default:
			return fmt.Errorf("unexpected method %s", reqs[i].Method)
		}

		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(raw, reqs[i].Result); err != nil {
			return err
		}
	}

	return nil
}

type poolEntry struct {
	section string
	hash    string
	from    string
	nonce   uint64
	tip     int64
	fee     int64
}

// txpoolContent builds a txpool_content response the way execution clients format it.
func txpoolContent(t *testing.T, entries ...poolEntry) map[string]any {
	t.Helper()

	content := map[string]map[string]map[string]any{"pending": {}, "queued": {}}

	for _, e := range entries {
		if content[e.section][e.from] == nil {
			content[e.section][e.from] = map[string]any{}
		}

		content[e.section][e.from][fmt.Sprint(e.nonce)] = map[string]any{
			"hash":                 e.hash,
			"from":                 e.from,
			"to":                   "0x00000000000000000000000000000000000000aa",
			"nonce":                hexutil.EncodeUint64(e.nonce),
			"type":                 "0x2",
			"gas":                  "0x5208",
			"maxPriorityFeePerGas": hexutil.EncodeUint64(uint64(e.tip)),
			"maxFeePerGas":         hexutil.EncodeUint64(uint64(e.fee)),
		}
	}

	raw, err := json.Marshal(content)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))

	return out
}

func hashOf(n int) string {
	return common.BigToHash(new(big.Int).Lsh(big.NewInt(1), uint(n))).Hex()
}

func TestPoolTrackerClassifiesRemovedTransactions(t *testing.T) {
	var (
		ctx    = context.Background()
		logger = logrus.New()
		x      = "0x1111111111111111111111111111111111111111"
		y      = "0x2222222222222222222222222222222222222222"
		z      = "0x3333333333333333333333333333333333333333"
		w      = "0x4444444444444444444444444444444444444444"
		t1     = time.Unix(1000, 0)
		t2     = time.Unix(1045, 0)
		t3     = time.Unix(1090, 0)
	)

	logger.SetOutput(io.Discard)

	chain := &chainStub{
		latest: 100,
		blocks: map[uint64][]string{98: {}, 99: {}},
		nonces: map[common.Address]uint64{
			common.HexToAddress(y): 8, // moved past C's nonce 7 without C
			common.HexToAddress(z): 3, // still waiting on D's nonce 3
			common.HexToAddress(w): 9,
		},
	}

	tracker := newPoolTracker(chain, logger)

	// Block 100 is imported while the first snapshot is taken, so F is already included when it is first seen gone.
	chain.blocks[100] = []string{hashOf(6)}

	removed, err := tracker.update(ctx, txpoolContent(t,
		poolEntry{"pending", hashOf(1), x, 5, 1, 10}, // A: included in block 101
		poolEntry{"pending", hashOf(2), x, 6, 1, 10}, // B: replaced by B2
		poolEntry{"pending", hashOf(3), y, 7, 1, 10}, // C: nonce consumed by an unseen transaction
		poolEntry{"queued", hashOf(4), z, 4, 1, 10},  // D: dropped from the queue
		poolEntry{"pending", hashOf(5), w, 9, 1, 10}, // E: stays, then dropped
		poolEntry{"pending", hashOf(6), w, 8, 1, 10}, // F: included in block 100
	), t1)
	require.NoError(t, err)
	assert.Empty(t, removed, "the first snapshot has nothing to diff against")

	chain.latest = 101
	chain.blocks[101] = []string{hashOf(1)}
	chain.nonces[common.HexToAddress(z)] = 4

	removed, err = tracker.update(ctx, txpoolContent(t,
		poolEntry{"pending", hashOf(7), x, 6, 3, 20}, // B2
		poolEntry{"pending", hashOf(5), w, 9, 1, 10}, // E
	), t2)
	require.NoError(t, err)

	sort.Slice(removed, func(i, j int) bool { return removed[i].Hash < removed[j].Hash })
	require.Len(t, removed, 3, "A and F were included, E is still pooled")

	b, c, d := removed[0], removed[1], removed[2]

	assert.Equal(t, hashOf(2), b.Hash)
	assert.Equal(t, RemovalReasonReplaced, b.Reason)
	require.NotNil(t, b.Replacement)
	assert.Equal(t, hashOf(7), b.Replacement.Hash)
	assert.Equal(t, int64(3), b.Replacement.GasTipCap.Int64())
	assert.Equal(t, int64(20), b.Replacement.GasFeeCap.Int64())
	assert.Nil(t, b.AccountNonce)

	assert.Equal(t, hashOf(3), c.Hash)
	assert.Equal(t, RemovalReasonNonceConsumed, c.Reason)
	require.NotNil(t, c.AccountNonce)
	assert.Equal(t, uint64(8), *c.AccountNonce)

	// D's nonce 4 equals the account nonce, so it was still executable: dropped, not consumed.
	assert.Equal(t, hashOf(4), d.Hash)
	assert.Equal(t, RemovalReasonDropped, d.Reason)
	assert.Equal(t, "queued", d.Pool)
	assert.Equal(t, t1, d.FirstSeen)
	assert.Equal(t, t1, d.LastSeen)
	assert.Equal(t, t2, d.DetectedAt)
	assert.Equal(t, common.HexToAddress(z), d.From)

	// E keeps the time it was first seen across snapshots.
	chain.latest = 102
	chain.blocks[102] = []string{}

	removed, err = tracker.update(ctx, txpoolContent(t, poolEntry{"pending", hashOf(7), x, 6, 3, 20}), t3)
	require.NoError(t, err)
	require.Len(t, removed, 1)
	assert.Equal(t, hashOf(5), removed[0].Hash)
	assert.Equal(t, RemovalReasonDropped, removed[0].Reason)
	assert.Equal(t, t1, removed[0].FirstSeen)
	assert.Equal(t, t2, removed[0].LastSeen)
}

func TestPoolTrackerSkipsWipedPool(t *testing.T) {
	var (
		ctx    = context.Background()
		logger = logrus.New()
		from   = "0x1111111111111111111111111111111111111111"
	)

	logger.SetOutput(io.Discard)

	chain := &chainStub{latest: 10, blocks: map[uint64][]string{8: {}, 9: {}, 10: {}, 11: {}, 12: {}}, nonces: map[common.Address]uint64{}}
	tracker := newPoolTracker(chain, logger)

	entries := make([]poolEntry, 0, 300)
	for i := range 300 {
		entries = append(entries, poolEntry{"pending", hashOf(i + 1), from, uint64(i), 1, 10})
	}

	_, err := tracker.update(ctx, txpoolContent(t, entries...), time.Unix(0, 0))
	require.NoError(t, err)

	// A restarted node comes back with a nearly empty pool: nothing it lost was evicted on the network.
	chain.latest = 11

	removed, err := tracker.update(ctx, txpoolContent(t, entries[0]), time.Unix(45, 0))
	require.NoError(t, err)
	assert.Empty(t, removed)

	_, err = tracker.update(ctx, txpoolContent(t, entries...), time.Unix(90, 0))
	require.NoError(t, err)

	chain.latest = 12

	removed, err = tracker.update(ctx, txpoolContent(t), time.Unix(135, 0))
	require.NoError(t, err)
	assert.Empty(t, removed)
}

func TestPoolTrackerChecksConsumedNoncesAgainstTheChain(t *testing.T) {
	var (
		ctx    = context.Background()
		logger = logrus.New()
		x      = "0x1111111111111111111111111111111111111111"
		y      = "0x2222222222222222222222222222222222222222"
	)

	logger.SetOutput(io.Discard)

	chain := &chainStub{
		latest: 10,
		blocks: map[uint64][]string{8: {}, 9: {}, 10: {}},
		nonces: map[common.Address]uint64{common.HexToAddress(x): 1, common.HexToAddress(y): 1},
		mined:  map[string]uint64{},
	}
	tracker := newPoolTracker(chain, logger)

	_, err := tracker.update(ctx, txpoolContent(t,
		poolEntry{"pending", hashOf(1), x, 0, 1, 10},
		poolEntry{"pending", hashOf(2), y, 0, 1, 10},
		poolEntry{"pending", hashOf(3), y, 5, 1, 10},
	), time.Unix(0, 0))
	require.NoError(t, err)

	// Block 11 as first read did not include A, then a reorg replaced it with a block that did. The tracker already
	// read block 11, so only the node can tell A was included.
	chain.latest = 11
	chain.blocks[11] = []string{}
	chain.mined[hashOf(1)] = 11

	stays := poolEntry{"pending", hashOf(3), y, 5, 1, 10}

	removed, err := tracker.update(ctx, txpoolContent(t, stays), time.Unix(45, 0))
	require.NoError(t, err)
	require.Len(t, removed, 1)
	assert.Equal(t, hashOf(2), removed[0].Hash)
	assert.Equal(t, RemovalReasonNonceConsumed, removed[0].Reason)

	// A block the node cannot return yet is an error, not an empty block.
	chain.latest = 13
	chain.blocks[13] = []string{}

	_, err = tracker.update(ctx, txpoolContent(t, stays), time.Unix(90, 0))
	require.Error(t, err)
}
