package buffer

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type DecoratedEventBuffer struct {
	ch chan *xatu.DecoratedEvent

	onAddedCallbacks []func(ctx context.Context, event *xatu.DecoratedEvent)
}

func NewDecoratedEventBuffer(size int) *DecoratedEventBuffer {
	return &DecoratedEventBuffer{
		ch: make(chan *xatu.DecoratedEvent, size),
	}
}

func (d *DecoratedEventBuffer) Write(ctx context.Context, event *xatu.DecoratedEvent) {
	d.ch <- event

	for _, callback := range d.onAddedCallbacks {
		go callback(ctx, event)
	}
}

func (d *DecoratedEventBuffer) Len() int {
	return len(d.ch)
}

func (d *DecoratedEventBuffer) OnAdded(callback func(ctx context.Context, event *xatu.DecoratedEvent)) {
	d.onAddedCallbacks = append(d.onAddedCallbacks, callback)
}

func (d *DecoratedEventBuffer) Read() <-chan *xatu.DecoratedEvent {
	return d.ch
}

func (d *DecoratedEventBuffer) Channel() chan *xatu.DecoratedEvent {
	return d.ch
}
