package output

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/sentry/output/http"
	"github.com/ethpandaops/xatu/pkg/xatu"
)

type SinkType string

const (
	SinkTypeUnknown SinkType = "unknown"
	SinkTypeHTTP    SinkType = http.SinkType
)

type Sink interface {
	Type() string
	HandleNewDecoratedEvent(ctx context.Context, event xatu.DecoratedEvent) error
}
