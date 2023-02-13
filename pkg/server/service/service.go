package service

import (
	"context"

	"github.com/creasty/defaults"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/service/coordinator"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
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

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, cfg *Config, c store.Cache, g geoip.Provider) ([]GRPCService, error) {
	services := []GRPCService{}

	if cfg.EventIngester.Enabled {
		if err := defaults.Set(&cfg.EventIngester); err != nil {
			return nil, err
		}

		service, err := eventingester.New(ctx, log, &cfg.EventIngester, c, g)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	if cfg.Coordinator.Enabled {
		if err := defaults.Set(&cfg.Coordinator); err != nil {
			return nil, err
		}

		service, err := coordinator.New(ctx, log, &cfg.Coordinator, c, g)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}
