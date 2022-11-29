/*
 * This processor tests was adapted from the OpenTelemetry Collector's batch processor tests.
 *
 * Authors: OpenTelemetry
 * URL: https://github.com/open-telemetry/opentelemetry-go/blob/496c086ece129182662c14d6a023a2b2da09fe30/sdk/trace/batch_event_processor_test.go
 */
package processor

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testBatchExporter struct {
	mu            sync.Mutex
	events        []*xatu.DecoratedEvent
	sizes         []int
	batchCount    int
	shutdownCount int
	errors        []error
	droppedCount  int
	idx           int
	err           error
}

func (t *testBatchExporter) ExportEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.idx < len(t.errors) {
		t.droppedCount += len(events)
		err := t.errors[t.idx]
		t.idx++

		return err
	}

	select {
	case <-ctx.Done():
		t.err = ctx.Err()
		return ctx.Err()
	default:
	}

	t.events = append(t.events, events...)
	t.sizes = append(t.sizes, len(events))
	t.batchCount++

	return nil
}

func (t *testBatchExporter) Shutdown(context.Context) error {
	t.shutdownCount++
	return nil
}

func (t *testBatchExporter) len() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.events)
}

func (t *testBatchExporter) getBatchCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.batchCount
}

func TestNewBatchDecoratedEventProcessorWithNilExporter(t *testing.T) {
	bsp := NewBatchDecoratedEventProcessor(nil, nullLogger())

	event := &xatu.DecoratedEvent{}

	bsp.Write(event)

	if err := bsp.ForceFlush(context.Background()); err != nil {
		t.Errorf("failed to ForceFlush the BatchDecoratedEventProcessor: %v", err)
	}

	if err := bsp.Shutdown(context.Background()); err != nil {
		t.Errorf("failed to Shutdown the BatchDecoratedEventProcessor: %v", err)
	}
}

type testOption struct {
	name           string
	o              []BatchDecoratedEventProcessorOption
	writeNumEvents int
	genNumEvents   int
	wantNumEvents  int
	wantBatchCount int
}

func TestNewBatchDecoratedEventProcessorWithOptions(t *testing.T) {
	schDelay := 100 * time.Millisecond
	options := []testOption{
		{
			name:           "default",
			o:              []BatchDecoratedEventProcessorOption{},
			genNumEvents:   2053,
			wantNumEvents:  2048,
			wantBatchCount: 4,
		},
		{
			name: "non-default BatchTimeout",
			o: []BatchDecoratedEventProcessorOption{
				WithBatchTimeout(schDelay),
			},
			writeNumEvents: 2048,
			genNumEvents:   2053,
			wantNumEvents:  2053,
			wantBatchCount: 5,
		},
		{
			name: "non-default MaxQueueSize and BatchTimeout",
			o: []BatchDecoratedEventProcessorOption{
				WithBatchTimeout(schDelay),
				WithMaxQueueSize(200),
			},
			writeNumEvents: 200,
			genNumEvents:   205,
			wantNumEvents:  205,
			wantBatchCount: 1,
		},
		{
			name: "blocking option",
			o: []BatchDecoratedEventProcessorOption{
				WithBatchTimeout(schDelay),
				WithMaxQueueSize(200),
				WithMaxExportBatchSize(20),
			},
			writeNumEvents: 200,
			genNumEvents:   205,
			wantNumEvents:  205,
			wantBatchCount: 11,
		},
	}

	for _, option := range options {
		t.Run(option.name, func(t *testing.T) {
			te := testBatchExporter{}
			ssp := createAndRegisterBatchSP(option.o, &te)
			if ssp == nil {
				t.Fatalf("%s: Error creating new instance of BatchDecoratedEventProcessor\n", option.name)
			}

			for i := 0; i < option.genNumEvents; i++ {
				if option.writeNumEvents > 0 && i%option.writeNumEvents == 0 {
					time.Sleep(10 * time.Millisecond)
				}
				ssp.Write(&xatu.DecoratedEvent{
					Meta: &xatu.Meta{
						Client: &xatu.ClientMeta{
							Name: strconv.Itoa(i),
						},
					},
				})
			}

			time.Sleep(schDelay * 2)

			gotNumOfEvents := te.len()
			if option.wantNumEvents > 0 && option.wantNumEvents != gotNumOfEvents {
				t.Errorf("number of exported events: got %+v, want %+v\n",
					gotNumOfEvents, option.wantNumEvents)
			}

			gotBatchCount := te.getBatchCount()
			if option.wantBatchCount > 0 && gotBatchCount != option.wantBatchCount {
				t.Errorf("number batches: got %+v, want >= %+v\n",
					gotBatchCount, option.wantBatchCount)
				t.Errorf("Batches %v\n", te.sizes)
			}
		})
	}
}

