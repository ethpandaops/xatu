package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

type rpcBootstrap struct {
	log logrus.FieldLogger

	client *ethclient.Client

	mu          sync.Mutex
	statusCache *rpcBootstrapStatus
}

type rpcBootstrapStatus struct {
	networkID               uint64
	genesis                 common.Hash
	terminalTotalDifficulty *big.Int
	headNumber              uint64
	headHash                common.Hash
	forkID                  forkid.ID
	expiresAt               time.Time
}

var sharedRPCBootstraps = struct {
	sync.Mutex
	byURL map[string]*rpcBootstrap
}{
	byURL: map[string]*rpcBootstrap{},
}

func newRPCBootstrap(ctx context.Context, log logrus.FieldLogger, url string) (*rpcBootstrap, error) {
	sharedRPCBootstraps.Lock()

	if bootstrap, ok := sharedRPCBootstraps.byURL[url]; ok {
		sharedRPCBootstraps.Unlock()

		return bootstrap, nil
	}

	sharedRPCBootstraps.Unlock()

	rpcCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(rpcCtx, url)
	if err != nil {
		return nil, err
	}

	bootstrap := &rpcBootstrap{
		log:    log.WithField("bootstrap_rpc_url", url),
		client: client,
	}

	sharedRPCBootstraps.Lock()

	if existing, ok := sharedRPCBootstraps.byURL[url]; ok {
		sharedRPCBootstraps.Unlock()
		client.Close()

		return existing, nil
	}

	sharedRPCBootstraps.byURL[url] = bootstrap
	sharedRPCBootstraps.Unlock()

	return bootstrap, nil
}

func (b *rpcBootstrap) status(ctx context.Context, protocolVersion uint, peerStatus mimicry.Status) (mimicry.Status, error) {
	snapshot, err := b.currentStatus(ctx)
	if err != nil {
		return nil, err
	}

	if err := validatePeerStatus(peerStatus, snapshot.networkID, snapshot.genesis, snapshot.forkID); err != nil {
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

func (b *rpcBootstrap) currentStatus(ctx context.Context) (*rpcBootstrapStatus, error) {
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

	b.statusCache = &rpcBootstrapStatus{
		networkID:               networkID.Uint64(),
		genesis:                 genesis.Hash(),
		terminalTotalDifficulty: new(big.Int).Set(terminalTotalDifficulty),
		headNumber:              head.Number.Uint64(),
		headHash:                head.Hash(),
		forkID:                  forkID,
		expiresAt:               time.Now().Add(statusCacheTTL),
	}

	return b.statusCache, nil
}

func validatePeerStatus(peerStatus mimicry.Status, networkID uint64, genesis common.Hash, forkID forkid.ID) error {
	if peerStatus == nil {
		return nil
	}

	if peerStatus.GetNetworkID() != networkID {
		return fmt.Errorf("peer network id %d does not match bootstrap RPC network id %d", peerStatus.GetNetworkID(), networkID)
	}

	if !bytes.Equal(peerStatus.GetGenesis(), genesis[:]) {
		return fmt.Errorf("peer genesis %x does not match bootstrap genesis %x", peerStatus.GetGenesis(), genesis[:])
	}

	if !bytes.Equal(peerStatus.GetForkIDHash(), forkID.Hash[:]) || peerStatus.GetForkIDNext() != forkID.Next {
		return fmt.Errorf(
			"peer fork id 0x%x/%d does not match bootstrap fork id 0x%x/%d",
			peerStatus.GetForkIDHash(),
			peerStatus.GetForkIDNext(),
			forkID.Hash[:],
			forkID.Next,
		)
	}

	return nil
}

func (b *rpcBootstrap) headers(ctx context.Context, request *eth.GetBlockHeadersRequest) ([]*types.Header, error) {
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
			next = current + step
		}

		nextHeader, herr := b.client.HeaderByNumber(rpcCtx, new(big.Int).SetUint64(next))
		if herr != nil {
			if errors.Is(herr, ethereum.NotFound) {
				break
			}

			return headers, herr
		}

		if nextHeader == nil {
			break
		}

		headers = append(headers, nextHeader)
	}

	return headers, nil
}

func (b *rpcBootstrap) bodies(ctx context.Context, hashes []common.Hash) ([]eth.BlockBody, error) {
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

			return bodies, err
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

func (b *rpcBootstrap) receipts(ctx context.Context, request mimicry.ReceiptRequest) ([]*eth.ReceiptList, error) {
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

			return receipts, err
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

func (b *rpcBootstrap) originHeader(ctx context.Context, origin eth.HashOrNumber) (*types.Header, error) {
	if origin.Hash != (common.Hash{}) {
		return b.client.HeaderByHash(ctx, origin.Hash)
	}

	return b.client.HeaderByNumber(ctx, new(big.Int).SetUint64(origin.Number))
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
