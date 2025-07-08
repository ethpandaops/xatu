package p2p

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/crawler"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/host"
	"github.com/ethpandaops/ethcore/pkg/discovery"
	"github.com/ethpandaops/ethcore/pkg/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/sirupsen/logrus"
)

type ConsensusCrawler struct {
	log           *logrus.Entry
	crawler       *crawler.Crawler
	manual        *discovery.Manual
	eventHandlers map[string]func(*xatu.ConsensusNodeStatus)
	networkID     uint64
	mu            sync.RWMutex
}

func NewConsensusCrawler(ctx context.Context, log logrus.FieldLogger, beaconNodeURL string, networkID uint64) (*ConsensusCrawler, error) {
	crawlerConfig := &crawler.Config{
		Node: &host.Config{
			IPAddr: net.ParseIP("127.0.0.1"),
		},
		Beacon: &ethereum.Config{
			BeaconNodeAddress: beaconNodeURL,
		},
		DialConcurrency:  10,
		DialTimeout:      5 * time.Second,
		CooloffDuration:  10 * time.Second,
		UserAgent:        xatu.Full(),
		MaxRetryAttempts: 1,
		RetryBackoff:     2 * time.Second,
	}

	manual := &discovery.Manual{}

	// The crawler is noisy, but we want to see INFO logs for debugging
	// Create a new logger instance specifically for the crawler
	crawlerLogger := logrus.New()
	crawlerLogger.SetLevel(logrus.WarnLevel)

	// Init the crawler.
	c := crawler.New(crawlerLogger, crawlerConfig, manual)
	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start crawler: %w", err)
	}

	cc := &ConsensusCrawler{
		log:           log.WithField("module", "consensus_crawler").WithField("network_id", networkID),
		crawler:       c,
		manual:        manual,
		eventHandlers: make(map[string]func(*xatu.ConsensusNodeStatus)),
		networkID:     networkID,
	}

	c.OnSuccessfulCrawl(cc.handleSuccessfulCrawl)
	c.OnFailedCrawl(cc.handleFailedCrawl)

	select {
	case <-c.OnReady:
		cc.log.Info("Consensus crawler ready")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return cc, nil
}

func (c *ConsensusCrawler) AddNodeRecord(nodeRecord string, handler func(*xatu.ConsensusNodeStatus)) error {
	node, err := enode.Parse(enode.ValidSchemes, nodeRecord)
	if err != nil {
		return fmt.Errorf("failed to parse node record: %w", err)
	}

	// Store the handler keyed by the node record string
	c.mu.Lock()
	c.eventHandlers[nodeRecord] = handler
	c.mu.Unlock()

	if err := c.manual.AddNode(context.Background(), node); err != nil {
		c.mu.Lock()
		delete(c.eventHandlers, nodeRecord)
		c.mu.Unlock()

		return fmt.Errorf("failed to add node to crawler: %w", err)
	}

	return nil
}

func (c *ConsensusCrawler) handleSuccessfulCrawl(peerID peer.ID, enr *enode.Node, status *common.Status, metadata *common.MetaData) {
	agentVersion := c.crawler.GetPeerAgentVersion(peerID)

	// Get the ENR string
	var nodeRecord string
	if enr != nil {
		nodeRecord = enr.String()
	}

	// Look up the handler by the node record
	c.mu.RLock()
	handler, exists := c.eventHandlers[nodeRecord]
	c.mu.RUnlock()

	if !exists {
		return
	}

	// Extract node_id and peer_id.
	var (
		nodeID    string
		peerIDStr = peerID.String()
	)

	if enr != nil {
		nodeID = enr.ID().String()
	}

	nodeStatus := &xatu.ConsensusNodeStatus{
		NodeRecord:     nodeRecord,
		NodeId:         nodeID,
		PeerId:         peerIDStr,
		Name:           agentVersion,
		ForkDigest:     status.ForkDigest[:],
		FinalizedRoot:  status.FinalizedRoot[:],
		FinalizedEpoch: make([]byte, 8),
		HeadRoot:       status.HeadRoot[:],
		HeadSlot:       make([]byte, 8),
		NetworkId:      c.networkID,
	}

	// Convert epoch and slot to bytes
	epochBytes := nodeStatus.FinalizedEpoch
	slotBytes := nodeStatus.HeadSlot

	for i := 0; i < 8; i++ {
		epochBytes[i] = byte(status.FinalizedEpoch >> (8 * (7 - i)))
		slotBytes[i] = byte(status.HeadSlot >> (8 * (7 - i)))
	}

	handler(nodeStatus)

	// Clean up the handler after use
	c.mu.Lock()
	delete(c.eventHandlers, nodeRecord)
	c.mu.Unlock()
}

func (c *ConsensusCrawler) handleFailedCrawl(peerID peer.ID, err crawler.CrawlError) {
	c.log.WithError(err).WithField("peer_id", peerID).Error("Failed to crawl peer")
}

func (c *ConsensusCrawler) Stop(ctx context.Context) error {
	if err := c.crawler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop crawler: %w", err)
	}

	c.mu.Lock()
	c.eventHandlers = make(map[string]func(*xatu.ConsensusNodeStatus))
	c.mu.Unlock()

	return nil
}
