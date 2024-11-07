package beaconp2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	perrors "github.com/pkg/errors"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/crawler"
	"github.com/ethpandaops/xatu/pkg/discovery/provider"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Node struct {
	log logrus.FieldLogger

	config *Config

	crawler *crawler.Crawler

	enodeProvider provider.EnodeProvider
}

func NewNode(ctx context.Context, log logrus.FieldLogger, config *Config) (*Node, error) {
	p2pDisc, err := NewNodeDiscoverer(
		config.Discovery.Type,
		config.Discovery.Config,
		log,
	)
	if err != nil {
		return nil, perrors.Wrap(err, "failed to create beacon node discoverer")
	}

	c := crawler.New(
		log,
		&config.Crawler,
		xatu.FullWithModule(xatu.ModuleName_DISCOVERY),
		"xatu_discovery",
		provider.NewWrappedNodeFinder(p2pDisc),
	)

	return &Node{
		log:           log.WithField("module", "discovery/beacon_p2p/node"),
		config:        config,
		crawler:       c,
		enodeProvider: p2pDisc,
	}, nil
}

func (p *Node) Start(ctx context.Context) error {
	p.crawler.OnPeerStatusUpdated(func(peerID peer.ID, status *common.Status) {
		agentVersion := "unknown"

		rawAgentVersion, err := p.crawler.GetNode().Peerstore().Get(peerID, "AgentVersion")
		if err != nil {
			p.log.WithError(err).Debug("Failed to get agent version")
		} else {
			agentVersion = rawAgentVersion.(string)
		}

		p.log.WithFields(logrus.Fields{
			"finalized_epoch": status.FinalizedEpoch,
			"finalized_root":  status.FinalizedRoot,
			"head_slot":       status.HeadSlot,
			"head_root":       status.HeadRoot,
			"fork_digest":     status.ForkDigest,
			"peer_id":         peerID,
			"agent_version":   agentVersion,
		}).Info("Status received from peer!")
	})

	p.crawler.OnMetadataReceived(func(peerID peer.ID, metadata *common.MetaData) {
		p.log.WithFields(logrus.Fields{
			"peer_id":       peerID,
			"metadata":      metadata,
			"agent_version": p.crawler.GetPeerAgentVersion(peerID),
		}).Info("Metadata received from peer!")
	})

	if err := p.crawler.Start(ctx); err != nil {
		return perrors.Wrap(err, "failed to start crawler")
	}

	return nil
}

func (p *Node) Stop(ctx context.Context) error {
	return p.crawler.Stop(ctx)
}
