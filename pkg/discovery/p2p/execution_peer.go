package p2p

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionPeer struct {
	log logrus.FieldLogger

	nodeRecord string

	client *mimicry.Client

	hello *mimicry.Hello

	handlerFunc func(ctx context.Context, status *xatu.ExecutionNodeStatus)
}

func NewExecutionPeer(ctx context.Context, log logrus.FieldLogger, nodeRecord string, handlerFunc func(ctx context.Context, status *xatu.ExecutionNodeStatus)) (*ExecutionPeer, error) {
	client, err := mimicry.New(ctx, log, nodeRecord, "xatu")
	if err != nil {
		return nil, err
	}

	return &ExecutionPeer{
		log:         log.WithField("node_record", nodeRecord),
		nodeRecord:  nodeRecord,
		client:      client,
		handlerFunc: handlerFunc,
	}, nil
}

func (p *ExecutionPeer) Start(ctx context.Context) (<-chan error, error) {
	response := make(chan error, 1)

	p.client.OnHello(ctx, func(ctx context.Context, hello *mimicry.Hello) error {
		p.hello = hello

		return nil
	})

	p.client.OnStatus(ctx, func(ctx context.Context, status *mimicry.Status) error {
		s := &xatu.ExecutionNodeStatus{NodeRecord: p.nodeRecord}

		if p.hello != nil {
			s.Name = p.hello.Name
			s.ProtocolVersion = p.hello.Version

			if p.hello.Caps != nil {
				for _, cap := range p.hello.Caps {
					s.Capabilities = append(s.Capabilities, &xatu.ExecutionNodeStatus_Capability{
						Name:    cap.Name,
						Version: uint32(cap.Version),
					})
				}
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

		p.handlerFunc(ctx, s)
		response <- nil

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

func (p *ExecutionPeer) Stop(ctx context.Context) error {
	return p.client.Stop(ctx)
}
