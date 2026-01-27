package coordinator

import (
	"context"
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

// Client is a gRPC client for the coordinator service.
type Client struct {
	config *Config
	log    logrus.FieldLogger

	conn *grpc.ClientConn
	pb   xatu.CoordinatorClient
}

// New creates a new coordinator client.
func New(config *Config, log logrus.FieldLogger) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	var opts []grpc.DialOption

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return nil, fmt.Errorf("fail to get host from address: %w", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("fail to create client: %w", err)
	}

	pbClient := xatu.NewCoordinatorClient(conn)

	return &Client{
		config: config,
		log:    log.WithField("component", "coordinator"),
		conn:   conn,
		pb:     pbClient,
	}, nil
}

// Start starts the coordinator client.
func (c *Client) Start(ctx context.Context) error {
	return nil
}

// Stop stops the coordinator client and closes the connection.
func (c *Client) Stop(ctx context.Context) error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

// GetHorizonLocation retrieves the horizon location for a given type and network.
func (c *Client) GetHorizonLocation(
	ctx context.Context,
	typ xatu.HorizonType,
	networkID string,
) (*xatu.HorizonLocation, error) {
	req := xatu.GetHorizonLocationRequest{
		Type:      typ,
		NetworkId: networkID,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	res, err := c.pb.GetHorizonLocation(ctx, &req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return nil, err
	}

	return res.Location, nil
}

// UpsertHorizonLocation creates or updates a horizon location.
func (c *Client) UpsertHorizonLocation(ctx context.Context, location *xatu.HorizonLocation) error {
	req := xatu.UpsertHorizonLocationRequest{
		Location: location,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.pb.UpsertHorizonLocation(ctx, &req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	return nil
}
