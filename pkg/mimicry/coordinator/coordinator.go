package coordinator

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/mimicry/coordinator/manual"
	xatuCoordinator "github.com/ethpandaops/xatu/pkg/mimicry/coordinator/xatu"
)

type Type string

const (
	TypeUnknown Type = "unknown"
	TypeManual  Type = manual.Type
	TypeXatu    Type = xatuCoordinator.Type
)

type Coordinator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
}
