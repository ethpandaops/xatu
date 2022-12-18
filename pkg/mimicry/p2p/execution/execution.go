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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethpandaops/xatu/pkg/execution"
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

	client *execution.Client

	// don't send duplicate events from the same client
	duplicateCache *cache.DuplicateCache
	// shared cache between clients
	sharedCache *coordCache.SharedCache

	txProc *processor.BatchItemProcessor[TransactionHashItem]

	network     *networks.Network
	chainConfig *params.ChainConfig
	signer      types.Signer

	implmentation string
	version       string
	forkID        *xatu.ForkID
}

func New(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlers *handler.Peer, sharedCache *coordCache.SharedCache) (*Peer, error) {
	client, err := execution.New(ctx, log, nodeRecord)
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
			Implementation: p.implmentation,
			Version:        p.version,
			ForkId:         p.forkID,
			NodeRecord:     p.nodeRecord,
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
		// I think you can get away with have it as 4096 though as thats the technical limit
		// of the PooledTransactions response EVEN though most clients support way more.
		processor.WithMaxExportBatchSize(50000),
	)

	p.duplicateCache.Start()

	p.client.OnHello(ctx, func(ctx context.Context, hello *execution.Hello) error {
		// setup client implementation and version info
		split := strings.SplitN(hello.Name, "/", 2)
		p.implmentation = split[0]
		p.version = split[1]

		p.log.WithFields(logrus.Fields{
			"implementation": p.implmentation,
			"version":        p.version,
		}).Info("connected to client")

		return nil
	})

	p.client.OnStatus(ctx, func(ctx context.Context, status *execution.Status) error {
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
		}).Info("got client status")

		return nil
	})

	p.client.OnNewPooledTransactionHashes(ctx, func(ctx context.Context, hashes *execution.NewPooledTransactionHashes) error {
		now := time.Now()
		if hashes != nil {
			for _, hash := range *hashes {
				// check if transaction is already in the shared cache
				tx := p.sharedCache.Transaction.Get(hash.String())
				if tx != nil {
					event, errT := p.handleTransaction(ctx, now, tx.Value())
					if errT != nil {
						p.log.WithError(errT).Error("failed handling transaction")
					}

					if event != nil {
						if errT := p.handlers.DecoratedEvent(ctx, event); errT != nil {
							p.log.WithError(errT).Error("failed handling decorated event")
						}
					}
				} else {
					item := TransactionHashItem{
						Hash: hash,
						Seen: now,
					}
					p.txProc.Write(&item)
				}
			}
		}

		return nil
	})

	p.client.OnTransactions(ctx, func(ctx context.Context, txs *execution.Transactions) error {
		if txs != nil {
			now := time.Now()
			for _, tx := range *txs {
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
		return nil
	})

	p.client.OnDisconnect(ctx, func(ctx context.Context, reason *execution.Disconnect) error {
		str := "unknown"
		if reason != nil {
			str = reason.Reason.String()
		}

		response <- errors.New("disconnected from peer (reason " + str + ")")

		return nil
	})

	err = p.client.Start(ctx)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *Peer) Stop(ctx context.Context) error {
	return p.client.Stop(ctx)
}

func (p *Peer) Type() string {
	return PeerType
}
