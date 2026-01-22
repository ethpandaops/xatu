package coordinator

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	metrics *Metrics
}

func NewCoordinator(ctx context.Context, name string, config *Config, log logrus.FieldLogger) (*Coordinator, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	// If networkConfig is provided, fetch and apply the devnet configuration
	if config.NetworkConfig != nil && config.NetworkConfig.URL != "" {
		log.WithField("url", config.NetworkConfig.URL).Info("Fetching network configuration from URL")

		fetched, err := config.NetworkConfig.Fetch(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch network config: %w", err)
		}

		// Apply fetched values to config (overrides any manual config)
		config.NetworkIds = []uint64{fetched.ChainID}
		config.ForkIDHashes = []string{fetched.ForkIDHashHex()}
		config.Enodes = fetched.Enodes

		log.WithFields(logrus.Fields{
			"chain_id":     fetched.ChainID,
			"fork_id_hash": fetched.ForkIDHashHex(),
			"boot_nodes":   len(fetched.BootNodes),
			"enodes":       len(fetched.Enodes),
		}).Info("Applied network configuration from URL")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	var opts []grpc.DialOption

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return nil, fmt.Errorf("fail to get host from address: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial: %v", err)
	}

	pbClient := xatu.NewCoordinatorClient(conn)

	return &Coordinator{
		name:    name,
		config:  config,
		log:     log,
		conn:    conn,
		pb:      pbClient,
		metrics: NewMetrics("xatu_mimicry_coordinator_xatu"),
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
		NetworkIds:   c.config.NetworkIds,
		ForkIdHashes: forkIDHashes,
		Capabilities: c.config.Capabilities,
		NodeRecords:  records,
		Limit:        c.config.MaxPeers,
		ClientId:     c.name,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	res, err := c.pb.CoordinateExecutionNodeRecords(ctx, &req, grpc.UseCompressor(gzip.Name))

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Coordinator) HandleExecutionNodeRecordStatus(ctx context.Context, status *xatu.ExecutionNodeStatus) error {
	c.log.WithField("record", status.NodeRecord).Debug("found execution node status, sending to coordinator")

	req := xatu.CreateExecutionNodeRecordStatusRequest{
		Status: status,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.pb.CreateExecutionNodeRecordStatus(ctx, &req, grpc.UseCompressor(gzip.Name))

	if err == nil {
		c.metrics.AddNodeRecordStatus(1, fmt.Sprintf("%d", status.GetNetworkId()), fmt.Sprintf("0x%x", status.GetForkId().GetHash()))
	}

	// TODO: create and send status event for clickhouse

	return err
}

// GetEnodes returns the execution layer enodes from the network config.
func (c *Coordinator) GetEnodes() []string {
	return c.config.Enodes
}
