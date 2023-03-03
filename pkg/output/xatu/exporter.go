package xatu

import (
	"context"
	"fmt"
	"net"

	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger

	client pb.EventIngesterClient
	conn   *grpc.ClientConn
}

func NewItemExporter(config *Config, log logrus.FieldLogger) (ItemExporter, error) {
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

	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return ItemExporter{}, fmt.Errorf("fail to dial: %v", err)
	}

	return ItemExporter{
		config: config,
		log:    log,
		conn:   conn,
		client: pb.NewEventIngesterClient(conn),
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*pb.DecoratedEvent) error {
	e.log.WithField("events", len(items)).Debug("Sending batch of events to Xatu sink")

	if err := e.sendUpstream(ctx, items); err != nil {
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

	md := metadata.New(e.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	rsp, err := e.client.CreateEvents(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	e.log.WithField("response", rsp).Debug("Received response from Xatu sink")

	return nil
}
