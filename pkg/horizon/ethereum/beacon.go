package ethereum

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ehttp "github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	backoff "github.com/cenkalti/backoff/v5"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
)

// ErrNoHealthyNodes is returned when no healthy beacon nodes are available.
var ErrNoHealthyNodes = errors.New("no healthy beacon nodes available")

// NodeState represents the connection state of a beacon node.
type NodeState int

const (
	// NodeStateDisconnected indicates the node has not connected yet.
	NodeStateDisconnected NodeState = iota
	// NodeStateConnecting indicates the node is attempting to connect.
	NodeStateConnecting
	// NodeStateConnected indicates the node is connected but may not be healthy.
	NodeStateConnected
	// NodeStateReconnecting indicates the node is reconnecting after a failure.
	NodeStateReconnecting
)

// BeaconNodeWrapper wraps a single beacon node with its health status.
type BeaconNodeWrapper struct {
	config  BeaconNodeConfig
	node    beacon.Node
	healthy bool
	state   NodeState
	mu      sync.RWMutex
	log     logrus.FieldLogger
}

// IsHealthy returns whether the beacon node is healthy.
func (w *BeaconNodeWrapper) IsHealthy() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.healthy
}

// SetHealthy sets the health status of the beacon node.
func (w *BeaconNodeWrapper) SetHealthy(healthy bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.healthy = healthy
}

// Name returns the name of the beacon node.
func (w *BeaconNodeWrapper) Name() string {
	return w.config.Name
}

// Node returns the underlying beacon node.
func (w *BeaconNodeWrapper) Node() beacon.Node {
	return w.node
}

// BeaconNodePool manages a pool of beacon nodes with health checking and failover.
type BeaconNodePool struct {
	config  *Config
	log     logrus.FieldLogger
	metrics *Metrics

	nodes []*BeaconNodeWrapper
	mu    sync.RWMutex

	// Shared services across all nodes (uses first healthy node)
	metadata *services.MetadataService
	duties   *services.DutiesService

	// Block cache shared across all nodes
	sfGroup          *singleflight.Group
	blockCache       *ttlcache.Cache[string, *spec.VersionedSignedBeaconBlock]
	blockPreloadChan chan string
	blockPreloadSem  chan struct{}

	onReadyCallbacks []func(ctx context.Context) error
	shutdownChan     chan struct{}
	wg               sync.WaitGroup
}

// NewBeaconNodePool creates a new BeaconNodePool with the given configuration.
func NewBeaconNodePool(_ context.Context, config *Config, log logrus.FieldLogger) (*BeaconNodePool, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	namespace := "xatu_horizon"
	metrics := NewMetrics(namespace)

	pool := &BeaconNodePool{
		config:           config,
		log:              log.WithField("component", "ethereum/beacon_pool"),
		metrics:          metrics,
		nodes:            make([]*BeaconNodeWrapper, 0, len(config.BeaconNodes)),
		sfGroup:          &singleflight.Group{},
		blockPreloadChan: make(chan string, config.BlockPreloadQueueSize),
		blockPreloadSem:  make(chan struct{}, config.BlockPreloadWorkers),
		shutdownChan:     make(chan struct{}),
	}

	// Create TTL cache for blocks
	pool.blockCache = ttlcache.New(
		ttlcache.WithTTL[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheTTL.Duration),
		ttlcache.WithCapacity[string, *spec.VersionedSignedBeaconBlock](config.BlockCacheSize),
	)

	// Create beacon node wrappers for each configured node
	for _, nodeCfg := range config.BeaconNodes {
		wrapper, err := pool.createNodeWrapper(nodeCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create beacon node %s: %w", nodeCfg.Name, err)
		}

		pool.nodes = append(pool.nodes, wrapper)

		metrics.SetBeaconNodeStatus(nodeCfg.Name, BeaconNodeStatusConnecting)
	}

	return pool, nil
}

// createNodeWrapper creates a new BeaconNodeWrapper for the given configuration.
func (p *BeaconNodePool) createNodeWrapper(nodeCfg BeaconNodeConfig) (*BeaconNodeWrapper, error) {
	opts := *beacon.
		DefaultOptions().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

	opts.GoEth2ClientParams = []ehttp.Parameter{
		ehttp.WithEnforceJSON(true),
	}

	opts.HealthCheck.Interval.Duration = p.config.HealthCheckInterval.Duration
	opts.HealthCheck.SuccessfulResponses = 1

	// Disable beacon subscriptions - Horizon will handle SSE separately
	opts.BeaconSubscription.Enabled = false

	node := beacon.NewNode(p.log, &beacon.Config{
		Name:    nodeCfg.Name,
		Addr:    nodeCfg.Address,
		Headers: nodeCfg.Headers,
	}, "xatu_horizon", opts)

	return &BeaconNodeWrapper{
		config:  nodeCfg,
		node:    node,
		healthy: false,
		log:     p.log.WithField("beacon_node", nodeCfg.Name),
	}, nil
}

