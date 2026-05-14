package p2p

import (
	"context"
	"sync"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusPeer struct {
	log        observability.ContextualLogger
	nodeRecord string
	handler    func(*xatu.ConsensusNodeStatus)
	crawler    *ConsensusCrawler
}

func NewConsensusPeer(
	_ context.Context,
	log observability.ContextualLogger, nodeRecord string,
	handler func(*xatu.ConsensusNodeStatus),
	crawler *ConsensusCrawler,
) (*ConsensusPeer, error) {
	return &ConsensusPeer{
		log:        log.WithField("node_record", nodeRecord),
		nodeRecord: nodeRecord,
		handler:    handler,
		crawler:    crawler,
	}, nil
}

func (p *ConsensusPeer) Start(_ context.Context) <-chan error {
	errChan := make(chan error, 1)

	var once sync.Once

	err := p.crawler.AddNodeRecord(p.nodeRecord, func(status *xatu.ConsensusNodeStatus) {
		p.handler(status)

		once.Do(func() {
			close(errChan)
		})
	})

	if err != nil {
		errChan <- err

		close(errChan)
	}

	return errChan
}

func (p *ConsensusPeer) Stop(_ context.Context) error {
	return nil
}

func (p *ConsensusPeer) Type() string {
	return "consensus"
}
