package output

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/output/http"
	"github.com/ethpandaops/xatu/pkg/output/kafka"
	"github.com/ethpandaops/xatu/pkg/output/stdout"
	xatuSink "github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

//go:generate mockgen -package mock -destination mock/sink.mock.go github.com/ethpandaops/xatu/pkg/output Sink

type SinkType string

const (
	SinkTypeUnknown SinkType = "unknown"
	SinkTypeHTTP    SinkType = http.SinkType
	SinkTypeStdOut  SinkType = stdout.SinkType
	SinkTypeXatu    SinkType = xatuSink.SinkType
	SinkTypeKafka   SinkType = kafka.SinkType
)

type Sink interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	Name() string
	HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error
	HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error
}
