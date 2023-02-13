package geoip

import (
	"context"
	"net"

	"github.com/ethpandaops/xatu/pkg/server/geoip/lookup"
	"github.com/ethpandaops/xatu/pkg/server/geoip/maxmind"
)

type Type string

const (
	TypeUnknown Type = "unknown"
	TypeMaxmind Type = maxmind.Type
)

type Provider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	LookupIP(ctx context.Context, ip net.IP) (*lookup.Result, error)
}
