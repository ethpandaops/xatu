package execution

import (
	"context"
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
	// reportedTTL is how long a reported hash is remembered, so a transaction that keeps leaving and re-entering the
	// pool is reported once. Production reth pools re-admit a dropped queued transaction a median 8 and a 99th
	// percentile 46 minutes later.
	reportedTTL = 2 * time.Hour
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
	// gone holds the transactions missing from the latest snapshot but not the one before it. They are reported
	// only if the next snapshot is missing them too.
	gone map[string]*goneTx
	// reported holds the hashes reported within reportedTTL and when.
	reported map[string]time.Time
}

// goneTx is a transaction first missing from a snapshot taken at goneAt.
type goneTx struct {
	tx     *PoolTx
	goneAt time.Time
	// replacement is the pooled transaction with the same sender and nonce when it went missing.
	replacement *PoolTx
}

func newPoolTracker(client ClientProvider, log logrus.FieldLogger) *poolTracker {
	return &poolTracker{
		client:   client,
		log:      log,
		included: make(map[string]uint64),
		gone:     make(map[string]*goneTx),
		reported: make(map[string]time.Time),
	}
}

// update records a new snapshot and returns the transactions that have been missing from two snapshots in a row
// without being included, each at most once per reportedTTL. Fetch the snapshot before calling update, so every
// block that removed a transaction from it is already at or below the latest block update reads.
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
		clear(t.gone)

		return nil, nil
	}

	// On an error the previous snapshot stays, so the next poll still finds what left the pool since it.
	if fetchErr := t.fetchIncluded(ctx, t.lastBlock+1, latest); fetchErr != nil {
		return nil, fetchErr
	}

	t.txs = current

	if previous == nil {
		return nil, nil
	}

	if len(current) == 0 && len(previous) > 0 {
		// An empty pool right after a populated one is a restart or a wiped pool, not mass eviction.
		t.log.WithField("previous_txs", len(previous)).Warn("Execution client txpool emptied, skipping removed transactions")
		clear(t.gone)

		return nil, nil
	}

	for hash, at := range t.reported {
		if now.Sub(at) > reportedTTL {
			delete(t.reported, hash)
		}
	}

	bySenderNonce := make(map[senderNonce]*PoolTx, len(current))
	for _, tx := range current {
		bySenderNonce[senderNonce{tx.From, tx.Nonce}] = tx
	}

	// Transactions missing from the previous snapshot too are confirmed gone, unless they came back or were included.
	confirmed := make([]*goneTx, 0, len(t.gone))

	for hash, g := range t.gone {
		delete(t.gone, hash)

		if _, ok := current[hash]; ok {
			continue
		}

		if _, ok := t.included[hash]; ok {
			continue
		}

		confirmed = append(confirmed, g)
	}

	newlyGone := 0

	for hash, tx := range previous {
		if _, ok := current[hash]; ok {
			continue
		}

		if _, ok := t.included[hash]; ok {
			continue
		}

		if _, ok := t.reported[hash]; ok {
			continue
		}

		t.gone[hash] = &goneTx{tx: tx, goneAt: now, replacement: bySenderNonce[senderNonce{tx.From, tx.Nonce}]}
		newlyGone++
	}

	if newlyGone > len(previous)/2 && newlyGone > 100 {
		t.log.WithFields(logrus.Fields{
			"previous_txs": len(previous),
			"gone_txs":     newlyGone,
		}).Warn("Over half the txpool vanished unincluded, assuming a restart and skipping removed transactions")
		clear(t.gone)
	}

	return t.classify(ctx, confirmed, bySenderNonce)
}

// classify gives each confirmed gone transaction its reason and marks it reported. A transaction whose sender's
// nonce cannot be read is left out.
func (t *poolTracker) classify(ctx context.Context, confirmed []*goneTx, bySenderNonce map[senderNonce]*PoolTx) ([]*RemovedTx, error) {
	var (
		removed    []*RemovedTx
		needsNonce []*RemovedTx
	)

	for _, g := range confirmed {
		r := &RemovedTx{PoolTx: *g.tx, DetectedAt: g.goneAt}

		replacement := bySenderNonce[senderNonce{g.tx.From, g.tx.Nonce}]
		if replacement == nil {
			replacement = g.replacement
		}

		if replacement != nil {
			r.Reason = RemovalReasonReplaced
			r.Replacement = replacement
			removed = append(removed, r)

			continue
		}

		needsNonce = append(needsNonce, r)
	}

	defer func() {
		for _, r := range removed {
			t.reported[r.Hash] = r.DetectedAt
		}
	}()

	nonces, err := t.accountNonces(ctx, needsNonce)
	if err != nil {
		// The replacements are still certain, so report them and drop the rest.
		return removed, err
	}

	var consumed []*RemovedTx

	for _, r := range needsNonce {
		nonce, ok := nonces[r.From]
		if !ok {
			continue
		}

		r.AccountNonce = &nonce

		if nonce > r.Nonce {
			r.Reason = RemovalReasonNonceConsumed
			consumed = append(consumed, r)

			continue
		}

		r.Reason = RemovalReasonDropped
		removed = append(removed, r)
	}

	// A transaction whose nonce was consumed may still have been included in a block the included set missed, such
	// as one that replaced a reorged head, so ask the node.
	mined, err := t.minedTransactions(ctx, consumed)
	if err != nil {
		return removed, err
	}

	for _, r := range consumed {
		if !mined[r.Hash] {
			removed = append(removed, r)
		}
	}

	return removed, nil
}

