package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// Reasons a transaction left the pool without being included.
const (
	RemovalReasonReplaced      = "replaced"
	RemovalReasonNonceConsumed = "nonce_consumed"
	RemovalReasonDropped       = "dropped"
)

const (
	// includedBlockWindow is how many recent blocks' transactions are kept to recognise included transactions.
	includedBlockWindow = 64
	// maxBlocksPerPoll caps the blocks fetched per poll; a larger gap means the node was stalled or syncing, so the
	// diff is skipped rather than misclassifying the transactions those blocks included.
	maxBlocksPerPoll = includedBlockWindow
	// removalRPCBatchSize caps the calls in one JSON-RPC batch.
	removalRPCBatchSize = 100
)

// PoolTx is a transaction as it appeared in a txpool_content snapshot.
type PoolTx struct {
	Hash          string
	From          common.Address
	To            *common.Address
	Nonce         uint64
	Type          uint8
	Gas           uint64
	GasTipCap     *big.Int
	GasFeeCap     *big.Int
	BlobGasFeeCap *big.Int
	// Pool is the txpool_content section that held it: "pending" or "queued".
	Pool      string
	FirstSeen time.Time
	LastSeen  time.Time
}

// RemovedTx is a transaction that left the pool without being included in a block.
type RemovedTx struct {
	PoolTx
	Reason string
	// Replacement is the pooled transaction with the same sender and nonce, set for RemovalReasonReplaced.
	Replacement *PoolTx
	// AccountNonce is the sender's nonce at the latest block, set unless the reason is RemovalReasonReplaced.
	AccountNonce *uint64
	DetectedAt   time.Time
}

type senderNonce struct {
	from  common.Address
	nonce uint64
}

// poolTracker diffs successive txpool_content snapshots to find transactions that left the pool without being
// included. It is only used from the txpool_content fetch loop, so it needs no locking.
type poolTracker struct {
	client ClientProvider
	log    logrus.FieldLogger

	// txs is the previous snapshot by hash; nil until the first snapshot.
	txs map[string]*PoolTx
	// included maps the transaction hashes of recent blocks to their block number.
	included map[string]uint64
	// lastBlock is the highest block whose transactions are in included.
	lastBlock uint64
}

func newPoolTracker(client ClientProvider, log logrus.FieldLogger) *poolTracker {
	return &poolTracker{
		client:   client,
		log:      log,
		included: make(map[string]uint64),
	}
}

