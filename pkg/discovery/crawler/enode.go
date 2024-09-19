package crawler

import (
	"context"

	"github.com/ethereum/go-ethereum/p2p/enode"
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
	RegisterHandler(handler func(ctx context.Context, node *enode.Node, source string) error)
}
