package p2p

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"

	"github.com/ethpandaops/xatu/pkg/internal/rpcbootstrap"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ExecutionPeer struct {
	log observability.ContextualLogger

	nodeRecord string

	client *mimicry.Client

	hello *mimicry.Hello

	disconnectReason atomic.Pointer[string]

	handlerFunc func(ctx context.Context, status *xatu.ExecutionNodeStatus)
}

func NewExecutionPeer(ctx context.Context, log observability.ContextualLogger, nodeRecord, bootstrapRPCURL string, handlerFunc func(ctx context.Context, status *xatu.ExecutionNodeStatus)) (*ExecutionPeer, error) {
	opts := []mimicry.Option{}

	if bootstrapRPCURL != "" {
		bootstrap, berr := rpcbootstrap.New(ctx, log, bootstrapRPCURL)
		if berr != nil {
			return nil, fmt.Errorf("initialize execution bootstrap RPC: %w", berr)
		}

		if _, berr = bootstrap.CurrentStatus(ctx); berr != nil {
			return nil, fmt.Errorf("validate execution bootstrap RPC: %w", berr)
		}

		opts = append(opts,
			mimicry.WithStatusProvider(bootstrap.Status),
			mimicry.WithHeaderProvider(bootstrap.Headers),
			mimicry.WithBodyProvider(bootstrap.Bodies),
			mimicry.WithReceiptProvider(bootstrap.Receipts),
		)
	}

	client, err := mimicry.New(ctx, log, nodeRecord, "xatu", opts...)
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

	p.client.OnStatus(ctx, func(ctx context.Context, status mimicry.Status) error {
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
			s.NetworkId = status.GetNetworkID()
			s.Head = status.GetHead()
			s.Genesis = status.GetGenesis()
			s.ForkId = &xatu.ExecutionNodeStatus_ForkID{
				Hash: status.GetForkIDHash(),
				Next: status.GetForkIDNext(),
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

		p.disconnectReason.Store(&str)

		response <- errors.New("disconnected from peer (reason " + str + ")")

		return nil
	})

	err := p.client.Start(ctx)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// DisconnectReason returns the devp2p disconnect reason sent by the remote
// peer, or an empty string if the session ended without one.
func (p *ExecutionPeer) DisconnectReason() string {
	if reason := p.disconnectReason.Load(); reason != nil {
		return *reason
	}

	return ""
}

func (p *ExecutionPeer) Stop(ctx context.Context) error {
	return p.client.Stop(ctx)
}
