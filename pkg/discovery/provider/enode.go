package provider

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enode"
	disco "github.com/ethpandaops/ethcore/pkg/discovery"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/ethpandaops/xatu/pkg/discovery/shared/static"
)

type EnodeProviderType string

const (
	EnodeProviderTypeUnknown EnodeProviderType = "unknown"
	EnodeProviderTypeStatic  EnodeProviderType = static.Type
	EnodeProviderTypeXatu    EnodeProviderType = xatu.Type
)

type EnodeProvider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	RegisterHandler(ctx context.Context, handler func(ctx context.Context, node *enode.Node, source string) error)
}

type WrappedNodeFinder struct {
	provider EnodeProvider
}

func NewWrappedNodeFinder(provider EnodeProvider) disco.NodeFinder {
	return &WrappedNodeFinder{provider: provider}
}

func (w *WrappedNodeFinder) Start(ctx context.Context) error {
	return w.provider.Start(ctx)
}

func (w *WrappedNodeFinder) Stop(ctx context.Context) error {
	return w.provider.Stop(ctx)
}

func (w *WrappedNodeFinder) OnNodeRecord(ctx context.Context, handler func(ctx context.Context, node *enode.Node) error) {
	w.provider.RegisterHandler(ctx, func(ctx context.Context, node *enode.Node, source string) error {
		return handler(ctx, node)
	})
}