// update records a new snapshot and returns the transactions that left the pool since the previous one without
// being included. Fetch the snapshot before calling update, so every block that removed a transaction from it is
// already at or below the latest block update reads.
func (t *poolTracker) update(ctx context.Context, content map[string]any, now time.Time) ([]*RemovedTx, error) {
	previous := t.txs
	current := parsePoolSnapshot(content, previous, now)

	latest, err := t.latestBlock(ctx)
	if err != nil {
		return nil, err
	}

	if t.lastBlock == 0 {
		// Start a few blocks back so a block imported while the first snapshot was taken is still recognised.
		t.lastBlock = latest - min(latest, 2)
	}

	if latest > t.lastBlock+maxBlocksPerPoll {
		t.log.WithFields(logrus.Fields{
			"last_block":   t.lastBlock,
			"latest_block": latest,
		}).Warn("Execution client jumped too many blocks between txpool snapshots, skipping removed transactions")

		t.txs = current
		t.lastBlock = latest
		clear(t.included)

		return nil, nil
	}

	// On an error the previous snapshot stays, so the next poll still finds what left the pool since it.
	if err = t.fetchIncluded(ctx, t.lastBlock+1, latest); err != nil {
		return nil, err
	}

	t.txs = current

	if previous == nil {
		return nil, nil
	}

	if len(current) == 0 && len(previous) > 0 {
		// An empty pool right after a populated one is a restart or a wiped pool, not mass eviction.
		t.log.WithField("previous_txs", len(previous)).Warn("Execution client txpool emptied, skipping removed transactions")

		return nil, nil
	}

	bySenderNonce := make(map[senderNonce]*PoolTx, len(current))
	for _, tx := range current {
		bySenderNonce[senderNonce{tx.From, tx.Nonce}] = tx
	}

	var (
		removed    []*RemovedTx
		needsNonce []*RemovedTx
	)

	for hash, tx := range previous {
		if _, ok := current[hash]; ok {
			continue
		}

		if _, ok := t.included[hash]; ok {
			continue
		}

		r := &RemovedTx{PoolTx: *tx, DetectedAt: now}

		if replacement, ok := bySenderNonce[senderNonce{tx.From, tx.Nonce}]; ok {
			r.Reason = RemovalReasonReplaced
			r.Replacement = replacement
			removed = append(removed, r)

			continue
		}

		needsNonce = append(needsNonce, r)
	}

	if len(needsNonce) > len(previous)/2 && len(needsNonce) > 100 {
		t.log.WithFields(logrus.Fields{
			"previous_txs": len(previous),
			"gone_txs":     len(needsNonce),
		}).Warn("Over half the txpool vanished unincluded, assuming a restart and skipping removed transactions")

		return removed, nil
	}

	nonces, err := t.accountNonces(ctx, needsNonce)
	if err != nil {
		// The replacements are still certain, so report them and drop the rest.
		return removed, err
	}

	for _, r := range needsNonce {
		nonce, ok := nonces[r.From]
		if !ok {
			continue
		}

		r.AccountNonce = &nonce

		if nonce > r.Nonce {
			r.Reason = RemovalReasonNonceConsumed
		} else {
			r.Reason = RemovalReasonDropped
		}

		removed = append(removed, r)
	}

	return removed, nil
}

func (t *poolTracker) latestBlock(ctx context.Context) (uint64, error) {
	var latest hexutil.Uint64
	if err := t.client.CallContext(ctx, &latest, "eth_blockNumber"); err != nil {
		return 0, fmt.Errorf("eth_blockNumber: %w", err)
	}

	return uint64(latest), nil
}

// fetchIncluded adds the transaction hashes of blocks from..to to included and prunes blocks that left the window.
func (t *poolTracker) fetchIncluded(ctx context.Context, from, to uint64) error {
	for start := from; start <= to; start += removalRPCBatchSize {
		end := min(to, start+removalRPCBatchSize-1)

		blocks := make([]struct {
			Transactions []string `json:"transactions"`
		}, end-start+1)
		reqs := make([]rpc.BatchElem, len(blocks))

		for i := range blocks {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getBlockByNumber",
				Args:   []any{hexutil.Uint64(start + uint64(i)), false},
				Result: &blocks[i],
			}
		}

		if err := t.client.BatchCall(ctx, reqs); err != nil {
			return fmt.Errorf("eth_getBlockByNumber batch: %w", err)
		}

		for i, req := range reqs {
			if req.Error != nil {
				return fmt.Errorf("eth_getBlockByNumber %d: %w", start+uint64(i), req.Error)
			}

			for _, hash := range blocks[i].Transactions {
				t.included[strings.ToLower(hash)] = start + uint64(i)
			}
		}

		t.lastBlock = end
	}

	if t.lastBlock > includedBlockWindow {
		for hash, block := range t.included {
			if block < t.lastBlock-includedBlockWindow {
				delete(t.included, hash)
			}
		}
	}

	return nil
}

