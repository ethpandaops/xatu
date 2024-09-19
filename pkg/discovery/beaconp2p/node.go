package beaconp2p

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/go-co-op/gocron"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	perrors "github.com/pkg/errors"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry"
	"github.com/ethpandaops/xatu/pkg/discovery/crawler"
	"github.com/ethpandaops/xatu/pkg/discovery/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Node struct {
	log logrus.FieldLogger

	config *Config

	client *mimicry.Client

	enodeProvider crawler.EnodeProvider

	statusSet bool

	beacon *ethereum.BeaconNode

	handlerFunc func(ctx context.Context, status *xatu.ExecutionNodeStatus)
}

func NewNode(ctx context.Context, log logrus.FieldLogger, config *Config) (*Node, error) {
	beacon, err := ethereum.NewBeaconNode(ctx, "upstream", config.Ethereum, log)
	if err != nil {
		return nil, perrors.Wrap(err, "failed to create beacon node")
	}

	client := mimicry.NewClient(log, &config.Node, mimicry.Mode(mimicry.ModeDiscovery), "xatu")

	// Disable discv4 = false

	p2pDisc, err := NewNodeDiscoverer(config.Discovery.Type, config.Discovery.Config, log)
	if err != nil {
		return nil, perrors.Wrap(err, "failed to create beacon node discoverer")
	}

	p2pDisc.RegisterHandler(func(ctx context.Context, node *enode.Node, source string) error {
		log.WithFields(logrus.Fields{
			"enr":    node.String(),
			"source": source,
		}).Info("Enode received")

		return client.ConnectToPeer(ctx, peer.AddrInfo{
			ID: peer.ID(node.ID().String()),
			Addrs: []ma.Multiaddr{
				ma.StringCast(node.URLv4()),
			},
		}, node)
	})

	return &Node{
		log:           log.WithField("module", "discovery/beacon_p2p/node"),
		config:        config,
		beacon:        beacon,
		client:        client,
		enodeProvider: p2pDisc,
		statusSet:     false,
	}, nil
}

func (p *Node) Start(ctx context.Context) error {
	p.client.OnStatusFromPeer(func(peerID peer.ID, status *common.Status) {
		p.log.WithFields(logrus.Fields{
			"finalized_epoch": status.FinalizedEpoch,
			"finalized_root":  status.FinalizedRoot,
			"head_slot":       status.HeadSlot,
			"head_root":       status.HeadRoot,
			"fork_digest":     status.ForkDigest,
			"peer_id":         peerID,
		}).Info("Status received from peer!")
	})

	p.beacon.OnReady(ctx, func(ctx context.Context) error {
		p.log.Info("Upstream beacon node is ready!")

		p.beacon.Node().OnHead(ctx, func(ctx context.Context, event *v1.HeadEvent) error {
			logctx := p.log.WithFields(logrus.Fields{
				"slot":  event.Slot,
				"event": "head",
			})

			logctx.Debug("Beacon head event received")

			if err := p.fetchAndSetStatus(ctx); err != nil {
				logctx.WithError(err).Error("Failed to fetch and set status")
			}

			return nil
		})

		// Start crons
		if err := p.startCrons(ctx); err != nil {
			return err
		}

		return nil
	})

	if err := p.beacon.Start(ctx); err != nil {
		return perrors.Wrap(err, "failed to start beacon node")
	}

	return nil
}

func (p *Node) startDiscovery(ctx context.Context) error {
	p.log.Info("Starting enode discovery service")

	if err := p.enodeProvider.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (p *Node) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("60s").Do(func() {
		if err := p.fetchAndSetStatus(ctx); err != nil {
			p.log.WithError(err).Error("Failed to fetch and set status")
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}

func (p *Node) fetchAndSetStatus(ctx context.Context) error {
	status := &common.Status{}

	// Fetch the status from the upstream beacon node.
	checkpoint, err := p.beacon.Node().FetchFinality(ctx, "head")
	if err != nil {
		return err
	}

	status.FinalizedRoot = tree.Root(checkpoint.Finalized.Root)
	status.FinalizedEpoch = common.Epoch(checkpoint.Finalized.Epoch)

	headers, err := p.beacon.Node().FetchBeaconBlockHeader(ctx, &api.BeaconBlockHeaderOpts{
		Block: "head",
	})
	if err != nil {
		return err
	}

	status.HeadRoot = tree.Root(headers.Header.Message.StateRoot)
	status.HeadSlot = common.Slot(headers.Header.Message.Slot)

	forkDigest, err := p.beacon.ForkDigest()
	if err != nil {
		return err
	}

	status.ForkDigest = common.ForkDigest(forkDigest)

	// Set the status on the mimicry node.
	p.client.SetStatus(status)

	p.log.WithFields(logrus.Fields{
		"finalized_epoch": status.FinalizedEpoch,
		"finalized_root":  status.FinalizedRoot,
		"head_slot":       status.HeadSlot,
		"head_root":       status.HeadRoot,
		"fork_digest":     status.ForkDigest,
	}).Info("Status set on mimicry node")

	if !p.statusSet {
		p.statusSet = true

		// Start the libp2p node.
		if err := p.client.Start(ctx); err != nil {
			p.log.WithError(err).Fatal("Failed to start libp2p node")
		}

		// We now have a valid status and can start feeding peers in to the mimicry node.
		if err := p.startDiscovery(ctx); err != nil {
			p.log.WithError(err).Fatal("Failed to start discovery")
		}
	}

	return nil
}

func (p *Node) Stop(ctx context.Context) error {
	return p.client.Stop(ctx)
}
