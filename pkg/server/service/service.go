package service

import (
	"context"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/service/coordinator"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// GRPCService is a service that implements a single gRPC service as defined in
// our Protobuf definition.
type GRPCService interface {
	Start(ctx context.Context, server *grpc.Server) error
	Stop(ctx context.Context) error
	Name() string
}

type Type string

const (
	ServiceTypeUnknown       Type = "unknown"
	ServiceTypeEventIngester Type = eventingester.ServiceType
	ServiceTypeCoordinator   Type = coordinator.ServiceType
)

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, cfg *Config, clockDrift *time.Duration, p *persistence.Client, c store.Cache, g geoip.Provider, healthServer *health.Server) ([]GRPCService, error) {
	services := []GRPCService{}

	if cfg.EventIngester.Enabled {
		if err := defaults.Set(&cfg.EventIngester); err != nil {
			return nil, err
		}

		service, err := eventingester.NewIngester(ctx, log, &cfg.EventIngester, clockDrift, g, c, healthServer)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	if cfg.Coordinator.Enabled {
		if err := defaults.Set(&cfg.Coordinator); err != nil {
			return nil, err
		}

		service, err := coordinator.NewClient(ctx, log, &cfg.Coordinator, p, g, healthServer)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}
