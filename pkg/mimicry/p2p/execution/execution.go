package execution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	coordCache "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/execution/cache"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const PeerType = "execution"

type Peer struct {
	log logrus.FieldLogger

	nodeRecord string
	handlers   *handler.Peer

	client *mimicry.Client

	// don't send duplicate events from the same client
	duplicateCache *cache.DuplicateCache
	// shared cache between clients
	sharedCache *coordCache.SharedCache

	txProc *processor.BatchItemProcessor[TransactionHashItem]

	network     *networks.Network
	chainConfig *params.ChainConfig
	signer      types.Signer

	name            string
	protocolVersion uint64
	implmentation   string
	version         string
	forkID          *xatu.ForkID
	capabilities    *[]p2p.Cap
}

func New(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlers *handler.Peer, sharedCache *coordCache.SharedCache) (*Peer, error) {
	client, err := mimicry.New(ctx, log, nodeRecord, "xatu")
	if err != nil {
		return nil, err
	}

	duplicateCache := cache.NewDuplicateCache()

	return &Peer{
		log:            log.WithField("node_record", nodeRecord),
		nodeRecord:     nodeRecord,
		handlers:       handlers,
		client:         client,
		duplicateCache: duplicateCache,
		sharedCache:    sharedCache,
		network: &networks.Network{
			Name: networks.NetworkNameNone,
		},
	}, nil
}

func (p *Peer) createNewClientMeta(ctx context.Context) (*xatu.ClientMeta, error) {
	if p.handlers.CreateNewClientMeta == nil {
		return nil, nil
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

	p.txProc = processor.NewBatchItemProcessor[TransactionHashItem](exporter,
		p.log,
		processor.WithMaxQueueSize(100000),
		processor.WithBatchTimeout(1*time.Second),
		processor.WithExportTimeout(1*time.Second),
		// TODO: technically this should actually be 256 and throttle requests to 1 per second(?)
		// https://github.com/ethereum/devp2p/blob/master/caps/eth.md#getpooledtransactions-0x09
		// I think we can get away with much higher as long as it doesn't go above the
		// max client message size.
		processor.WithMaxExportBatchSize(50000),
	)

	p.duplicateCache.Start()

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
				s.TotalDifficulty = status.TD.String()
				s.Head = status.Head[:]
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

		// setup signer and chain config for working out transaction "from" addresses
		networkID := status.NetworkID
		p.chainConfig = params.AllEthashProtocolChanges
		chainID := new(big.Int).SetUint64(networkID)
		p.chainConfig.ChainID = chainID
		p.chainConfig.EIP155Block = big.NewInt(0)
		p.signer = types.MakeSigner(p.chainConfig, big.NewInt(0))

		// setup peer network/fork info
		p.network = networks.DeriveFromID(status.NetworkID)
		p.forkID = &xatu.ForkID{
			Hash: "0x" + fmt.Sprintf("%x", status.ForkID.Hash),
			Next: fmt.Sprintf("%d", status.ForkID.Next),
		}

		p.log.WithFields(logrus.Fields{
			"network":      p.network.Name,
			"fork_id_hash": "0x" + fmt.Sprintf("%x", status.ForkID.Hash),
			"fork_id_next": fmt.Sprintf("%d", status.ForkID.Next),
		}).Debug("got client status")

		return nil
	})

	p.client.OnNewPooledTransactionHashes(ctx, func(ctx context.Context, hashes *mimicry.NewPooledTransactionHashes) error {
		now := time.Now()
		if p.handlers.DecoratedEvent != nil && hashes != nil {
			for _, hash := range *hashes {
				if errT := p.processTransaction(ctx, now, hash); errT != nil {
					p.log.WithError(errT).Error("failed processing event")
				}
			}
		}

		return nil
	})

	p.client.OnNewPooledTransactionHashes68(ctx, func(ctx context.Context, hashes *mimicry.NewPooledTransactionHashes68) error {
		now := time.Now()
		if p.handlers.DecoratedEvent != nil && hashes != nil {
			// TODO: handle eth68+ transaction size/types as well
			for _, hash := range hashes.Transactions {
				if errT := p.processTransaction(ctx, now, hash); errT != nil {
					p.log.WithError(errT).Error("failed processing event")
				}
			}
		}

		return nil
	})

	p.client.OnTransactions(ctx, func(ctx context.Context, txs *mimicry.Transactions) error {
		if p.handlers.DecoratedEvent != nil && txs != nil {
			now := time.Now()
			for _, tx := range *txs {
				// only process transactions we haven't seen before across all peers
				exists := p.sharedCache.Transaction.Get(tx.Hash().String())
				if exists == nil {
					p.sharedCache.Transaction.Set(tx.Hash().String(), tx, ttlcache.DefaultTTL)

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

func (p *Peer) processTransaction(ctx context.Context, now time.Time, hash common.Hash) error {
	// check if transaction is already in the shared cache, no need to fetch it again
	exists := p.sharedCache.Transaction.Get(hash.String())
	if exists == nil {
		item := TransactionHashItem{
			Hash: hash,
			Seen: now,
		}
		p.txProc.Write(&item)
	}

	return nil
}

func (p *Peer) Stop(ctx context.Context) error {
	p.duplicateCache.Stop()

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
