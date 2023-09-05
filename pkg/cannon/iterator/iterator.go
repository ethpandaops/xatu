package iterator

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Iterator interface {
	UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error
	Next(ctx context.Context) (xatu.CannonLocation, error)
}

var (
	ErrLocationUpToDate = errors.New("location up to date")
)