// Start starts the beacon node pool and all its nodes.
func (p *BeaconNodePool) Start(ctx context.Context) error {
	p.log.Info("Starting beacon node pool")

	// Start block cache eviction tracking
	p.blockCache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, *spec.VersionedSignedBeaconBlock]) {
		p.log.WithField("identifier", item.Key()).WithField("reason", reason).Trace("Block evicted from cache")
	})

	go p.blockCache.Start()

	// Start block preload workers
	for i := uint64(0); i < p.config.BlockPreloadWorkers; i++ {
		p.wg.Add(1)

		go func() {
			defer p.wg.Done()

			for {
				select {
				case <-p.shutdownChan:
					return
				case identifier := <-p.blockPreloadChan:
					p.log.WithField("identifier", identifier).Trace("Preloading block")
					_, _ = p.GetBeaconBlock(ctx, identifier)
				}
			}
		}()
	}

	// Start each beacon node with retry logic
	for _, wrapper := range p.nodes {
		p.wg.Add(1)

		go func(w *BeaconNodeWrapper) {
			defer p.wg.Done()

			p.startNodeWithRetry(ctx, w)
		}(wrapper)
	}

	// Start health check goroutine
	p.wg.Add(1)

	go p.runHealthChecks(ctx)

	// Wait for at least one node to become healthy
	if err := p.waitForHealthyNode(ctx); err != nil {
		return err
	}

	// Initialize shared services using first healthy node
	if err := p.initializeServices(ctx); err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Run on-ready callbacks
	for _, callback := range p.onReadyCallbacks {
		if err := callback(ctx); err != nil {
			return fmt.Errorf("on-ready callback failed: %w", err)
		}
	}

	p.log.Info("Beacon node pool started")

	return nil
}

// Stop stops the beacon node pool.
func (p *BeaconNodePool) Stop(_ context.Context) error {
	p.log.Info("Stopping beacon node pool")

	close(p.shutdownChan)
	p.blockCache.Stop()
	p.wg.Wait()

	return nil
}

// waitForHealthyNode waits for at least one beacon node to become healthy.
func (p *BeaconNodePool) waitForHealthyNode(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.NewTimer(60 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return ErrNoHealthyNodes
		case <-ticker.C:
			if _, err := p.GetHealthyNode(); err == nil {
				return nil
			}
		}
	}
}

// runHealthChecks runs periodic health checks on all beacon nodes.
func (p *BeaconNodePool) runHealthChecks(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckInterval.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.shutdownChan:
			return
		case <-ticker.C:
			p.checkAllNodesHealth()
		}
	}
}

// checkAllNodesHealth checks the health of all beacon nodes.
func (p *BeaconNodePool) checkAllNodesHealth() {
	for _, wrapper := range p.nodes {
		start := time.Now()

		healthy := p.checkNodeHealth(wrapper)
		wrapper.SetHealthy(healthy)

		duration := time.Since(start).Seconds()
		p.metrics.ObserveHealthCheckDuration(wrapper.Name(), duration)

		if healthy {
			p.metrics.SetBeaconNodeStatus(wrapper.Name(), BeaconNodeStatusHealthy)
			p.metrics.IncHealthCheck(wrapper.Name(), BeaconNodeStatusHealthy)
		} else {
			p.metrics.SetBeaconNodeStatus(wrapper.Name(), BeaconNodeStatusUnhealthy)
			p.metrics.IncHealthCheck(wrapper.Name(), BeaconNodeStatusUnhealthy)
		}
	}
}

// checkNodeHealth checks if a beacon node is healthy.
func (p *BeaconNodePool) checkNodeHealth(wrapper *BeaconNodeWrapper) bool {
	status := wrapper.node.Status()
	if status == nil {
		p.log.WithField("node", wrapper.Name()).Trace("Node status is nil")

		return false
	}

	syncState := status.SyncState()
	if syncState == nil {
		p.log.WithField("node", wrapper.Name()).Trace("Node sync state is nil")

		return false
	}

	// Consider healthy if sync distance is reasonable
	if syncState.SyncDistance > 10 {
		p.log.WithField("node", wrapper.Name()).
			WithField("sync_distance", syncState.SyncDistance).
			Trace("Node sync distance too high")

		return false
	}

	return true
}

