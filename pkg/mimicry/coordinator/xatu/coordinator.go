package xatu

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

type Coordinator struct {
	name   string
	config *Config
	log    logrus.FieldLogger

	conn *grpc.ClientConn
	pb   xatu.CoordinatorClient
}

func NewCoordinator(name string, config *Config, log logrus.FieldLogger) (*Coordinator, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial: %v", err)
	}

	pbClient := xatu.NewCoordinatorClient(conn)

	return &Coordinator{
		name:   name,
		config: config,
		log:    log,
		conn:   conn,
		pb:     pbClient,
	}, nil
}

func (c *Coordinator) Start(ctx context.Context) error {
	return nil
}

func (c *Coordinator) Stop(ctx context.Context) error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Coordinator) CoordinateExecutionNodeRecords(ctx context.Context, records []*xatu.CoordinatedNodeRecord) (*xatu.CoordinateExecutionNodeRecordsResponse, error) {
	forkIDHashes := make([][]byte, len(c.config.ForkIDHashes))

	for i, forkIDHash := range c.config.ForkIDHashes {
		forkIDHashBytes, err := hex.DecodeString(forkIDHash[2:])
		if err == nil {
			forkIDHashes[i] = forkIDHashBytes
		}
	}

	req := xatu.CoordinateExecutionNodeRecordsRequest{
		NetworkIds:   c.config.NetworkIDs,
		ForkIdHashes: forkIDHashes,
		NodeRecords:  records,
		Limit:        c.config.MaxPeers,
		ClientId:     c.name,
	}

	c.log.WithField("records", c.config.ForkIDHashes).Info("asdasdasd")

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	res, err := c.pb.CoordinateExecutionNodeRecords(ctx, &req, grpc.UseCompressor(gzip.Name))

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Coordinator) HandleExecutionNodeRecordStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) error {
	c.log.WithField("record", status.NodeRecord).Info("found execution node status, sending to coordinator")

	req := xatu.CreateExecutionNodeRecordStatusRequest{
		Status: status,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.pb.CreateExecutionNodeRecordStatus(ctx, &req, grpc.UseCompressor(gzip.Name))

	return err
}
