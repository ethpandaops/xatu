package cannon

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CannonFactory provides a testable factory for creating Cannon instances
type CannonFactory struct {
	config       *Config
	beacon       BeaconNode
	coordinator  Coordinator
	blockprint   Blockprint
	logger       logrus.FieldLogger
	sinks        []output.Sink
	scheduler    Scheduler
	timeProvider TimeProvider
	ntpClient    NTPClient
	overrides    *Override
	id           uuid.UUID
	metrics      *Metrics
	isTestMode   bool
}

// NewCannonFactory creates a new factory for creating Cannon instances
func NewCannonFactory() *CannonFactory {
	return &CannonFactory{
		timeProvider: &DefaultTimeProvider{},
		ntpClient:    &DefaultNTPClient{},
		isTestMode:   false,
	}
}

// NewTestCannonFactory creates a new factory for testing
func NewTestCannonFactory() *CannonFactory {
	return &CannonFactory{
		timeProvider: &DefaultTimeProvider{},
		ntpClient:    &DefaultNTPClient{},
		isTestMode:   true,
	}
}

// WithConfig sets the configuration
func (f *CannonFactory) WithConfig(config *Config) *CannonFactory {
	f.config = config

	return f
}

// WithBeaconNode sets the beacon node
func (f *CannonFactory) WithBeaconNode(beacon BeaconNode) *CannonFactory {
	f.beacon = beacon

	return f
}

// WithCoordinator sets the coordinator client
func (f *CannonFactory) WithCoordinator(coord Coordinator) *CannonFactory {
	f.coordinator = coord

	return f
}

// WithBlockprint sets the blockprint client
func (f *CannonFactory) WithBlockprint(bp Blockprint) *CannonFactory {
	f.blockprint = bp
	return f
}

// WithLogger sets the logger
func (f *CannonFactory) WithLogger(logger logrus.FieldLogger) *CannonFactory {
	f.logger = logger
	return f
}

// WithSinks sets the output sinks
func (f *CannonFactory) WithSinks(sinks []output.Sink) *CannonFactory {
	f.sinks = sinks
	return f
}

// WithScheduler sets the scheduler
func (f *CannonFactory) WithScheduler(scheduler Scheduler) *CannonFactory {
	f.scheduler = scheduler
	return f
}

// WithTimeProvider sets the time provider
func (f *CannonFactory) WithTimeProvider(tp TimeProvider) *CannonFactory {
	f.timeProvider = tp
	return f
}

// WithNTPClient sets the NTP client
func (f *CannonFactory) WithNTPClient(ntp NTPClient) *CannonFactory {
	f.ntpClient = ntp
	return f
}

// WithOverrides sets the overrides
func (f *CannonFactory) WithOverrides(overrides *Override) *CannonFactory {
	f.overrides = overrides
	return f
}

// WithID sets the cannon ID
func (f *CannonFactory) WithID(id uuid.UUID) *CannonFactory {
	f.id = id
	return f
}

// WithMetrics sets the metrics
func (f *CannonFactory) WithMetrics(metrics *Metrics) *CannonFactory {
	f.metrics = metrics
	return f
}

// Build creates a new Cannon instance with the configured components
func (f *CannonFactory) Build() (*TestableCannon, error) {
	if f.config == nil {
		return nil, ErrConfigRequired
	}

	if f.logger == nil {
		f.logger = logrus.NewEntry(logrus.New())
	}

	if f.id == uuid.Nil {
		f.id = uuid.New()
	}

	if f.metrics == nil {
		if f.isTestMode {
			// For tests, we don't want to register real metrics to avoid conflicts
			f.metrics = &Metrics{}
		} else {
			f.metrics = NewMetrics("xatu_cannon")
		}
	}

	if f.scheduler == nil {
		scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
		if err != nil {
			return nil, err
		}
		f.scheduler = &DefaultScheduler{scheduler: scheduler}
	}

	return &TestableCannon{
		config:        f.config,
		sinks:         f.sinks,
		beacon:        f.beacon,
		coordinator:   f.coordinator,
		blockprint:    f.blockprint,
		clockDrift:    time.Duration(0),
		log:           f.logger,
		id:            f.id,
		metrics:       f.metrics,
		scheduler:     f.scheduler,
		timeProvider:  f.timeProvider,
		ntpClient:     f.ntpClient,
		eventDerivers: nil,
		shutdownFuncs: []func(ctx context.Context) error{},
		overrides:     f.overrides,
	}, nil
}

