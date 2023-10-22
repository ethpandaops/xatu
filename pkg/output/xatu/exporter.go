package xatu

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/ethpandaops/xatu/pkg/observability"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger

	clients     []pb.EventIngesterClient
	conns       []*grpc.ClientConn
	mu          sync.Mutex
	currentConn int
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	var opts []grpc.DialOption

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return ItemExporter{}, fmt.Errorf("fail to get host from address: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	var conns []*grpc.ClientConn

	var clients []pb.EventIngesterClient

	for i := 0; i < config.Connections; i++ {
		conn, err := grpc.Dial(config.Address, opts...)
		if err != nil {
			for _, c := range conns {
				c.Close()
			}

			return ItemExporter{}, fmt.Errorf("fail to dial: %v", err)
		}

		conns = append(conns, conn)
		clients = append(clients, pb.NewEventIngesterClient(conn))
	}

	return ItemExporter{
		config:  config,
		log:     log.WithField("output_name", name).WithField("output_type", SinkType),
		conns:   conns,
		clients: clients,
	}, nil
}

func (e *ItemExporter) getNextConn() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	conn := e.currentConn
	e.currentConn = (e.currentConn + 1) % len(e.conns)

	return conn
}

func (e *ItemExporter) ExportItems(ctx context.Context, items []*pb.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx, "XatuItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to xatu sink")

	connIndex := e.getNextConn()
	if err := e.sendUpstream(ctx, items, connIndex); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func (e *ItemExporter) Shutdown(ctx context.Context) error {
	for _, conn := range e.conns {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*pb.DecoratedEvent, connIndex int) error {
	req := &pb.CreateEventsRequest{
		Events: items,
	}

	md := metadata.New(e.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	rsp, err := e.clients[connIndex].CreateEvents(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	e.log.WithField("response", rsp).Debug("Received response from Xatu sink")

	return nil
}
