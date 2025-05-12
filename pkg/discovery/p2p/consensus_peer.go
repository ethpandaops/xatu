package p2p

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ConsensusPeer struct {
	log logrus.FieldLogger

	nodeRecord string

	// TODO: replace with consensus ethcore client
	client *mimicry.Client

	handlerFunc func(ctx context.Context, status *xatu.ConsensusNodeStatus)
}

func NewConsensusPeer(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlerFunc func(ctx context.Context, status *xatu.ConsensusNodeStatus)) (*ConsensusPeer, error) {
	client, err := mimicry.New(ctx, log, nodeRecord, "xatu")
	if err != nil {
		return nil, err
	}

	return &ConsensusPeer{
		log:         log.WithField("node_record", nodeRecord),
		nodeRecord:  nodeRecord,
		client:      client,
		handlerFunc: handlerFunc,
	}, nil
}

func (p *ConsensusPeer) Start(ctx context.Context) (<-chan error, error) {
	response := make(chan error, 1)

	p.client.OnHello(ctx, func(ctx context.Context, hello *mimicry.Hello) error {
		// TODO: implement
		return nil
	})

	p.client.OnStatus(ctx, func(ctx context.Context, status *mimicry.Status) error {
		// TODO: implement
		return nil
	})

	p.client.OnDisconnect(ctx, func(ctx context.Context, reason *mimicry.Disconnect) error {
		str := "unknown"
		if reason != nil {
			str = reason.Reason.String()
		}

		response <- errors.New("disconnected from peer (reason " + str + ")")

		return nil
	})

	err := p.client.Start(ctx)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (p *ConsensusPeer) Stop(ctx context.Context) error {
	return p.client.Stop(ctx)
}
