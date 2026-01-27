package horizon

import (
	"context"
	"sync"

	"github.com/ethpandaops/xatu/pkg/horizon/cache"
	"github.com/ethpandaops/xatu/pkg/horizon/subscription"
	"github.com/sirupsen/logrus"
)

// BlockEventBroadcaster deduplicates block events and fan-outs to subscribers.
type BlockEventBroadcaster struct {
	log        logrus.FieldLogger
	dedup      *cache.DedupCache
	input      <-chan subscription.BlockEvent
	bufferSize int

	mu          sync.RWMutex
	subscribers []chan subscription.BlockEvent

	done chan struct{}
	wg   sync.WaitGroup
}

// NewBlockEventBroadcaster creates a new broadcaster.
func NewBlockEventBroadcaster(
	log logrus.FieldLogger,
	dedup *cache.DedupCache,
	input <-chan subscription.BlockEvent,
	bufferSize int,
) *BlockEventBroadcaster {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	return &BlockEventBroadcaster{
		log:        log.WithField("component", "block_broadcaster"),
		dedup:      dedup,
		input:      input,
		bufferSize: bufferSize,
		done:       make(chan struct{}),
	}
}

// Subscribe returns a channel that receives deduplicated block events.
func (b *BlockEventBroadcaster) Subscribe() <-chan subscription.BlockEvent {
	ch := make(chan subscription.BlockEvent, b.bufferSize)

	b.mu.Lock()
	b.subscribers = append(b.subscribers, ch)
	b.mu.Unlock()

	return ch
}

// Start begins processing incoming block events.
func (b *BlockEventBroadcaster) Start(ctx context.Context) {
	b.wg.Add(1)

	go func() {
		defer b.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-b.done:
				return
			case event, ok := <-b.input:
				if !ok {
					return
				}

				// Deduplicate by block root.
				if b.dedup.Check(event.BlockRoot.String()) {
					continue
				}

				b.mu.RLock()
				subscribers := append([]chan subscription.BlockEvent(nil), b.subscribers...)
				b.mu.RUnlock()

				for i, subscriber := range subscribers {
					select {
					case subscriber <- event:
					default:
						b.log.WithFields(logrus.Fields{
							"slot":        event.Slot,
							"subscriber":  i,
							"block_root":  event.BlockRoot.String(),
							"event_node":  event.NodeName,
							"buffer_size": b.bufferSize,
						}).Warn("Block event subscriber channel full, dropping event")
					}
				}
			}
		}
	}()
}

// Stop stops the broadcaster and closes subscriber channels.
func (b *BlockEventBroadcaster) Stop() {
	close(b.done)
	b.wg.Wait()

	b.mu.Lock()
	for _, subscriber := range b.subscribers {
		close(subscriber)
	}
	b.subscribers = nil
	b.mu.Unlock()
}
