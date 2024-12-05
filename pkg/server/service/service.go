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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// GRPCService is a service that implements a single gRPC service as defined in
// our Protobuf definition.
type GRPCService interface {
	Start(ctx context.Context, server *grpc.Server) error
	Stop(ctx context.Context) error
}

type Type string

const (
	ServiceTypeUnknown       Type = "unknown"
	ServiceTypeEventIngester Type = eventingester.ServiceType
	ServiceTypeCoordinator   Type = coordinator.ServiceType
)

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, cfg *Config, clockDrift *time.Duration, p *persistence.Client, c store.Cache, g geoip.Provider) ([]GRPCService, error) {
	services := []GRPCService{}

	if cfg.EventIngester.Enabled {
		if err := defaults.Set(&cfg.EventIngester); err != nil {
			return nil, err
		}

		service, err := eventingester.NewIngester(ctx, log, &cfg.EventIngester, clockDrift, g, c)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	if cfg.Coordinator.Enabled {
		if err := defaults.Set(&cfg.Coordinator); err != nil {
			return nil, err
		}

		service, err := coordinator.NewClient(ctx, log, &cfg.Coordinator, p, g)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

// Add this new function to coordinate shutdown across all services.
func ShutdownServices(ctx context.Context, services []GRPCService) error {
	// Create a timeout context for the entire shutdown sequence.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Use errgroup to handle multiple service shutdowns.
	g, ctx := errgroup.WithContext(ctx)

	for _, service := range services {
		svc := service

		g.Go(func() error {
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			return svc.Stop(shutdownCtx)
		})
	}

	return g.Wait()
}
