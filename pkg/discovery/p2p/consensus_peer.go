package p2p

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusPeer struct {
	log logrus.FieldLogger

	nodeRecord string

	handlerFunc func(ctx context.Context, status *xatu.ConsensusNodeStatus)
}

func NewConsensusPeer(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlerFunc func(ctx context.Context, status *xatu.ConsensusNodeStatus)) (*ConsensusPeer, error) {
	return &ConsensusPeer{
		log:         log.WithField("node_record", nodeRecord),
		nodeRecord:  nodeRecord,
		handlerFunc: handlerFunc,
	}, nil
}

func (p *ConsensusPeer) Start(ctx context.Context) (<-chan error, error) {
	response := make(chan error, 1)

	// TODO: implement

	return response, nil
}

func (p *ConsensusPeer) Stop(ctx context.Context) error {
	return nil
}
