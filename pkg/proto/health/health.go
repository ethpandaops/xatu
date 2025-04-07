package health

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the Health service for gRPC health checks.
type Server struct {
	UnimplementedHealthServer

	mu             sync.RWMutex
	statusMap      map[string]HealthCheckResponse_ServingStatus
	watchersMu     sync.RWMutex
	watchers       map[string][]*watcherState
	shutdownSignal chan struct{}
	serviceName    string
}

type watcherState struct {
	service string
	stream  grpc.ServerStreamingServer[HealthCheckResponse]
	done    chan struct{}
}

// NewServer creates a new health server with default status SERVING for empty service name.
// If serviceName is provided, it will be used as a prefix for component status names.
func NewServer(serviceName string) *Server {
	server := &Server{
		statusMap:      make(map[string]HealthCheckResponse_ServingStatus),
		watchers:       make(map[string][]*watcherState),
		shutdownSignal: make(chan struct{}),
		serviceName:    serviceName,
	}

	// Default overall health is SERVING
	server.statusMap[""] = HealthCheckResponse_SERVING

	return server
}

// RegisterWith registers the health server with a gRPC server.
func (s *Server) RegisterWith(grpcServer *grpc.Server) {
	RegisterHealthServer(grpcServer, s)
}

// ServiceName returns the base service name for this health server
func (s *Server) ServiceName() string {
	return s.serviceName
}

// SetServiceName changes the base service name for this health server
func (s *Server) SetServiceName(name string) {
	s.serviceName = name
}

// GetComponentName returns a fully qualified component name based on the base service name
func (s *Server) GetComponentName(component string) string {
	if s.serviceName == "" {
		return component
	}
	return fmt.Sprintf("%s.%s", s.serviceName, component)
}

// SetComponentStatus updates the serving status for a specific component
// using the base service name as a prefix
func (s *Server) SetComponentStatus(component string, status HealthCheckResponse_ServingStatus) {
	componentName := s.GetComponentName(component)
	s.SetServingStatus(componentName, status)
}

// SetAllComponentStatus sets the serving status for all components
// This does not affect non-component status entries
func (s *Server) SetAllComponentStatus(status HealthCheckResponse_ServingStatus) {
	// If no service name is set, we can't distinguish components from other statuses,
	// so fall back to setting all statuses
	if s.serviceName == "" {
		s.SetAllServingStatus(status)
		return
	}

	s.mu.RLock()
	// Find services that start with our service name prefix
	componentsPrefix := s.serviceName + "."
	services := make([]string, 0)
	for service := range s.statusMap {
		if len(service) > len(componentsPrefix) && service[:len(componentsPrefix)] == componentsPrefix {
			services = append(services, service)
		}
	}
	s.mu.RUnlock()

	// Update each component status
	for _, service := range services {
		s.SetServingStatus(service, status)
	}
}

// SetServingStatus sets the serving status of a service.
func (s *Server) SetServingStatus(service string, status HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	s.statusMap[service] = status
	s.mu.Unlock()

	// Notify watchers of this service about the status change
	s.watchersMu.RLock()
	watchers := s.watchers[service]
	s.watchersMu.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.done:
			// Watcher is done
		default:
			if err := w.stream.Send(&HealthCheckResponse{Status: status}); err != nil {
				// Failed to send to this watcher, but continue for others
				close(w.done)
			}
		}
	}
}

// SetAllServingStatus sets the serving status for all registered services.
func (s *Server) SetAllServingStatus(status HealthCheckResponse_ServingStatus) {
	s.mu.RLock()
	services := make([]string, 0, len(s.statusMap))
	for service := range s.statusMap {
		services = append(services, service)
	}
	s.mu.RUnlock()

	for _, service := range services {
		s.SetServingStatus(service, status)
	}
}

// ClearStatus removes the status of the given service.
func (s *Server) ClearStatus(service string) {
	s.mu.Lock()
	delete(s.statusMap, service)
	s.mu.Unlock()
}

// Check implements the Check method from the Health service.
func (s *Server) Check(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	s.mu.RLock()
	serviceStatus, ok := s.statusMap[req.Service]
	s.mu.RUnlock()

	if !ok {
		return &HealthCheckResponse{
			Status: HealthCheckResponse_SERVICE_UNKNOWN,
		}, nil
	}

	return &HealthCheckResponse{
		Status: serviceStatus,
	}, nil
}

// Watch implements the Watch method from the Health service.
func (s *Server) Watch(req *HealthCheckRequest, stream grpc.ServerStreamingServer[HealthCheckResponse]) error {
	watcher := &watcherState{
		service: req.Service,
		stream:  stream,
		done:    make(chan struct{}),
	}

	defer close(watcher.done)

	// First check current status and send it immediately
	s.mu.RLock()
	serviceStatus, ok := s.statusMap[req.Service]
	s.mu.RUnlock()

	if !ok {
		serviceStatus = HealthCheckResponse_SERVICE_UNKNOWN
	}

	if err := stream.Send(&HealthCheckResponse{Status: serviceStatus}); err != nil {
		return status.Error(codes.Canceled, "failed to send initial health status")
	}

	// Register this watcher
	s.watchersMu.Lock()
	if _, ok := s.watchers[req.Service]; !ok {
		s.watchers[req.Service] = make([]*watcherState, 0)
	}
	s.watchers[req.Service] = append(s.watchers[req.Service], watcher)
	s.watchersMu.Unlock()

	// Keep the stream open until context is canceled or server shuts down
	select {
	case <-stream.Context().Done():
		return status.Error(codes.Canceled, "client canceled the request")
	case <-s.shutdownSignal:
		return status.Error(codes.Unavailable, "server is shutting down")
	}
}

// Shutdown signals all watchers that the server is shutting down.
func (s *Server) Shutdown() {
	close(s.shutdownSignal)
}

// RegisterGrpcHealthCheckService is a convenience function to register the health service with a gRPC server.
func RegisterGrpcHealthCheckService(grpcServer *grpc.Server, serviceName string) *Server {
	healthServer := NewServer(serviceName)
	RegisterHealthServer(grpcServer, healthServer)
	return healthServer
}