type stuckExporter struct {
	testBatchExporter
}

// ExportEvents waits for ctx to expire and returns that error.
func (e *stuckExporter) ExportEvents(ctx context.Context, _ []*xatu.DecoratedEvent) error {
	<-ctx.Done()
	e.err = ctx.Err()

	return ctx.Err()
}

func TestBatchDecoratedEventProcessorExportTimeout(t *testing.T) {
	exp := new(stuckExporter)
	bvp := NewBatchDecoratedEventProcessor(
		exp,
		nullLogger(),
		// Set a non-zero export timeout so a deadline is set.
		WithExportTimeout(1*time.Microsecond),
		WithBatchTimeout(1*time.Microsecond),
	)

	bvp.Write(&xatu.DecoratedEvent{
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "test",
			},
		},
	})

	time.Sleep(1 * time.Millisecond)

	if exp.err != context.DeadlineExceeded {
		t.Errorf("context deadline error not returned: got %+v", exp.err)
	}
}

func createAndRegisterBatchSP(options []BatchDecoratedEventProcessorOption, te *testBatchExporter) *BatchDecoratedEventProcessor {
	return NewBatchDecoratedEventProcessor(te, nullLogger(), options...)
}

func TestBatchDecoratedEventProcessorShutdown(t *testing.T) {
	var bp testBatchExporter
	bvp := NewBatchDecoratedEventProcessor(&bp, nullLogger())

	err := bvp.Shutdown(context.Background())
	if err != nil {
		t.Error("Error shutting the BatchDecoratedEventProcessor down\n")
	}

	assert.Equal(t, 1, bp.shutdownCount, "shutdown from event exporter not called")

	// Multiple call to Shutdown() should not panic.
	err = bvp.Shutdown(context.Background())
	if err != nil {
		t.Error("Error shutting the BatchDecoratedEventProcessor down\n")
	}

	assert.Equal(t, 1, bp.shutdownCount)
}

func TestBatchDecoratedEventProcessorPostShutdown(t *testing.T) {
	be := testBatchExporter{}
	bsp := NewBatchDecoratedEventProcessor(&be, nullLogger(), WithMaxExportBatchSize(50))

	for i := 0; i < 60; i++ {
		bsp.Write(&xatu.DecoratedEvent{
			Meta: &xatu.Meta{
				Client: &xatu.ClientMeta{
					Name: strconv.Itoa(i),
				},
			},
		})
	}

	require.NoError(t, bsp.Shutdown(context.Background()), "shutting down BatchDecoratedEventProcessor")

	lenJustAfterShutdown := be.len()

	assert.NoError(t, bsp.ForceFlush(context.Background()), "force flushing BatchDecoratedEventProcessor")

	assert.Equal(t, lenJustAfterShutdown, be.len(), "Write and ForceFlush should have no effect after Shutdown")
}

