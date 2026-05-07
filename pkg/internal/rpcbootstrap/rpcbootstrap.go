// Package rpcbootstrap provides an execution-layer JSON-RPC backed source for
// devp2p Status, headers, bodies, and receipts. It is shared between the
// mimicry and discovery modules so both can act as well-behaved peers during
// the eth handshake using a real EL node as the source of truth for chain head
// and historical data.
package rpcbootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/sirupsen/logrus"
)

const (
	maxBootstrapHeaders  = 192
	maxBootstrapBodies   = 64
	maxBootstrapReceipts = 64
	statusCacheTTL       = 10 * time.Second
)

// Bootstrap serves devp2p Status / headers / bodies / receipts to an eth peer
// using a backing EL JSON-RPC endpoint as the source of truth.
type Bootstrap struct {
	log logrus.FieldLogger

	client *ethclient.Client

	mu          sync.Mutex
	statusCache *statusSnapshot
}

type statusSnapshot struct {
	networkID               uint64
	genesis                 common.Hash
	terminalTotalDifficulty *big.Int
	headNumber              uint64
	headHash                common.Hash
	forkID                  forkid.ID
	forkFilter              forkid.Filter
	expiresAt               time.Time
}

type bootstrapChain struct {
	config  *params.ChainConfig
	genesis *types.Block
	head    *types.Header
}

func (c *bootstrapChain) Config() *params.ChainConfig {
	return c.config
}

func (c *bootstrapChain) Genesis() *types.Block {
	return c.genesis
}

func (c *bootstrapChain) CurrentHeader() *types.Header {
	return c.head
}

var sharedBootstraps = struct {
	sync.Mutex
	byURL map[string]*Bootstrap
}{
	byURL: map[string]*Bootstrap{},
}

// New returns a Bootstrap for the given EL JSON-RPC URL. Multiple callers with
// the same URL share a single underlying client and Status cache.
func New(ctx context.Context, log logrus.FieldLogger, url string) (*Bootstrap, error) {
	sharedBootstraps.Lock()

	if bootstrap, ok := sharedBootstraps.byURL[url]; ok {
		sharedBootstraps.Unlock()

		return bootstrap, nil
	}

	sharedBootstraps.Unlock()

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(rpcCtx, url)
	if err != nil {
		return nil, err
	}

	bootstrap := &Bootstrap{
		log:    log.WithField("bootstrap_rpc_url", url),
		client: client,
	}

	sharedBootstraps.Lock()

	if existing, ok := sharedBootstraps.byURL[url]; ok {
		sharedBootstraps.Unlock()
		client.Close()

		return existing, nil
	}

	sharedBootstraps.byURL[url] = bootstrap
	sharedBootstraps.Unlock()

	return bootstrap, nil
}

// Validate dials the URL and confirms a Status snapshot can be built.
func Validate(ctx context.Context, log logrus.FieldLogger, url string) error {
	bootstrap, err := New(ctx, log, url)
	if err != nil {
		return err
	}

	_, err = bootstrap.CurrentStatus(ctx)

	return err
}