// initializeServices initializes shared services using the first healthy node.
func (p *BeaconNodePool) initializeServices(ctx context.Context) error {
	healthyWrapper, err := p.GetHealthyNode()
	if err != nil {
		return err
	}

	metadata := services.NewMetadataService(p.log, healthyWrapper.node)
	p.metadata = &metadata

	if p.config.OverrideNetworkName != "" {
		p.metadata.OverrideNetworkName(p.config.OverrideNetworkName)
	}

	duties := services.NewDutiesService(p.log, healthyWrapper.node, p.metadata)
	p.duties = &duties

	// Start metadata service
	if err := p.metadata.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metadata service: %w", err)
	}

	// Wait for metadata service to be ready
	readyChan := make(chan error, 1)

	p.metadata.OnReady(ctx, func(ctx context.Context) error {
		readyChan <- nil

		return nil
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-readyChan:
		if err != nil {
			return err
		}
	case <-time.After(30 * time.Second):
		return errors.New("timeout waiting for metadata service to be ready")
	}

	// Verify network
	if p.metadata.Network.Name == networks.NetworkNameUnknown {
		return errors.New("unknown network detected - please override the network name via config")
	}

	// Start duties service
	if err := p.duties.Start(ctx); err != nil {
		return fmt.Errorf("failed to start duties service: %w", err)
	}

	p.log.WithField("network", p.metadata.Network.Name).Info("Services initialized")

	return nil
}

// GetHealthyNode returns any healthy beacon node.
func (p *BeaconNodePool) GetHealthyNode() (*BeaconNodeWrapper, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, wrapper := range p.nodes {
		if wrapper.IsHealthy() {
			return wrapper, nil
		}
	}

	return nil, ErrNoHealthyNodes
}

// GetAllNodes returns all beacon node wrappers.
func (p *BeaconNodePool) GetAllNodes() []*BeaconNodeWrapper {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.nodes
}

// GetHealthyNodes returns all healthy beacon nodes.
func (p *BeaconNodePool) GetHealthyNodes() []*BeaconNodeWrapper {
	p.mu.RLock()
	defer p.mu.RUnlock()

	healthy := make([]*BeaconNodeWrapper, 0, len(p.nodes))

	for _, wrapper := range p.nodes {
		if wrapper.IsHealthy() {
			healthy = append(healthy, wrapper)
		}
	}

	return healthy
}

// Metadata returns the shared metadata service.
func (p *BeaconNodePool) Metadata() *services.MetadataService {
	return p.metadata
}

// Duties returns the shared duties service.
func (p *BeaconNodePool) Duties() *services.DutiesService {
	return p.duties
}

// OnReady registers a callback to be called when the pool is ready.
func (p *BeaconNodePool) OnReady(callback func(ctx context.Context) error) {
	p.onReadyCallbacks = append(p.onReadyCallbacks, callback)
}

// Synced checks if the pool has at least one synced beacon node.
func (p *BeaconNodePool) Synced(ctx context.Context) error {
	_, err := p.GetHealthyNode()
	if err != nil {
		return err
	}

	if p.metadata == nil {
		return errors.New("metadata service not initialized")
	}

	if err := p.metadata.Ready(ctx); err != nil {
		return fmt.Errorf("metadata service not ready: %w", err)
	}

	if p.duties == nil {
		return errors.New("duties service not initialized")
	}

	if err := p.duties.Ready(ctx); err != nil {
		return fmt.Errorf("duties service not ready: %w", err)
	}

	return nil
}

// GetBeaconBlock fetches a beacon block from any healthy node, using cache.
func (p *BeaconNodePool) GetBeaconBlock(ctx context.Context, identifier string) (*spec.VersionedSignedBeaconBlock, error) {
	// Check cache first
	if item := p.blockCache.Get(identifier); item != nil {
		p.metrics.IncBlockCacheHits(string(p.metadata.Network.Name))

		return item.Value(), nil
	}

	p.metrics.IncBlockCacheMisses(string(p.metadata.Network.Name))

	// Use singleflight to avoid duplicate requests
	result, err, _ := p.sfGroup.Do(identifier, func() (any, error) {
		// Acquire semaphore
		p.blockPreloadSem <- struct{}{}

		defer func() { <-p.blockPreloadSem }()

		// Get any healthy node and fetch the block
		wrapper, err := p.GetHealthyNode()
		if err != nil {
			return nil, err
		}

		p.metrics.IncBlocksFetched(wrapper.Name(), string(p.metadata.Network.Name))

		block, err := wrapper.node.FetchBlock(ctx, identifier)
		if err != nil {
			p.metrics.IncBlockFetchErrors(wrapper.Name(), string(p.metadata.Network.Name))

			return nil, fmt.Errorf("failed to fetch block from %s: %w", wrapper.Name(), err)
		}

		// Cache the block
		p.blockCache.Set(identifier, block, p.config.BlockCacheTTL.Duration)

		return block, nil
	})
	if err != nil {
		return nil, err
	}

	block, ok := result.(*spec.VersionedSignedBeaconBlock)
	if !ok {
		return nil, errors.New("unexpected result type from singleflight")
	}

	return block, nil
}