// minedTransactions returns which of txs the node has in a block.
func (t *poolTracker) minedTransactions(ctx context.Context, txs []*RemovedTx) (map[string]bool, error) {
	mined := make(map[string]bool, len(txs))

	for start := 0; start < len(txs); start += removalRPCBatchSize {
		batch := txs[start:min(len(txs), start+removalRPCBatchSize)]
		results := make([]*struct {
			BlockNumber *hexutil.Uint64 `json:"blockNumber"`
		}, len(batch))
		reqs := make([]rpc.BatchElem, len(batch))

		for i, tx := range batch {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getTransactionByHash",
				Args:   []any{tx.Hash},
				Result: &results[i],
			}
		}

		if err := t.client.BatchCall(ctx, reqs); err != nil {
			return nil, fmt.Errorf("eth_getTransactionByHash batch: %w", err)
		}

		for i, req := range reqs {
			if req.Error != nil {
				return nil, fmt.Errorf("eth_getTransactionByHash %s: %w", batch[i].Hash, req.Error)
			}

			if results[i] != nil && results[i].BlockNumber != nil {
				mined[batch[i].Hash] = true
			}
		}
	}

	return mined, nil
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

		blocks := make([]*struct {
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

			if blocks[i] == nil {
				return fmt.Errorf("eth_getBlockByNumber %d: block not found", start+uint64(i))
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

func parsePoolTx(fields map[string]any) (*PoolTx, error) {
	hash, _ := fields["hash"].(string)
	from, _ := fields["from"].(string)

	if !common.IsHexAddress(from) || len(hash) != 66 {
		return nil, fmt.Errorf("transaction without hash or sender")
	}

	nonce, err := hexUint(fields, "nonce")
	if err != nil {
		return nil, err
	}

	gas, err := hexUint(fields, "gas")
	if err != nil {
		return nil, err
	}

	txType, err := hexUint(fields, "type")
	if err != nil {
		return nil, err
	}

	tx := &PoolTx{
		Hash:  strings.ToLower(hash),
		From:  common.HexToAddress(from),
		Nonce: nonce,
		Gas:   gas,
		Type:  uint8(txType), //nolint:gosec // transaction types fit in a byte
	}

	if to, ok := fields["to"].(string); ok && common.IsHexAddress(to) {
		addr := common.HexToAddress(to)
		tx.To = &addr
	}

	gasPrice := hexBig(fields, "gasPrice")

	// Legacy and access list transactions pay their gas price as both the tip and the fee cap.
	tx.GasTipCap = firstBig(hexBig(fields, "maxPriorityFeePerGas"), gasPrice)
	tx.GasFeeCap = firstBig(hexBig(fields, "maxFeePerGas"), gasPrice)
	tx.BlobGasFeeCap = hexBig(fields, "maxFeePerBlobGas")

	return tx, nil
}

// hexUint decodes a hex quantity field, treating a missing field as zero.
func hexUint(fields map[string]any, key string) (uint64, error) {
	raw, ok := fields[key].(string)
	if !ok {
		return 0, nil
	}

	v, err := hexutil.DecodeUint64(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}

	return v, nil
}

// hexBig decodes a hex quantity field, returning nil when it is missing or malformed.
func hexBig(fields map[string]any, key string) *big.Int {
	raw, ok := fields[key].(string)
	if !ok {
		return nil
	}

	v, err := hexutil.DecodeBig(raw)
	if err != nil {
		return nil
	}

	return v
}

func firstBig(values ...*big.Int) *big.Int {
	for _, v := range values {
		if v != nil {
			return v
		}
	}

	return new(big.Int)
}