// Status returns a Status message for the given protocolVersion sourced from
// the current chain head. It also validates the peer's claimed Status against
// our snapshot for early rejection of incompatible peers.
func (b *Bootstrap) Status(ctx context.Context, protocolVersion uint, peerStatus mimicry.Status) (mimicry.Status, error) {
	snapshot, err := b.CurrentStatus(ctx)
	if err != nil {
		return nil, err
	}

	if err := validatePeerStatus(peerStatus, snapshot); err != nil {
		return nil, err
	}

	switch protocolVersion {
	case 68:
		return &mimicry.Status68{
			Status68Packet: mimicry.Status68Packet{
				ProtocolVersion: uint32(protocolVersion),
				NetworkID:       snapshot.networkID,
				TD:              new(big.Int).Set(snapshot.terminalTotalDifficulty),
				Head:            snapshot.headHash,
				Genesis:         snapshot.genesis,
				ForkID:          snapshot.forkID,
			},
		}, nil
	case 69, 70:
		return &mimicry.Status69{
			StatusPacket: eth.StatusPacket{
				ProtocolVersion: uint32(protocolVersion),
				NetworkID:       snapshot.networkID,
				Genesis:         snapshot.genesis,
				ForkID:          snapshot.forkID,
				EarliestBlock:   0,
				LatestBlock:     snapshot.headNumber,
				LatestBlockHash: snapshot.headHash,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported eth protocol version: %d", protocolVersion)
	}
}

// CurrentStatus returns the cached chain-head snapshot, refreshing it when
// stale.
func (b *Bootstrap) CurrentStatus(ctx context.Context) (*statusSnapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.statusCache != nil && time.Now().Before(b.statusCache.expiresAt) {
		return b.statusCache, nil
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	networkID, err := b.client.NetworkID(rpcCtx)
	if err != nil {
		return nil, fmt.Errorf("fetch network id: %w", err)
	}

	chainConfig, genesis, terminalTotalDifficulty, err := bootstrapNetwork(networkID.Uint64())
	if err != nil {
		return nil, err
	}

	head, err := b.client.HeaderByNumber(rpcCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch latest header: %w", err)
	}

	forkID := forkid.NewID(chainConfig, genesis, head.Number.Uint64(), head.Time)
	forkFilter := forkid.NewFilter(&bootstrapChain{
		config:  chainConfig,
		genesis: genesis,
		head:    head,
	})

	b.statusCache = &statusSnapshot{
		networkID:               networkID.Uint64(),
		genesis:                 genesis.Hash(),
		terminalTotalDifficulty: new(big.Int).Set(terminalTotalDifficulty),
		headNumber:              head.Number.Uint64(),
		headHash:                head.Hash(),
		forkID:                  forkID,
		forkFilter:              forkFilter,
		expiresAt:               time.Now().Add(statusCacheTTL),
	}

	return b.statusCache, nil
}

// Headers serves a GetBlockHeaders response sourced from the backing RPC.
func (b *Bootstrap) Headers(ctx context.Context, request *eth.GetBlockHeadersRequest) ([]*types.Header, error) {
	if request == nil || request.Amount == 0 {
		return nil, nil
	}

	amount := request.Amount
	if amount > maxBootstrapHeaders {
		amount = maxBootstrapHeaders
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	header, err := b.originHeader(rpcCtx, request.Origin)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return nil, nil
		}

		return nil, err
	}

	if header == nil {
		return nil, nil
	}

	headers := make([]*types.Header, 0, amount)
	headers = append(headers, header)

	if request.Skip == math.MaxUint64 {
		return headers, nil
	}

	step := request.Skip + 1

	for uint64(len(headers)) < amount {
		current := headers[len(headers)-1].Number.Uint64()

		var next uint64

		if request.Reverse {
			if current < step {
				break
			}

			next = current - step
		} else {
			if current > math.MaxUint64-step {
				break
			}

			next = current + step
		}

		nextHeader, herr := b.client.HeaderByNumber(rpcCtx, new(big.Int).SetUint64(next))
		if herr != nil {
			if errors.Is(herr, ethereum.NotFound) {
				break
			}

			return nil, herr
		}

		if nextHeader == nil {
			break
		}

		headers = append(headers, nextHeader)
	}

	return headers, nil
}

// Bodies serves a GetBlockBodies response sourced from the backing RPC.
func (b *Bootstrap) Bodies(ctx context.Context, hashes []common.Hash) ([]eth.BlockBody, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	if len(hashes) > maxBootstrapBodies {
		hashes = hashes[:maxBootstrapBodies]
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	bodies := make([]eth.BlockBody, 0, len(hashes))
	for _, hash := range hashes {
		block, err := b.client.BlockByHash(rpcCtx, hash)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}

			return nil, err
		}

		if block == nil {
			continue
		}

		body, err := encodeBlockBody(block)
		if err != nil {
			return bodies, err
		}

		bodies = append(bodies, body)
	}

	return bodies, nil
}

// Receipts serves a GetReceipts response sourced from the backing RPC.
func (b *Bootstrap) Receipts(ctx context.Context, request mimicry.ReceiptRequest) ([]*eth.ReceiptList, error) {
	hashes := request.Hashes
	if len(hashes) == 0 {
		return nil, nil
	}

	if len(hashes) > maxBootstrapReceipts {
		hashes = hashes[:maxBootstrapReceipts]
	}

	rpcCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	receipts := make([]*eth.ReceiptList, 0, len(hashes))
	for index, hash := range hashes {
		blockReceipts, err := b.client.BlockReceipts(rpcCtx, rpc.BlockNumberOrHashWithHash(hash, false))
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}

			return nil, err
		}

		if blockReceipts == nil {
			continue
		}

		if index == 0 && request.FirstBlockReceiptIndex > 0 {
			if request.FirstBlockReceiptIndex >= uint64(len(blockReceipts)) {
				blockReceipts = nil
			} else {
				blockReceipts = blockReceipts[request.FirstBlockReceiptIndex:]
			}
		}

		receipts = append(receipts, eth.NewReceiptList(blockReceipts))
	}

	return receipts, nil
}

