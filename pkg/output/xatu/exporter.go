package xatu

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger

	client pb.EventIngesterClient
	conn   *grpc.ClientConn
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
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
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),
		conn:   conn,
		client: pb.NewEventIngesterClient(conn),
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

	var rsp *pb.CreateEventsResponse

	var err error

	for attempt := 0; attempt < e.config.Retry.MaxAttempts; attempt++ {
		rsp, err = e.client.CreateEvents(ctx, req, grpc.UseCompressor(gzip.Name))
		if err != nil {
			st, ok := status.FromError(err)

			if ok && st.Code() == grpcCodes.Unknown || st.Code() == grpcCodes.Unavailable {
				logCtx.
					WithField("attempt", attempt+1).
					WithField("max_attempts", e.config.Retry.MaxAttempts).
					WithField("code", st.Code()).
					Warn("Transient error occurred when exporting items, retrying...")

				time.Sleep(e.config.Retry.Interval)

				continue
			}

			return err
		}

		break
	}

	logCtx.WithField("response", rsp).Debug("Received response from Xatu sink")

	return nil
}
