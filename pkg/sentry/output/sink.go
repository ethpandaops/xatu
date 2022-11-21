package output

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/output/http"
	"github.com/ethpandaops/xatu/pkg/sentry/output/stdout"
)

type SinkType string

const (
	SinkTypeUnknown SinkType = "unknown"
	SinkTypeHTTP    SinkType = http.SinkType
	SinkTypeStdOut  SinkType = stdout.SinkType
)

type Sink interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error
}
