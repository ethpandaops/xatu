package iterator

import (
	"context"
	"errors"
	"sync"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// CoordinatorConfig holds configuration for the iterator coordinator.
type CoordinatorConfig struct {
	// Head is the configuration for the HEAD iterator.
	Head HeadIteratorConfig `yaml:"head"`
	// Fill is the configuration for the FILL iterator.
	Fill FillIteratorConfig `yaml:"fill"`
}

// Validate validates the configuration.
func (c *CoordinatorConfig) Validate() error {
	if err := c.Head.Validate(); err != nil {
		return err
	}

	if err := c.Fill.Validate(); err != nil {
		return err
	}

	return nil
}

// Coordinator manages the dual HEAD and FILL iterators, ensuring they run
// in separate goroutines without blocking each other. HEAD has priority for
// real-time block processing, while FILL handles consistency catch-up.
type Coordinator struct {
	log     logrus.FieldLogger
	config  *CoordinatorConfig
	metrics *CoordinatorMetrics

	headIterator *HeadIterator
	fillIterator *FillIterator

	// wg tracks running goroutines for graceful shutdown.
	wg sync.WaitGroup

	// done signals shutdown to all goroutines.
	done chan struct{}
}

// CoordinatorMetrics tracks metrics for the iterator coordinator.
type CoordinatorMetrics struct {
	headRunning prometheus.Gauge
	fillRunning prometheus.Gauge
}

// NewCoordinatorMetrics creates metrics for the coordinator.
func NewCoordinatorMetrics(namespace string) *CoordinatorMetrics {
	m := &CoordinatorMetrics{
		headRunning: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "iterator_coordinator",
			Name:      "head_running",
			Help:      "Indicates if the HEAD iterator is running (1) or stopped (0)",
		}),
		fillRunning: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "iterator_coordinator",
			Name:      "fill_running",
			Help:      "Indicates if the FILL iterator is running (1) or stopped (0)",
		}),
	}

	prometheus.MustRegister(
		m.headRunning,
		m.fillRunning,
	)

	return m
}

// NewCoordinator creates a new iterator coordinator.
func NewCoordinator(
	log logrus.FieldLogger,
	config *CoordinatorConfig,
	headIterator *HeadIterator,
	fillIterator *FillIterator,
) *Coordinator {
	if config == nil {
		config = &CoordinatorConfig{}
	}

	return &Coordinator{
		log:          log.WithField("component", "iterator/coordinator"),
		config:       config,
		metrics:      NewCoordinatorMetrics("xatu_horizon"),
		headIterator: headIterator,
		fillIterator: fillIterator,
		done:         make(chan struct{}),
	}
}

// Start starts both iterators in their own goroutines.
// HEAD iterator runs first for priority, FILL iterator follows.
// Both iterators coordinate through the coordinator service to avoid
// processing the same slots.
func (c *Coordinator) Start(ctx context.Context, activationFork spec.DataVersion) error {
	c.log.WithField("activation_fork", activationFork.String()).
		Info("Starting dual-iterator coordinator")

	// Start HEAD iterator in its dedicated goroutine.
	// HEAD has priority and processes real-time SSE block events immediately.
	if c.config.Head.Enabled {
		if err := c.headIterator.Start(ctx, activationFork); err != nil {
			return err
		}

		c.wg.Add(1)

		go c.runHeadIterator(ctx)

		c.metrics.headRunning.Set(1)

		c.log.Info("HEAD iterator started in dedicated goroutine")
	} else {
		c.log.Warn("HEAD iterator is disabled")
	}

	// Start FILL iterator in its separate goroutine.
	// FILL runs independently and never blocks HEAD.
	if c.config.Fill.Enabled {
		if err := c.fillIterator.Start(ctx, activationFork); err != nil {
			return err
		}

		c.wg.Add(1)

		go c.runFillIterator(ctx)

		c.metrics.fillRunning.Set(1)

		c.log.Info("FILL iterator started in separate goroutine")
	} else {
		c.log.Warn("FILL iterator is disabled")
	}

	return nil
}

// runHeadIterator runs the HEAD iterator loop in its own goroutine.
// HEAD has priority - it receives real-time SSE block events and processes them immediately.
func (c *Coordinator) runHeadIterator(ctx context.Context) {
	defer c.wg.Done()
	defer c.metrics.headRunning.Set(0)

	c.log.Debug("HEAD iterator goroutine started")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("HEAD iterator stopping due to context cancellation")

			return
		case <-c.done:
			c.log.Info("HEAD iterator stopping due to coordinator shutdown")

			return
		default:
			// Get next position from HEAD iterator.
			// This blocks until a block event is received from SSE.
			pos, err := c.headIterator.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, ErrIteratorClosed) {
					return
				}

				// Log and continue on other errors.
				c.log.WithError(err).Debug("HEAD iterator Next() returned error")

				continue
			}

			if pos == nil {
				continue
			}

			// Position is available for processing.
			// The deriver will call UpdateLocation after processing completes.
			c.log.WithField("slot", pos.Slot).Trace("HEAD position ready for processing")
		}
	}
}

// runFillIterator runs the FILL iterator loop in its own goroutine.
// FILL runs independently and never blocks HEAD.
func (c *Coordinator) runFillIterator(ctx context.Context) {
	defer c.wg.Done()
	defer c.metrics.fillRunning.Set(0)

	c.log.Debug("FILL iterator goroutine started")

	for {
		select {
		case <-ctx.Done():
			c.log.Info("FILL iterator stopping due to context cancellation")

			return
		case <-c.done:
			c.log.Info("FILL iterator stopping due to coordinator shutdown")

			return
		default:
			// Get next position from FILL iterator.
			// This walks slots from fill_slot toward HEAD - LAG.
			pos, err := c.fillIterator.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, ErrIteratorClosed) {
					return
				}

				// Log and continue on other errors.
				c.log.WithError(err).Debug("FILL iterator Next() returned error")

				continue
			}

			if pos == nil {
				continue
			}

			// Position is available for processing.
			// The deriver will call UpdateLocation after processing completes.
			c.log.WithField("slot", pos.Slot).Trace("FILL position ready for processing")
		}
	}
}

// Stop stops both iterators and waits for goroutines to finish.
func (c *Coordinator) Stop(ctx context.Context) error {
	c.log.Info("Stopping dual-iterator coordinator")

	// Signal all goroutines to stop.
	close(c.done)

	// Stop individual iterators.
	if c.config.Head.Enabled {
		if err := c.headIterator.Stop(ctx); err != nil {
			c.log.WithError(err).Warn("Error stopping HEAD iterator")
		}
	}

	if c.config.Fill.Enabled {
		if err := c.fillIterator.Stop(ctx); err != nil {
			c.log.WithError(err).Warn("Error stopping FILL iterator")
		}
	}

	// Wait for goroutines to finish.
	c.wg.Wait()

	c.log.Info("Dual-iterator coordinator stopped")

	return nil
}

// HeadIterator returns the HEAD iterator.
func (c *Coordinator) HeadIterator() *HeadIterator {
	return c.headIterator
}

// FillIterator returns the FILL iterator.
func (c *Coordinator) FillIterator() *FillIterator {
	return c.fillIterator
}
