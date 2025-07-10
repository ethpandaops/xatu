package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/consensus/mimicry/crawler"
	"github.com/ethpandaops/ethcore/pkg/discovery"
	coreenr "github.com/ethpandaops/ethcore/pkg/ethereum/node/enr"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

type ConsensusCrawler struct {
	log           *logrus.Entry
	crawler       *crawler.Crawler
	manual        *discovery.Manual
	eventHandlers map[string]func(*xatu.ConsensusNodeStatus)
	mu            sync.RWMutex
}

func NewConsensusCrawler(ctx context.Context, log logrus.FieldLogger, cfg *crawler.Config) (*ConsensusCrawler, error) {
	// The crawler is noisy, but we want to see INFO logs for debugging
	// Create a new logger instance specifically for the crawler
	crawlerLogger := logrus.New()
	crawlerLogger.SetLevel(logrus.WarnLevel)

	// Use manual discovery for our crawler instance.
	manual := &discovery.Manual{}

	// Init the crawler.
	c := crawler.New(crawlerLogger, cfg, manual)
	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start crawler: %w", err)
	}

	cc := &ConsensusCrawler{
		log:           log.WithField("module", "consensus_crawler"),
		crawler:       c,
		manual:        manual,
		eventHandlers: make(map[string]func(*xatu.ConsensusNodeStatus)),
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

func (c *ConsensusCrawler) Stop(ctx context.Context) error {
	if err := c.crawler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop crawler: %w", err)
	}

	c.mu.Lock()
	c.eventHandlers = make(map[string]func(*xatu.ConsensusNodeStatus))
	c.mu.Unlock()

	return nil
}

func (c *ConsensusCrawler) handleSuccessfulCrawl(crawl *crawler.SuccessfulCrawl) {
	// Get the ENR string
	var nodeRecord string
	if crawl.ENR != nil {
		nodeRecord = crawl.ENR.String()
	}

	// Look up the handler by the node record
	c.mu.RLock()
	handler, exists := c.eventHandlers[nodeRecord]
	c.mu.RUnlock()

	if !exists {
		return
	}

	nodeStatus, err := c.createNodeStatus(crawl)
	if err != nil {
		c.log.WithError(err).Error("Failed to create node status")

		return
	}

	handler(nodeStatus)

	// Clean up the handler after use
	c.mu.Lock()
	delete(c.eventHandlers, nodeRecord)
	c.mu.Unlock()
}

func (c *ConsensusCrawler) handleFailedCrawl(crawl *crawler.FailedCrawl) {
	c.log.WithError(crawl.Error).WithField("peer_id", crawl.PeerID).Error("Failed to crawl peer")
}

func (c *ConsensusCrawler) createNodeStatus(crawl *crawler.SuccessfulCrawl) (*xatu.ConsensusNodeStatus, error) {
	var nodeRecord string
	if crawl.ENR != nil {
		nodeRecord = crawl.ENR.String()
	}

	enrRecord, err := coreenr.Parse(nodeRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ENR record: %w", err)
	}

	return &xatu.ConsensusNodeStatus{
		NodeRecord:     nodeRecord,
		NodeId:         crawl.NodeID,
		PeerId:         crawl.PeerID.String(),
		Name:           crawl.AgentVersion,
		ForkDigest:     crawl.Status.ForkDigest[:],
		NextForkDigest: extractNFD(enrRecord),
		FinalizedRoot:  crawl.Status.FinalizedRoot[:],
		FinalizedEpoch: encodeUint64ToBytes(uint64(crawl.Status.FinalizedEpoch)),
		HeadRoot:       crawl.Status.HeadRoot[:],
		HeadSlot:       encodeUint64ToBytes(uint64(crawl.Status.HeadSlot)),
		Cgc:            extractCGC(enrRecord),
		NetworkId:      crawl.NetworkID,
	}, nil
}

func encodeUint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		bytes[i] = byte(value >> (8 * (7 - i)))
	}

	return bytes
}

func extractCGC(enr *coreenr.ENR) []byte {
	if enr != nil {
		if cgc := enr.GetCGC(); cgc != nil {
			return *cgc
		}
	}

	return nil
}

func extractNFD(enr *coreenr.ENR) []byte {
	if enr != nil {
		if nfd := enr.GetNFD(); nfd != nil {
			return *nfd
		}
	}

	return nil
}
