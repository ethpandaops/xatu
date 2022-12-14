package handler

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Peer struct {
	CreateNewClientMeta func(ctx context.Context) (*xatu.ClientMeta, error)
	DecoratedEvent      func(ctx context.Context, event *xatu.DecoratedEvent) error
}