func (b *Bootstrap) originHeader(ctx context.Context, origin eth.HashOrNumber) (*types.Header, error) {
	if origin.Hash != (common.Hash{}) {
		return b.client.HeaderByHash(ctx, origin.Hash)
	}

	return b.client.HeaderByNumber(ctx, new(big.Int).SetUint64(origin.Number))
}

func validatePeerStatus(peerStatus mimicry.Status, snapshot *statusSnapshot) error {
	if peerStatus == nil {
		return nil
	}

	if peerStatus.GetNetworkID() != snapshot.networkID {
		return fmt.Errorf("peer network id %d does not match bootstrap RPC network id %d", peerStatus.GetNetworkID(), snapshot.networkID)
	}

	if !bytes.Equal(peerStatus.GetGenesis(), snapshot.genesis[:]) {
		return fmt.Errorf("peer genesis %x does not match bootstrap genesis %x", peerStatus.GetGenesis(), snapshot.genesis[:])
	}

	peerForkID, err := forkIDFromPeerStatus(peerStatus)
	if err != nil {
		return err
	}

	if err := snapshot.forkFilter(peerForkID); err != nil {
		return fmt.Errorf(
			"peer fork id 0x%x/%d is incompatible with bootstrap fork id 0x%x/%d: %w",
			peerForkID.Hash[:],
			peerForkID.Next,
			snapshot.forkID.Hash[:],
			snapshot.forkID.Next,
			err,
		)
	}

	return nil
}

func forkIDFromPeerStatus(peerStatus mimicry.Status) (forkid.ID, error) {
	hash := peerStatus.GetForkIDHash()
	if len(hash) != 4 {
		return forkid.ID{}, fmt.Errorf("peer fork id hash must be 4 bytes, got %d", len(hash))
	}

	var forkHash [4]byte
	copy(forkHash[:], hash)

	return forkid.ID{Hash: forkHash, Next: peerStatus.GetForkIDNext()}, nil
}

func encodeBlockBody(block *types.Block) (eth.BlockBody, error) {
	transactions, err := rlp.EncodeToRawList([]*types.Transaction(block.Transactions()))
	if err != nil {
		return eth.BlockBody{}, err
	}

	uncles, err := rlp.EncodeToRawList(block.Uncles())
	if err != nil {
		return eth.BlockBody{}, err
	}

	body := eth.BlockBody{
		Transactions: transactions,
		Uncles:       uncles,
	}

	if block.Withdrawals() != nil {
		withdrawals, err := rlp.EncodeToRawList([]*types.Withdrawal(block.Withdrawals()))
		if err != nil {
			return eth.BlockBody{}, err
		}

		body.Withdrawals = &withdrawals
	}

	return body, nil
}

func bootstrapNetwork(networkID uint64) (*params.ChainConfig, *types.Block, *big.Int, error) {
	switch networkID {
	case 1:
		return params.MainnetChainConfig, core.DefaultGenesisBlock().ToBlock(), copyBigInt(params.MainnetTerminalTotalDifficulty), nil
	case 11155111:
		return params.SepoliaChainConfig, core.DefaultSepoliaGenesisBlock().ToBlock(), copyBigInt(params.SepoliaChainConfig.TerminalTotalDifficulty), nil
	case 17000:
		return params.HoleskyChainConfig, core.DefaultHoleskyGenesisBlock().ToBlock(), copyBigInt(params.HoleskyChainConfig.TerminalTotalDifficulty), nil
	default:
		return nil, nil, nil, fmt.Errorf("unsupported bootstrap RPC network id: %d", networkID)
	}
}

func copyBigInt(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}

	return new(big.Int).Set(v)
}
