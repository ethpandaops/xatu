package service

import (
	"context"
	"fmt"

	"github.com/creasty/defaults"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GRPCService is a service that implements a single gRPC service as defined in
// our Protobuf definition.
type GRPCService interface {
	Start(server *grpc.Server) error
	Stop() error
}

type ServiceType string

const (
	ServiceTypeUnknown       ServiceType = "unknown"
	ServiceTypeEventIngester ServiceType = eventingester.ServiceType
)

func CreateGRPCServices(ctx context.Context, log logrus.FieldLogger, serviceConfigs []Config) ([]GRPCService, error) {
	services := make([]GRPCService, 0, len(serviceConfigs))

	for _, conf := range serviceConfigs {
		service, err := CreateGRPCService(ctx, conf.ServiceType, conf.Config, log)
		if err != nil {
			return nil, err
		}

		services = append(services, service)
	}

	return services, nil
}

func CreateGRPCService(ctx context.Context, name ServiceType, config *RawMessage, log logrus.FieldLogger) (GRPCService, error) {
	switch name {
	case eventingester.ServiceType:
		conf := &eventingester.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return eventingester.New(ctx, log, conf)
	default:
		return nil, fmt.Errorf("unknown service name: %s", name)
	}
}
