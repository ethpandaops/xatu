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

type Client struct {
	config *Config
	log    logrus.FieldLogger

	conn *grpc.ClientConn
	pb   xatu.CoordinatorClient
}

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

	return &Client{
		config: config,
		log:    log,
		conn:   conn,
		pb:     pbClient,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if err := c.conn.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Client) GetCannonLocation(ctx context.Context, typ xatu.CannonType, networkID string) (*xatu.CannonLocation, error) {
	req := xatu.GetCannonLocationRequest{
		Type:      typ,
		NetworkId: networkID,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	res, err := c.pb.GetCannonLocation(ctx, &req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return nil, err
	}

	return res.Location, nil
}

func (c *Client) UpsertCannonLocationRequest(ctx context.Context, location *xatu.CannonLocation) error {
	req := xatu.UpsertCannonLocationRequest{
		Location: location,
	}

	md := metadata.New(c.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := c.pb.UpsertCannonLocation(ctx, &req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	return nil
}
