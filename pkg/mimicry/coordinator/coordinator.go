package coordinator

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/static"
	xatuCoordinator "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu"
)

type Type string

const (
	TypeUnknown Type = "unknown"
	TypeStatic  Type = static.Type
	TypeXatu    Type = xatuCoordinator.Type
)

type Coordinator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
}
