package execution

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// StateSizeWatcher polls the execution client's debug_stateSize endpoint
// to collect state size metrics. It supports three trigger modes:
// - "head": Triggered by consensus layer head events
// - "block": Triggered by execution layer block events (requires WebSocket)
// - "interval": Periodic polling at a configured interval
type StateSizeWatcher struct {
	client   *Client
	log      logrus.FieldLogger
	config   *StateSizeConfig
	wg       sync.WaitGroup
	ctx      context.Context //nolint:containedctx // This is a derived context from the parent context.
	cancel   context.CancelFunc
	callback func(context.Context, *DebugStateSizeResponse) error
}

// NewStateSizeWatcher creates a new StateSizeWatcher instance.
func NewStateSizeWatcher(
	client *Client,
	log logrus.FieldLogger,
	config *StateSizeConfig,
	callback func(context.Context, *DebugStateSizeResponse) error,
) *StateSizeWatcher {
	return &StateSizeWatcher{
		client:   client,
		log:      log.WithField("component", "execution/state_size_watcher"),
		config:   config,
		wg:       sync.WaitGroup{},
		callback: callback,
	}
}

// Start initializes the watcher's context and launches background goroutines
// based on the configured trigger mode.
func (w *StateSizeWatcher) Start(parentCtx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(parentCtx)

	switch w.config.TriggerMode {
	case "interval":
		w.log.WithField("interval_seconds", w.config.IntervalSeconds).Info("Starting state size watcher in interval mode")
		w.startPeriodicPoller()
	case "block":
		w.log.Info("Starting state size watcher in block mode (subscribing to execution layer blocks)")

		if err := w.startBlockSubscription(); err != nil {
			return err
		}
	default:
		// Default to "head" mode
		w.log.Info("State size watcher initialized in head mode (will be triggered by consensus head events)")
	}

	return nil
}

// Stop gracefully shuts down the watcher and waits for all goroutines to complete.
func (w *StateSizeWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}

	w.wg.Wait()

	w.log.Info("State size watcher stopped")

	return nil
}

// OnHeadEvent should be called when a new head event is received from the consensus layer.
// This is only used when TriggerMode is "head".
func (w *StateSizeWatcher) OnHeadEvent(ctx context.Context) error {
	if w.config.TriggerMode != "head" {
		return nil
	}

	return w.fetchAndReport(ctx)
}

// startPeriodicPoller starts a goroutine that polls debug_stateSize at regular intervals.
func (w *StateSizeWatcher) startPeriodicPoller() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(time.Duration(w.config.IntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				if err := w.fetchAndReport(w.ctx); err != nil {
					w.log.WithError(err).Error("Failed to fetch and report state size")
				}
			}
		}
	}()
}

// startBlockSubscription subscribes to new execution layer blocks and polls state size on each new block.
func (w *StateSizeWatcher) startBlockSubscription() error {
	headerChan, errChan, err := w.client.SubscribeToNewHeads(w.ctx)
	if err != nil {
		return err
	}

	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		for {
			select {
			case <-w.ctx.Done():
				return
			case err := <-errChan:
				if err != nil {
					w.log.WithError(err).Error("Error in block subscription")
				}

				return
			case header := <-headerChan:
				if header != nil {
					w.log.WithField("block_number", header.Number.String()).Debug("Received new block, fetching state size")

					if err := w.fetchAndReport(w.ctx); err != nil {
						w.log.WithError(err).Error("Failed to fetch and report state size")
					}
				}
			}
		}
	}()

	return nil
}

// fetchAndReport polls the debug_stateSize endpoint and invokes the callback with the result.
func (w *StateSizeWatcher) fetchAndReport(ctx context.Context) error {
	result, err := w.client.DebugStateSize(ctx, "latest")
	if err != nil {
		w.log.WithError(err).Warn("Failed to fetch state size from execution client")

		return err
	}

	w.log.WithFields(logrus.Fields{
		"block_number": result.BlockNumber,
		"state_root":   result.StateRoot,
	}).Debug("Fetched state size data")

	if err := w.callback(ctx, result); err != nil {
		w.log.WithError(err).Error("Failed to process state size data")

		return err
	}

	return nil
}
