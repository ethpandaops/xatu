// Package contributoor provides a gRPC service that returns configuration data
// for beacon subscriptions and user/network settings. It supports Basic Auth
// authentication and filters user-specific configuration based on the authenticated user.
package contributoor

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/service/contributoor/auth"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client implements the contributoor gRPC service that provides configuration data
// for beacon subscriptions and attestation settings. It supports user authentication
// and returns user-specific configuration filtered by the authenticated user.
type Client struct {
	xatu.UnimplementedContributoorServer

	log          logrus.FieldLogger
	config       *Config
	auth         *auth.Authorization
	healthServer *health.Server
}

// NewClient creates a new contributoor service client with the provided configuration.
// It initializes authentication if enabled and returns a configured client ready for service.
func NewClient(
	_ context.Context,
	log logrus.FieldLogger,
	conf *Config,
	healthServer *health.Server,
) (*Client, error) {
	log = log.WithField("server/module", ServiceType)

	authorization, err := auth.NewAuthorization(log, conf.Authorization)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	return &Client{
		log:          log,
		config:       conf,
		auth:         authorization,
		healthServer: healthServer,
	}, nil
}

// Start initializes and registers the contributoor service with the gRPC server.
// It starts authentication services and registers the service for health checks.
func (c *Client) Start(ctx context.Context, grpcServer *grpc.Server) error {
	c.log.Info("Starting contributoor service")

	if err := c.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

	xatu.RegisterContributoorServer(grpcServer, c)

	c.healthServer.SetServingStatus(c.Name(), grpc_health_v1.HealthCheckResponse_SERVING)

	return nil
}

// Stop gracefully shuts down the contributoor service and updates health status.
func (c *Client) Stop(ctx context.Context) error {
	c.log.Info("Stopping contributoor service")

	c.healthServer.SetServingStatus(c.Name(), grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	return nil
}

// Name returns the service type identifier for this contributoor service.
func (c *Client) Name() string {
	return ServiceType
}

// GetConfiguration returns the contributoor configuration data filtered by the authenticated user.
// If authentication is enabled, only the authenticated user's configuration is returned in the user section.
// Global and network configurations are always included for all authenticated users.
func (c *Client) GetConfiguration(
	ctx context.Context,
	_ *xatu.GetConfigurationRequest,
) (*xatu.GetConfigurationResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get metadata from context")
	}

	var authenticatedUsername string

	if c.config.Authorization.Enabled {
		authorization := md.Get("authorization")

		if len(authorization) == 0 {
			return nil, status.Error(codes.Unauthenticated, "no authorization header provided")
		}

		username, err := c.auth.IsAuthorized(authorization[0])
		if err != nil {
			c.log.WithError(err).Warn("Failed to authorize user")

			return nil, status.Error(codes.Unauthenticated, "failed to authorize user")
		}

		if username == "" {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		authenticatedUsername = username
		c.log.WithField("user", username).Debug("Authenticated user")
	}

	configData := &c.config.BootConfiguration

	if len(configData.Global.BeaconSubscriptions.Topics) == 0 {
		c.log.Debug("Using default config because parsed config is empty")

		configData = DefaultBootConfiguration()
	} else {
		c.log.Debug("Using parsed config from YAML")
	}

	c.log.Debug("Successfully generated configuration response")

	return &xatu.GetConfigurationResponse{
		Version:       c.config.Version,
		UpdatedAt:     timestamppb.New(time.Now()),
		Configuration: configData.ToProtoForUser(authenticatedUsername),
	}, nil
}
