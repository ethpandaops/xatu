package coordinator

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"

	"github.com/ethpandaops/xatu/pkg/observability"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type ItemExporter struct {
	config *Config
	log    observability.ContextualLogger

	conn   *grpc.ClientConn
	client pb.CoordinatorClient
}

func NewItemExporter(config *Config, log observability.ContextualLogger) (ItemExporter, error) {
	opts := []grpc.DialOption{
		observability.GRPCClientOption(),
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
		log:    log,
		conn:   conn,
		client: pb.NewCoordinatorClient(conn),
	}, nil
}

func (e ItemExporter) ExportItems(ctx context.Context, items []*string) error {
	e.log.WithField("records", len(items)).WithContext(ctx).Debug("Sending batch of records to coordinator")

	if err := e.sendUpstream(ctx, items); err != nil {
		return err
	}

	return nil
}

func (e ItemExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*string) error {
	records := make([]string, len(items))

	for i, item := range items {
		records[i] = *item
	}

	req := &pb.CreateNodeRecordsRequest{
		NodeRecords: records,
	}

	md := metadata.New(e.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	rsp, err := e.client.CreateNodeRecords(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	e.log.WithField("response", rsp).WithContext(ctx).Debug("Received response from Xatu sink")

	return nil
}
