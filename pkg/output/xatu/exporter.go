package xatu

import (
	"context"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EventExporter struct {
	config *Config
	log    logrus.FieldLogger

	client pb.XatuClient
	conn   *grpc.ClientConn
}

func NewEventExporter(config *Config, log logrus.FieldLogger) (EventExporter, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(*&config.Address, opts...)
	if err != nil {
		return EventExporter{}, fmt.Errorf("fail to dial: %v", err)
	}

	return EventExporter{
		config: config,
		log:    log,
		conn:   conn,
		client: pb.NewXatuClient(conn),
	}, nil
}

func (e EventExporter) ExportEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	e.log.WithField("events", len(events)).Info("Sending batch of events to Xatu sink")

	if err := e.sendUpstream(ctx, events); err != nil {
		return err
	}

	return nil
}

func (e EventExporter) Shutdown(ctx context.Context) error {
	return e.conn.Close()
}

func (e *EventExporter) sendUpstream(ctx context.Context, events []*xatu.DecoratedEvent) error {
	req := &pb.CreateEventsRequest{
		Events: events,
	}

	rsp, err := e.client.CreateEvents(ctx, req)
	if err != nil {
		return err
	}

	e.log.WithField("response", rsp).Info("Received response from Xatu sink")

	return nil
}
