package iterator

import (
	"context"
	"errors"
	"sync"

	"github.com/attestantio/go-eth2-client/spec"
	cldataIterator "github.com/ethpandaops/xatu/pkg/cldata/iterator"
	"github.com/sirupsen/logrus"
)

// ErrDualIteratorClosed is returned when the dual iterator is closed.
var ErrDualIteratorClosed = errors.New("dual iterator closed")

// DualIterator multiplexes HEAD and FILL iterators with HEAD priority.
// It implements the shared cldata iterator interface so derivers can consume
// a single iterator while still getting both real-time and catch-up positions.
type DualIterator struct {
	log    logrus.FieldLogger
	config *CoordinatorConfig

	head *HeadIterator
	fill *FillIterator

	headCh chan *cldataIterator.Position
	fillCh chan *cldataIterator.Position

	done chan struct{}
	wg   sync.WaitGroup
}

// NewDualIterator creates a new DualIterator.
func NewDualIterator(
	log logrus.FieldLogger,
	config *CoordinatorConfig,
	head *HeadIterator,
	fill *FillIterator,
) *DualIterator {
	if config == nil {
		config = &CoordinatorConfig{
			Head: HeadIteratorConfig{Enabled: true},
			Fill: FillIteratorConfig{Enabled: true},
		}
	}

	return &DualIterator{
		log:    log.WithField("component", "iterator/dual"),
		config: config,
		head:   head,
		fill:   fill,
		done:   make(chan struct{}),
	}
}

// Start initializes both iterators and begins their processing loops.
func (d *DualIterator) Start(ctx context.Context, activationFork spec.DataVersion) error {
	if d.config.Head.Enabled {
		if err := d.head.Start(ctx, activationFork); err != nil {
			return err
		}

		d.headCh = make(chan *cldataIterator.Position, 16)
		d.wg.Add(1)
		go d.runHead(ctx)
	} else {
		d.log.Warn("HEAD iterator disabled")
	}

	if d.config.Fill.Enabled {
		if err := d.fill.Start(ctx, activationFork); err != nil {
			return err
		}

		d.fillCh = make(chan *cldataIterator.Position, 16)
		d.wg.Add(1)
		go d.runFill(ctx)
	} else {
		d.log.Warn("FILL iterator disabled")
	}

	return nil
}

func (d *DualIterator) runHead(ctx context.Context) {
	defer d.wg.Done()

	for {
		pos, err := d.head.Next(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrIteratorClosed) {
				return
			}

			d.log.WithError(err).Debug("HEAD iterator Next() returned error")
			continue
		}

		if pos == nil {
			continue
		}

		select {
		case d.headCh <- pos:
		case <-ctx.Done():
			return
		case <-d.done:
			return
		}
	}
}

func (d *DualIterator) runFill(ctx context.Context) {
	defer d.wg.Done()

	for {
		pos, err := d.fill.Next(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, ErrIteratorClosed) {
				return
			}

			d.log.WithError(err).Debug("FILL iterator Next() returned error")
			continue
		}

		if pos == nil {
			continue
		}

		select {
		case d.fillCh <- pos:
		case <-ctx.Done():
			return
		case <-d.done:
			return
		}
	}
}

// Next returns the next position to process, prioritizing HEAD positions.
func (d *DualIterator) Next(ctx context.Context) (*cldataIterator.Position, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-d.done:
			return nil, ErrDualIteratorClosed
		default:
		}

		// Non-blocking check for HEAD priority.
		select {
		case pos, ok := <-d.headCh:
			if !ok {
				d.headCh = nil
				break
			}

			return pos, nil
		default:
		}

		select {
		case pos, ok := <-d.headCh:
			if !ok {
				d.headCh = nil
				break
			}

			return pos, nil
		case pos, ok := <-d.fillCh:
			if !ok {
				d.fillCh = nil
				break
			}

			return pos, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-d.done:
			return nil, ErrDualIteratorClosed
		}

		if d.headCh == nil && d.fillCh == nil {
			return nil, ErrDualIteratorClosed
		}
	}
}

// UpdateLocation persists the current position to the appropriate iterator.
func (d *DualIterator) UpdateLocation(ctx context.Context, position *cldataIterator.Position) error {
	switch position.Direction {
	case cldataIterator.DirectionForward:
		if d.head == nil {
			return errors.New("head iterator not available")
		}

		return d.head.UpdateLocation(ctx, position)
	case cldataIterator.DirectionBackward:
		if d.fill == nil {
			return errors.New("fill iterator not available")
		}

		return d.fill.UpdateLocation(ctx, position)
	default:
		return errors.New("unknown iterator direction")
	}
}

// Stop stops both iterators and waits for goroutines to finish.
func (d *DualIterator) Stop(ctx context.Context) error {
	close(d.done)

	if d.head != nil {
		_ = d.head.Stop(ctx)
	}
	if d.fill != nil {
		_ = d.fill.Stop(ctx)
	}

	d.wg.Wait()

	if d.headCh != nil {
		close(d.headCh)
	}
	if d.fillCh != nil {
		close(d.fillCh)
	}

	return nil
}

// Verify DualIterator implements the Iterator interface.
var _ cldataIterator.Iterator = (*DualIterator)(nil)
