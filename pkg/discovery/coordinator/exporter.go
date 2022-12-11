package coordinator

import (
	"context"
	"fmt"

	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger

	client pb.CoordinatorClient
	conn   *grpc.ClientConn
}

func NewItemExporter(config *Config, log logrus.FieldLogger) (ItemExporter, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

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
	e.log.WithField("records", len(items)).Info("Sending batch of records to coordinator")

	if err := e.sendUpstream(ctx, items); err != nil {
		return err
	}

	return nil
}

func (e ItemExporter) Shutdown(ctx context.Context) error {
	return e.conn.Close()
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*string) error {
	records := make([]string, len(items))

	for i, item := range items {
		records[i] = *item
	}

	req := &pb.CreateNodeRecordsRequest{
		NodeRecords: records,
	}

	rsp, err := e.client.CreateNodeRecords(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	e.log.WithField("response", rsp).Info("Received response from Xatu sink")

	return nil
}