// BuildDefault creates a Cannon instance using default production components
func (f *CannonFactory) BuildDefault(ctx context.Context) (*TestableCannon, error) {
	if f.config == nil {
		return nil, ErrConfigRequired
	}

	if f.logger == nil {
		f.logger = logrus.NewEntry(logrus.New())
	}

	// Create default components if not provided
	if f.beacon == nil {
		beacon, err := ethereum.NewBeaconNode(ctx, f.config.Name, &f.config.Ethereum, f.logger)
		if err != nil {
			return nil, err
		}
		f.beacon = &DefaultBeaconNodeWrapper{beacon: beacon}
	}

	if f.coordinator == nil {
		coord, err := coordinator.New(&f.config.Coordinator, f.logger)
		if err != nil {
			return nil, err
		}
		f.coordinator = &DefaultCoordinatorWrapper{client: coord}
	}

	if f.sinks == nil {
		sinks, err := f.config.CreateSinks(f.logger)
		if err != nil {
			return nil, err
		}
		f.sinks = sinks
	}

	return f.Build()
}

// TestableCannon is a testable version of Cannon that exposes internal components
type TestableCannon struct {
	config        *Config
	sinks         []output.Sink
	beacon        BeaconNode
	coordinator   Coordinator
	blockprint    Blockprint
	clockDrift    time.Duration
	log           logrus.FieldLogger
	id            uuid.UUID
	metrics       *Metrics
	scheduler     Scheduler
	timeProvider  TimeProvider
	ntpClient     NTPClient
	eventDerivers []Deriver
	shutdownFuncs []func(ctx context.Context) error
	overrides     *Override
}

// GetBeacon returns the beacon node for testing
func (c *TestableCannon) GetBeacon() BeaconNode {
	return c.beacon
}

// GetCoordinator returns the coordinator for testing
func (c *TestableCannon) GetCoordinator() Coordinator {
	return c.coordinator
}

// GetScheduler returns the scheduler for testing
func (c *TestableCannon) GetScheduler() Scheduler {
	return c.scheduler
}

// GetTimeProvider returns the time provider for testing
func (c *TestableCannon) GetTimeProvider() TimeProvider {
	return c.timeProvider
}

// GetNTPClient returns the NTP client for testing
func (c *TestableCannon) GetNTPClient() NTPClient {
	return c.ntpClient
}

// GetSinks returns the sinks for testing
func (c *TestableCannon) GetSinks() []output.Sink {
	return c.sinks
}

// GetEventDerivers returns the event derivers for testing
func (c *TestableCannon) GetEventDerivers() []Deriver {
	return c.eventDerivers
}

// SetEventDerivers sets the event derivers for testing
func (c *TestableCannon) SetEventDerivers(derivers []Deriver) {
	c.eventDerivers = derivers
}

// Start starts the cannon - delegates to the original Start method logic
func (c *TestableCannon) Start(ctx context.Context) error {
	// Start the scheduler (this calls scheduler.Start())
	if c.scheduler != nil {
		c.scheduler.Start()
	}

	// Start the beacon node
	if c.beacon != nil {
		if err := c.beacon.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Shutdown shuts down the cannon
func (c *TestableCannon) Shutdown(ctx context.Context) error {
	for _, sink := range c.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	for _, fun := range c.shutdownFuncs {
		if err := fun(ctx); err != nil {
			return err
		}
	}

	if err := c.scheduler.Shutdown(); err != nil {
		return err
	}

	for _, deriver := range c.eventDerivers {
		if err := deriver.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}