func TestBatchDecoratedEventProcessorForceFlushSucceeds(t *testing.T) {
	te := testBatchExporter{}
	option := testOption{
		name: "default BatchDecoratedEventProcessorOptions",
		o: []BatchDecoratedEventProcessorOption{
			WithMaxQueueSize(0),
			WithMaxExportBatchSize(3000),
		},
		genNumEvents:   2053,
		wantNumEvents:  2053,
		wantBatchCount: 1,
	}
	ssp := createAndRegisterBatchSP(option.o, &te)

	if ssp == nil {
		t.Fatalf("%s: Error creating new instance of BatchDecoratedEventProcessor\n", option.name)
	}

	for i := 0; i < option.genNumEvents; i++ {
		ssp.Write(&xatu.DecoratedEvent{
			Meta: &xatu.Meta{
				Client: &xatu.ClientMeta{
					Name: strconv.Itoa(i),
				},
			},
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held event batches
	err := ssp.ForceFlush(context.Background())

	assertMaxEventDiff(t, te.len(), option.wantNumEvents, 10)

	gotBatchCount := te.getBatchCount()
	if gotBatchCount < option.wantBatchCount {
		t.Errorf("number batches: got %+v, want >= %+v\n",
			gotBatchCount, option.wantBatchCount)
		t.Errorf("Batches %v\n", te.sizes)
	}

	assert.NoError(t, err)
}

func TestBatchDecoratedEventProcessorDropBatchIfFailed(t *testing.T) {
	te := testBatchExporter{
		errors: []error{errors.New("fail to export")},
	}
	option := testOption{
		o: []BatchDecoratedEventProcessorOption{
			WithMaxQueueSize(0),
			WithMaxExportBatchSize(2000),
		},
		genNumEvents:   1000,
		wantNumEvents:  1000,
		wantBatchCount: 1,
	}
	ssp := createAndRegisterBatchSP(option.o, &te)

	if ssp == nil {
		t.Fatalf("%s: Error creating new instance of BatchDecoratedEventProcessor\n", option.name)
	}

	for i := 0; i < option.genNumEvents; i++ {
		ssp.Write(&xatu.DecoratedEvent{
			Meta: &xatu.Meta{
				Client: &xatu.ClientMeta{
					Name: strconv.Itoa(i),
				},
			},
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held event batches
	err := ssp.ForceFlush(context.Background())
	assert.Error(t, err)
	assert.EqualError(t, err, "fail to export")

	// First flush will fail, nothing should be exported.
	assertMaxEventDiff(t, te.droppedCount, option.wantNumEvents, 10)
	assert.Equal(t, 0, te.len())
	assert.Equal(t, 0, te.getBatchCount())

	// Generate a new batch, this will succeed
	for i := 0; i < option.genNumEvents; i++ {
		ssp.Write(&xatu.DecoratedEvent{
			Meta: &xatu.Meta{
				Client: &xatu.ClientMeta{
					Name: strconv.Itoa(i),
				},
			},
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held event batches
	err = ssp.ForceFlush(context.Background())
	assert.NoError(t, err)

	assertMaxEventDiff(t, te.len(), option.wantNumEvents, 10)
	gotBatchCount := te.getBatchCount()

	if gotBatchCount < option.wantBatchCount {
		t.Errorf("number batches: got %+v, want >= %+v\n",
			gotBatchCount, option.wantBatchCount)
		t.Errorf("Batches %v\n", te.sizes)
	}
}

func assertMaxEventDiff(t *testing.T, got, want, maxDif int) {
	t.Helper()

	eventDifference := want - got
	if eventDifference < 0 {
		eventDifference *= -1
	}

	if eventDifference > maxDif {
		t.Errorf("number of exported event not equal to or within %d less than: got %+v, want %+v\n",
			maxDif, got, want)
	}
}

type indefiniteExporter struct{}

func (indefiniteExporter) Shutdown(context.Context) error { return nil }
func (indefiniteExporter) ExportEvents(ctx context.Context, _ []*xatu.DecoratedEvent) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestBatchDecoratedEventProcessorForceFlushTimeout(t *testing.T) {
	// Add timeout to context to test deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	bsp := NewBatchDecoratedEventProcessor(indefiniteExporter{}, nullLogger())
	if got, want := bsp.ForceFlush(ctx), context.DeadlineExceeded; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func TestBatchDecoratedEventProcessorForceFlushCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context
	cancel()

	bsp := NewBatchDecoratedEventProcessor(indefiniteExporter{}, nullLogger())
	if got, want := bsp.ForceFlush(ctx), context.Canceled; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func nullLogger() *logrus.Logger {
	log := logrus.New()
	log.Out = io.Discard

	return log
}
