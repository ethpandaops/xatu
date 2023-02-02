package p2p

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
)

type Type string

const (
	TypeUnknown Type = "unknown"
	TypeStatic  Type = static.Type
	TypeXatu    Type = xatu.Type
)

type P2P interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
}
