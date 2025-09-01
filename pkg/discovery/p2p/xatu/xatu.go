package xatu

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/discovery"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

const Type = "xatu"

type Coordinator struct {
	config *Config

	discV4  *discovery.DiscV4
	discV5  *discovery.DiscV5
	handler func(ctx context.Context, node *enode.Node, source string) error

	log logrus.FieldLogger

	conn *grpc.ClientConn
	pb   xatu.CoordinatorClient

	scheduler gocron.Scheduler
}

func New(config *Config, handler func(ctx context.Context, node *enode.Node, source string) error, log logrus.FieldLogger) (*Coordinator, error) {
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

	return &Coordinator{
		config:  config,
		log:     log,
		handler: handler,
		conn:    conn,
		pb:      pbClient,
	}, nil
}

func (c *Coordinator) Type() string {
	return Type
}

func (c *Coordinator) Start(ctx context.Context) error {
	if err := c.startCrons(ctx); err != nil {
		return err
	}

	if c.config.DiscV4 {
		c.discV4 = discovery.NewDiscV4(ctx, c.config.Restart, c.log)

		c.discV4.OnNodeRecord(ctx, func(ctx context.Context, node *enode.Node) error {
			return c.handler(ctx, node, "discV4")
		})
	}

	if c.config.DiscV5 {
		c.discV5 = discovery.NewDiscV5(ctx, c.config.Restart, c.log)

		c.discV5.OnNodeRecord(ctx, func(ctx context.Context, node *enode.Node) error {
			return c.handler(ctx, node, "discV5")
		})
	}

	return nil
}

func (c *Coordinator) Stop(ctx context.Context) error {
	if c.scheduler != nil {
		if err := c.scheduler.Shutdown(); err != nil {
			c.log.WithError(err).Error("Failed to shutdown scheduler")
		}
	}

	if c.config.DiscV4 {
		if err := c.discV4.Stop(ctx); err != nil {
			return err
		}
	}

	if c.config.DiscV5 {
		if err := c.discV5.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Coordinator) startCrons(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	c.scheduler = scheduler

	if _, err := c.scheduler.NewJob(
		gocron.DurationJob(c.config.Restart),
		gocron.NewTask(
			func(ctx context.Context) {
				var bootNodes []string

				// Fetch execution boot node
				forkIDHashes := make([][]byte, len(c.config.ForkIDHashes))
				for i, forkIDHash := range c.config.ForkIDHashes {
					forkIDHashBytes, err := hex.DecodeString(forkIDHash[2:])
					if err == nil {
						forkIDHashes[i] = forkIDHashBytes
					}
				}

				execReq := xatu.GetDiscoveryExecutionNodeRecordRequest{
					NetworkIds:   c.config.NetworkIds,
					ForkIdHashes: forkIDHashes,
				}

				md := metadata.New(c.config.Headers)
				ctx = metadata.NewOutgoingContext(ctx, md)

				execRes, err := c.pb.GetDiscoveryExecutionNodeRecord(ctx, &execReq, grpc.UseCompressor(gzip.Name))
				if err != nil {
					c.log.WithError(err).Error("Failed to get execution discovery node record")
				} else {
					bootNodes = append(bootNodes, execRes.NodeRecord)
				}

				// Fetch consensus boot node
				forkDigests := make([][]byte, len(c.config.ForkDigests))
				for i, forkDigest := range c.config.ForkDigests {
					forkDigestBytes, err := hex.DecodeString(forkDigest[2:])
					if err == nil {
						forkDigests[i] = forkDigestBytes
					}
				}

				consReq := xatu.GetDiscoveryConsensusNodeRecordRequest{
					NetworkIds:  c.config.NetworkIds,
					ForkDigests: forkDigests,
				}

				consRes, err := c.pb.GetDiscoveryConsensusNodeRecord(ctx, &consReq, grpc.UseCompressor(gzip.Name))
				if err != nil {
					c.log.WithError(err).Error("Failed to get consensus discovery node record")
				} else {
					bootNodes = append(bootNodes, consRes.NodeRecord)
				}

				// Update boot nodes if we have any
				if len(bootNodes) > 0 {
					if err = c.discV5.UpdateBootNodes(bootNodes); err != nil {
						c.log.WithError(err).Error("Failed to update discV5 boot nodes")

						return
					}

					if err := c.discV5.Start(ctx); err != nil {
						c.log.WithError(err).Error("Failed to start discV5")

						return
					}
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	c.scheduler.Start()

	return nil
}

func (c *Coordinator) GetNetworkIds() []uint64 {
	return c.config.NetworkIds
}

func (c *Coordinator) GetForkIdHashes() []string {
	return c.config.ForkIDHashes
}

func (c *Coordinator) GetForkDigests() []string {
	return c.config.ForkDigests
}
