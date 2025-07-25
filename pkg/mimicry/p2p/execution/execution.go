package execution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	coordCache "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const PeerType = "execution"

type Peer struct {
	log logrus.FieldLogger

	nodeRecord   string
	handlers     *handler.Peer
	captureDelay time.Duration

	client *mimicry.Client

	// shared cache between clients
	sharedCache *coordCache.SharedCache

	txProc *processor.BatchItemProcessor[TransactionHashItem]

	network        *networks.Network
	chainConfig    *params.ChainConfig
	signer         types.Signer
	ethereumConfig *ethereum.Config

	name            string
	protocolVersion uint64
	implmentation   string
	version         string
	forkID          *xatu.ForkID
	capabilities    *[]p2p.Cap

	mu           *sync.Mutex
	ignoreBefore *time.Time
}

func New(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlers *handler.Peer, captureDelay time.Duration, sharedCache *coordCache.SharedCache, ethereumConfig *ethereum.Config) (*Peer, error) {
	client, err := mimicry.New(ctx, log, nodeRecord, "xatu")
	if err != nil {
		return nil, err
	}

	return &Peer{
		log:            log.WithField("node_record", nodeRecord),
		nodeRecord:     nodeRecord,
		handlers:       handlers,
		captureDelay:   captureDelay,
		client:         client,
		sharedCache:    sharedCache,
		ethereumConfig: ethereumConfig,
		network: &networks.Network{
			Name: networks.NetworkNameNone,
		},
		mu: &sync.Mutex{},
	}, nil
}

func (p *Peer) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	if p.handlers.CreateNewClientMeta == nil {
		return nil, errors.New("no CreateNewClientMeta handler")
	}

	meta, err := p.handlers.CreateNewClientMeta(ctx)
	if err != nil {
		return nil, err
	}

	ethereum := &xatu.ClientMeta_Ethereum{
		Network: &xatu.ClientMeta_Ethereum_Network{
			Name: string(p.network.Name),
			Id:   p.network.ID,
		},
		Execution: &xatu.ClientMeta_Ethereum_Execution{
			ForkId: p.forkID,
		},
	}

	meta.Ethereum = ethereum

	return meta, nil
}