// accountNonces returns the latest-block nonce of each sender of txs.
func (t *poolTracker) accountNonces(ctx context.Context, txs []*RemovedTx) (map[common.Address]uint64, error) {
	senders := make([]common.Address, 0, len(txs))
	seen := make(map[common.Address]struct{}, len(txs))

	for _, tx := range txs {
		if _, ok := seen[tx.From]; ok {
			continue
		}

		seen[tx.From] = struct{}{}
		senders = append(senders, tx.From)
	}

	nonces := make(map[common.Address]uint64, len(senders))

	for start := 0; start < len(senders); start += removalRPCBatchSize {
		batch := senders[start:min(len(senders), start+removalRPCBatchSize)]
		results := make([]hexutil.Uint64, len(batch))
		reqs := make([]rpc.BatchElem, len(batch))

		for i, sender := range batch {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getTransactionCount",
				Args:   []any{sender, "latest"},
				Result: &results[i],
			}
		}

		if err := t.client.BatchCall(ctx, reqs); err != nil {
			return nonces, fmt.Errorf("eth_getTransactionCount batch: %w", err)
		}

		for i, req := range reqs {
			if req.Error == nil {
				nonces[batch[i]] = uint64(results[i])
			}
		}
	}

	return nonces, nil
}

// parsePoolSnapshot reads every transaction in a txpool_content response, keeping the first-seen time of those
// already in previous.
func parsePoolSnapshot(content map[string]any, previous map[string]*PoolTx, now time.Time) map[string]*PoolTx {
	snapshot := make(map[string]*PoolTx, len(previous))

	for _, section := range []string{"pending", "queued"} {
		accounts, ok := content[section].(map[string]any)
		if !ok {
			continue
		}

		for _, account := range accounts {
			txs, ok := account.(map[string]any)
			if !ok {
				continue
			}

			for _, raw := range txs {
				fields, ok := raw.(map[string]any)
				if !ok {
					continue
				}

				tx, err := parsePoolTx(fields)
				if err != nil {
					continue
				}

				tx.Pool = section
				tx.FirstSeen = now
				tx.LastSeen = now

				if prev, ok := previous[tx.Hash]; ok {
					tx.FirstSeen = prev.FirstSeen
				}

				snapshot[tx.Hash] = tx
			}
		}
	}

	return snapshot
}

// poolTxJSON holds the txpool_content fields a PoolTx needs.
type poolTxJSON struct {
	Hash                 common.Hash     `json:"hash"`
	From                 common.Address  `json:"from"`
	To                   *common.Address `json:"to"`
	Nonce                hexutil.Uint64  `json:"nonce"`
	Type                 *hexutil.Uint64 `json:"type"`
	Gas                  hexutil.Uint64  `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxFeePerBlobGas     *hexutil.Big    `json:"maxFeePerBlobGas"`
}

func parsePoolTx(fields map[string]any) (*PoolTx, error) {
	// Round-trip through JSON so hex quantities decode with go-ethereum's types.
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	var dec poolTxJSON
	if err := json.Unmarshal(raw, &dec); err != nil {
		return nil, err
	}

	if dec.Hash == (common.Hash{}) || dec.From == (common.Address{}) {
		return nil, fmt.Errorf("transaction without hash or sender")
	}

	tx := &PoolTx{
		Hash:  strings.ToLower(dec.Hash.Hex()),
		From:  dec.From,
		To:    dec.To,
		Nonce: uint64(dec.Nonce),
		Gas:   uint64(dec.Gas),
	}

	if dec.Type != nil {
		tx.Type = uint8(*dec.Type) //nolint:gosec // transaction types fit in a byte
	}

	// Legacy and access list transactions pay their gas price as both the tip and the fee cap.
	tx.GasTipCap = bigOr(dec.MaxPriorityFeePerGas, dec.GasPrice)
	tx.GasFeeCap = bigOr(dec.MaxFeePerGas, dec.GasPrice)

	if dec.MaxFeePerBlobGas != nil {
		tx.BlobGasFeeCap = dec.MaxFeePerBlobGas.ToInt()
	}

	return tx, nil
}

func bigOr(values ...*hexutil.Big) *big.Int {
	for _, v := range values {
		if v != nil {
			return v.ToInt()
		}
	}

	return new(big.Int)
}
