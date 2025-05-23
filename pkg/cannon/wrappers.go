package cannon

import (
	"context"
	"errors"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/beevik/ntp"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/go-co-op/gocron/v2"
)

// DefaultBeaconNodeWrapper wraps the production BeaconNode to implement our interface
type DefaultBeaconNodeWrapper struct {
	beacon *ethereum.BeaconNode
}

func (w *DefaultBeaconNodeWrapper) Start(ctx context.Context) error {
	return w.beacon.Start(ctx)
}

func (w *DefaultBeaconNodeWrapper) GetBeaconBlock(ctx context.Context, identifier string, ignoreMetrics ...bool) (*spec.VersionedSignedBeaconBlock, error) {
	return w.beacon.GetBeaconBlock(ctx, identifier, ignoreMetrics...)
}

func (w *DefaultBeaconNodeWrapper) GetValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	return w.beacon.GetValidators(ctx, identifier)
}

func (w *DefaultBeaconNodeWrapper) Node() beacon.Node {
	return w.beacon.Node()
}

func (w *DefaultBeaconNodeWrapper) Metadata() *services.MetadataService {
	return w.beacon.Metadata()
}

func (w *DefaultBeaconNodeWrapper) Duties() *services.DutiesService {
	return w.beacon.Duties()
}

func (w *DefaultBeaconNodeWrapper) OnReady(ctx context.Context, callback func(ctx context.Context) error) {
	w.beacon.OnReady(ctx, callback)
}

func (w *DefaultBeaconNodeWrapper) Synced(ctx context.Context) error {
	return w.beacon.Synced(ctx)
}

// DefaultCoordinatorWrapper wraps the production Coordinator to implement our interface
type DefaultCoordinatorWrapper struct {
	client *coordinator.Client
}

func (w *DefaultCoordinatorWrapper) GetCannonLocation(ctx context.Context, typ xatu.CannonType, networkID string) (*xatu.CannonLocation, error) {
	return w.client.GetCannonLocation(ctx, typ, networkID)
}

func (w *DefaultCoordinatorWrapper) UpsertCannonLocationRequest(ctx context.Context, location *xatu.CannonLocation) error {
	return w.client.UpsertCannonLocationRequest(ctx, location)
}

func (w *DefaultCoordinatorWrapper) Start(ctx context.Context) error {
	return nil // Coordinator doesn't have a Start method
}

func (w *DefaultCoordinatorWrapper) Stop(ctx context.Context) error {
	return nil // Coordinator doesn't have a Stop method
}

// DefaultScheduler wraps gocron.Scheduler to implement our interface
type DefaultScheduler struct {
	scheduler gocron.Scheduler
}

func (s *DefaultScheduler) Start() {
	s.scheduler.Start()
}

func (s *DefaultScheduler) Shutdown() error {
	return s.scheduler.Shutdown()
}

func (s *DefaultScheduler) NewJob(jobDefinition any, task any, options ...any) (any, error) {
	// This is a simplified implementation - would need more complex type handling for production
	return nil, errors.New("not implemented")
}

// DefaultTimeProvider provides real time operations
type DefaultTimeProvider struct{}

func (tp *DefaultTimeProvider) Now() time.Time {
	return time.Now()
}

func (tp *DefaultTimeProvider) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (tp *DefaultTimeProvider) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (tp *DefaultTimeProvider) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (tp *DefaultTimeProvider) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// DefaultNTPClient provides real NTP operations
type DefaultNTPClient struct{}

func (n *DefaultNTPClient) Query(host string) (NTPResponse, error) {
	response, err := ntp.Query(host)
	if err != nil {
		return nil, err
	}
	return &DefaultNTPResponse{response: response}, nil
}

// DefaultNTPResponse wraps ntp.Response to implement our interface
type DefaultNTPResponse struct {
	response *ntp.Response
}

func (r *DefaultNTPResponse) Validate() error {
	return r.response.Validate()
}

func (r *DefaultNTPResponse) ClockOffset() time.Duration {
	return r.response.ClockOffset
}

// Common errors
var (
	ErrConfigRequired = errors.New("config is required")
	ErrNotImplemented = errors.New("not implemented")
)