func (p *Peer) Start(ctx context.Context) (<-chan error, error) {
	response := make(chan error)

	exporter, err := NewTransactionExporter(p.log, p.ExportTransactions)
	if err != nil {
		return nil, err
	}

	p.txProc, err = processor.NewBatchItemProcessor[TransactionHashItem](exporter,
		xatu.ImplementationLower()+"mimicry_execution",
		p.log,
		processor.WithMaxQueueSize(100000),
		processor.WithBatchTimeout(1*time.Second),
		processor.WithExportTimeout(1*time.Second),
		// TODO: technically this should actually be 256 and throttle requests to 1 per second(?)
		// https://github.com/ethereum/devp2p/blob/master/caps/eth.md#getpooledtransactions-0x09
		// I think we can get away with much higher as long as it doesn't go above the
		// max client message size.
		processor.WithMaxExportBatchSize(50000),
		processor.WithWorkers(1),
	)
	if err != nil {
		return nil, err
	}

	p.txProc.Start(ctx)

	p.client.OnHello(ctx, func(ctx context.Context, hello *mimicry.Hello) error {
		// setup client implementation and version info
		split := strings.SplitN(hello.Name, "/", 2)
		p.implmentation = strings.ToLower(split[0])

		if len(split) > 1 {
			p.version = split[1]
		}

		p.name = hello.Name
		p.capabilities = &hello.Caps
		p.protocolVersion = hello.Version

		p.log.WithFields(logrus.Fields{
			"implementation": p.implmentation,
			"version":        p.version,
		}).Debug("connected to client")

		return nil
	})

	p.client.OnStatus(ctx, func(ctx context.Context, status *mimicry.Status) error {
		if p.handlers.ExecutionStatus != nil {
			s := &xatu.ExecutionNodeStatus{NodeRecord: p.nodeRecord}

			s.Name = p.name
			s.ProtocolVersion = p.protocolVersion

			if p.capabilities != nil {
				for _, cap := range *p.capabilities {
					s.Capabilities = append(s.Capabilities, &xatu.ExecutionNodeStatus_Capability{
						Name:    cap.Name,
						Version: uint32(cap.Version),
					})
				}
			}

			if status != nil {
				s.NetworkId = status.NetworkID
				s.Head = status.LatestBlockHash[:]
				s.Genesis = status.Genesis[:]
				s.ForkId = &xatu.ExecutionNodeStatus_ForkID{
					Hash: status.ForkID.Hash[:],
					Next: status.ForkID.Next,
				}
			}

			if serr := p.handlers.ExecutionStatus(ctx, s); serr != nil {
				p.log.WithError(serr).Error("failed to handle execution status")
			}
		}

		// setup peer network/fork info
		if p.ethereumConfig != nil && p.ethereumConfig.OverrideNetworkName != "" {
			p.log.WithField("network", p.ethereumConfig.OverrideNetworkName).Info("Using override network name")
			p.network = &networks.Network{
				Name: networks.NetworkName(p.ethereumConfig.OverrideNetworkName),
				ID:   status.NetworkID,
			}
		} else {
			p.network = networks.DeriveFromID(status.NetworkID)
		}

		p.forkID = &xatu.ForkID{
			Hash: "0x" + fmt.Sprintf("%x", status.ForkID.Hash),
			Next: fmt.Sprintf("%d", status.ForkID.Next),
		}

		// setup signer
		switch p.network.Name {
		case networks.NetworkNameMainnet:
			p.chainConfig = params.MainnetChainConfig
		case networks.NetworkNameHolesky:
			p.chainConfig = params.HoleskyChainConfig
		case networks.NetworkNameSepolia:
			p.chainConfig = params.SepoliaChainConfig
		default:
			p.chainConfig = params.MainnetChainConfig
		}

		chainID := new(big.Int).SetUint64(p.network.ID)
		p.signer = types.NewCancunSigner(chainID)

		p.log.WithFields(logrus.Fields{
			"network":      p.network.Name,
			"fork_id_hash": "0x" + fmt.Sprintf("%x", status.ForkID.Hash),
			"fork_id_next": fmt.Sprintf("%d", status.ForkID.Next),
		}).Debug("got client status")

		// This is the avoid the initial deluge of transactions when a peer is first connected to.
		ignoreBefore := time.Now().Add(p.captureDelay)
		p.ignoreBefore = &ignoreBefore

		return nil
	})

	p.client.OnNewPooledTransactionHashes(ctx, func(ctx context.Context, hashes *mimicry.NewPooledTransactionHashes) error {
		if !p.shouldGetTransactions() {
			return nil
		}

		if p.handlers.DecoratedEvent != nil && hashes != nil {
			now := time.Now()
			for _, hash := range hashes.Hashes {
				if errT := p.processTransaction(ctx, now, hash); errT != nil {
					p.log.WithError(errT).Error("failed processing event")
				}
			}
		}

		return nil
	})

	p.client.OnTransactions(ctx, func(ctx context.Context, txs *mimicry.Transactions) error {
		if !p.shouldGetTransactions() {
			return nil
		}

		if p.handlers.DecoratedEvent != nil && txs != nil {
			now := time.Now()

			for _, tx := range *txs {
				_, retrieved := p.sharedCache.Transaction.GetOrSet(tx.Hash().String(), true, ttlcache.WithTTL[string, bool](1*time.Hour))
				// transaction was just set in shared cache, so we need to handle it
				if !retrieved {
					event, errT := p.handleTransaction(ctx, now, tx)
					if errT != nil {
						p.log.WithError(errT).Error("failed handling transaction")
					}

					if event != nil {
						if errT := p.handlers.DecoratedEvent(ctx, event); errT != nil {
							p.log.WithError(errT).Error("failed handling decorated event")
						}
					}
				}
			}
		}

		return nil
	})

	p.client.OnDisconnect(ctx, func(ctx context.Context, reason *mimicry.Disconnect) error {
		str := "unknown"
		if reason != nil {
			str = reason.Reason.String()
		}

		p.log.WithFields(logrus.Fields{
			"reason": str,
		}).Debug("disconnected from client")

		response <- errors.New("disconnected from peer (reason " + str + ")")

		return nil
	})

	p.log.Debug("attempting to connect to client")

	err = p.client.Start(ctx)
	if err != nil {
		p.log.WithError(err).Debug("failed to dial client")

		return nil, err
	}

	return response, nil
}

// typically when first connecting to a peer, a dump of their transaction pool is sent.
// not looking to get stale/old transactions, so we can just ignore the first batch.
func (p *Peer) shouldGetTransactions() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ignoreBefore == nil {
		// no ignore before time set, so we should get transactions
		return true
	}

	if time.Now().Before(*p.ignoreBefore) {
		// ignore time set and is still in the future, so we should not get transactions
		return false
	}

	// set ignore time to nil, so we should get transactions
	p.ignoreBefore = nil

	return true
}

func (p *Peer) processTransaction(ctx context.Context, now time.Time, hash common.Hash) error {
	// check if transaction is already in the shared cache, no need to fetch it again
	exists := p.sharedCache.Transaction.Get(hash.String())
	if exists == nil {
		item := TransactionHashItem{
			Hash: hash,
			Seen: now,
		}
		if err := p.txProc.Write(ctx, []*TransactionHashItem{&item}); err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) Stop(ctx context.Context) error {
	if err := p.txProc.Shutdown(ctx); err != nil {
		return err
	}

	if p.client != nil {
		if err := p.client.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Peer) Type() string {
	return PeerType
}