// LazyLoadBeaconBlock queues a block for preloading.
func (p *BeaconNodePool) LazyLoadBeaconBlock(identifier string) {
	// Skip if already cached
	if item := p.blockCache.Get(identifier); item != nil {
		return
	}

	// Non-blocking send to preload channel
	select {
	case p.blockPreloadChan <- identifier:
	default:
		// Channel full, skip preloading
	}
}

// NodeCount returns the total number of configured beacon nodes.
func (p *BeaconNodePool) NodeCount() int {
	return len(p.nodes)
}

// HealthyNodeCount returns the number of healthy beacon nodes.
func (p *BeaconNodePool) HealthyNodeCount() int {
	count := 0

	for _, wrapper := range p.nodes {
		if wrapper.IsHealthy() {
			count++
		}
	}

	return count
}

// PreferNode returns the specified node if it's healthy, otherwise falls back to any healthy node.
// The nodeAddress should match the Address field of a configured beacon node.
func (p *BeaconNodePool) PreferNode(nodeAddress string) (*BeaconNodeWrapper, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// First, try to find the preferred node
	for _, wrapper := range p.nodes {
		if wrapper.config.Address == nodeAddress && wrapper.IsHealthy() {
			return wrapper, nil
		}
	}

	// Preferred node not available, fall back to any healthy node
	for _, wrapper := range p.nodes {
		if wrapper.IsHealthy() {
			p.log.WithFields(logrus.Fields{
				"preferred": nodeAddress,
				"fallback":  wrapper.config.Address,
			}).Debug("Preferred node unavailable, using fallback")

			return wrapper, nil
		}
	}

	return nil, ErrNoHealthyNodes
}

// startNodeWithRetry starts a beacon node with exponential backoff retry.
func (p *BeaconNodePool) startNodeWithRetry(ctx context.Context, wrapper *BeaconNodeWrapper) {
	wrapper.mu.Lock()
	wrapper.state = NodeStateConnecting
	wrapper.mu.Unlock()

	p.metrics.SetBeaconNodeStatus(wrapper.Name(), BeaconNodeStatusConnecting)

	operation := func() (struct{}, error) {
		select {
		case <-ctx.Done():
			return struct{}{}, backoff.Permanent(ctx.Err())
		case <-p.shutdownChan:
			return struct{}{}, backoff.Permanent(errors.New("pool shutting down"))
		default:
		}

		if err := wrapper.node.Start(ctx); err != nil {
			wrapper.log.WithError(err).Warn("Failed to start beacon node, will retry")
			p.metrics.SetBeaconNodeStatus(wrapper.Name(), BeaconNodeStatusUnhealthy)

			return struct{}{}, err
		}

		return struct{}{}, nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 30 * time.Second

	retryOpts := []backoff.RetryOption{
		backoff.WithBackOff(bo),
		backoff.WithNotify(func(err error, duration time.Duration) {
			wrapper.log.WithError(err).WithField("next_retry", duration).
				Warn("Beacon node connection failed, retrying")

			wrapper.mu.Lock()
			wrapper.state = NodeStateReconnecting
			wrapper.mu.Unlock()
		}),
	}
	if _, err := backoff.Retry(ctx, operation, retryOpts...); err != nil {
		// Only log if not a context cancellation or shutdown
		if !errors.Is(err, context.Canceled) {
			wrapper.log.WithError(err).Error("Beacon node connection permanently failed")
		}

		wrapper.mu.Lock()
		wrapper.state = NodeStateDisconnected
		wrapper.healthy = false
		wrapper.mu.Unlock()

		p.metrics.SetBeaconNodeStatus(wrapper.Name(), BeaconNodeStatusUnhealthy)

		return
	}

	wrapper.mu.Lock()
	wrapper.state = NodeStateConnected
	wrapper.mu.Unlock()

	wrapper.log.Info("Beacon node connected successfully")
}

// GetState returns the current connection state of the beacon node.
func (w *BeaconNodeWrapper) GetState() NodeState {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.state
}

// Address returns the address of the beacon node.
func (w *BeaconNodeWrapper) Address() string {
	return w.config.Address
}
