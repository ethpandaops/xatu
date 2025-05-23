package cannon

import (
	"context"
	"time"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// BeaconNode defines the interface for beacon node operations
type BeaconNode interface {
	// Core beacon node operations
	Start(ctx context.Context) error

	// Block operations
	GetBeaconBlock(ctx context.Context, identifier string, ignoreMetrics ...bool) (*spec.VersionedSignedBeaconBlock, error)

	// Validator operations
	GetValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*apiv1.Validator, error)

	// Node information
	Node() beacon.Node
	Metadata() *services.MetadataService
	Duties() *services.DutiesService

	// Event callbacks
	OnReady(ctx context.Context, callback func(ctx context.Context) error)

	// Synchronization status
	Synced(ctx context.Context) error
}

// Coordinator defines the interface for coordinator operations
type Coordinator interface {
	// Cannon location operations
	GetCannonLocation(ctx context.Context, typ xatu.CannonType, networkID string) (*xatu.CannonLocation, error)
	UpsertCannonLocationRequest(ctx context.Context, location *xatu.CannonLocation) error

	// Connection management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// BlockClassification represents a block classification from blockprint
type BlockClassification struct {
	Slot        uint64 `json:"slot"`
	Blockprint  string `json:"blockprint"`
	BlockHash   string `json:"blockHash"`
	BlockNumber uint64 `json:"blockNumber"`
}

// Blockprint defines the interface for blockprint operations
type Blockprint interface {
	// Block classification operations
	GetBlockClassifications(ctx context.Context, slots []uint64) (map[uint64]*BlockClassification, error)

	// Health check
	HealthCheck(ctx context.Context) error
}

// Scheduler defines the interface for job scheduling
type Scheduler interface {
	Start()
	Shutdown() error
	NewJob(jobDefinition any, task any, options ...any) (any, error)
}

// Wallclock defines the interface for time-related operations
type Wallclock interface {
	Epochs() any
	Slots() any
	OnEpochChanged(callback func(any))
	OnSlotChanged(callback func(any))
}

// Metadata defines the interface for metadata operations
type Metadata interface {
	Network() any
	Wallclock() any
	Client(ctx context.Context) string
	NodeVersion(ctx context.Context) string
	OverrideNetworkName(name string)
}

// Deriver defines the interface for event derivers
type Deriver interface {
	Name() string
	ActivationFork() spec.DataVersion
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	OnEventsDerived(ctx context.Context, callback func(ctx context.Context, events []*xatu.DecoratedEvent) error)
}

// Sink defines the interface for output sinks
type Sink interface {
	Name() string
	Type() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// Logger defines the interface for logging operations
type Logger interface {
	WithField(key string, value any) logrus.FieldLogger
	WithFields(fields logrus.Fields) logrus.FieldLogger
	WithError(err error) logrus.FieldLogger

	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// TimeProvider defines the interface for time operations
type TimeProvider interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
	Sleep(d time.Duration)
	After(d time.Duration) <-chan time.Time
}

// NTPClient defines the interface for NTP operations
type NTPClient interface {
	Query(host string) (NTPResponse, error)
}

// NTPResponse represents an NTP query response
type NTPResponse interface {
	Validate() error
	ClockOffset() time.Duration
}

// MetricsRecorder defines the interface for metrics operations
type MetricsRecorder interface {
	AddDecoratedEvent(count int, eventType *xatu.DecoratedEvent, network string)
}
