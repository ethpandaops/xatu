package xatu

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

type ItemExporter struct {
	config  *Config
	log     logrus.FieldLogger
	headers map[string]string

	client pb.EventIngesterClient
	conn   *grpc.ClientConn
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	log = log.WithField("output_name", name).WithField("output_type", SinkType)

	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor, retry.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor, retry.StreamClientInterceptor()),
	}

	if config.KeepAlive.Enabled != nil && *config.KeepAlive.Enabled {
		log.
			WithField("keepalive_time", config.KeepAlive.Time).
			WithField("keepalive_timeout", config.KeepAlive.Timeout).
			Info("Enabling keepalive")

		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAlive.Time,
			Timeout:             config.KeepAlive.Timeout,
			PermitWithoutStream: true,
		}))
	} else {
		log.Info("Disabling keepalive")
	}

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return ItemExporter{}, fmt.Errorf("fail to get host from address: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return ItemExporter{}, fmt.Errorf("fail to dial: %v", err)
	}

	return ItemExporter{
		config:  config,
		headers: config.Headers,
		log:     log,
		conn:    conn,
		client:  pb.NewEventIngesterClient(conn),
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*pb.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx, "XatuItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to xatu sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func (e ItemExporter) Shutdown(ctx context.Context) error {
	return e.conn.Close()
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*pb.DecoratedEvent) error {
	req := &pb.CreateEventsRequest{
		Events: items,
	}

	logCtx := e.log.WithField("num_events", len(items))

	md := metadata.New(e.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	for key, value := range e.headers {
		md.Set(key, value)
	}

	var rsp *pb.CreateEventsResponse

	var err error

	opts := []grpc.CallOption{
		grpc.UseCompressor(gzip.Name),
	}

	startTime := time.Now()

	if e.config.Retry.Enabled {
		opts = append(opts,
			retry.WithOnRetryCallback(func(ctx context.Context, attempt uint, err error) {
				duration := time.Since(startTime)

				logCtx.
					WithField("attempt", attempt).
					WithError(err).
					WithField("duration", duration).
					Warn("Failed to export events. Retrying...")

				// Reset the startTime to the current time
				startTime = time.Now()
			}),
			retry.WithMax(uint(e.config.Retry.MaxAttempts)),
			retry.WithBackoff(retry.BackoffExponential(e.config.Retry.Scalar)),
			retry.WithCodes(
				grpc_codes.Unavailable,
				grpc_codes.Internal,
				grpc_codes.ResourceExhausted,
				grpc_codes.Unknown,
				grpc_codes.Canceled,
			),
		)
	}

	rsp, err = e.client.CreateEvents(ctx, req, opts...)
	if err != nil {
		return err
	}

	logCtx.WithField("response", rsp).Debug("Received response from Xatu sink")

	return nil
}
