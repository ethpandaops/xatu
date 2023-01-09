/*
 * This processor tests was adapted from the OpenTelemetry Collector's batch processor tests.
 *
 * Authors: OpenTelemetry
 * URL: https://github.com/open-telemetry/opentelemetry-go/blob/496c086ece129182662c14d6a023a2b2da09fe30/sdk/trace/batch_item_processor_test.go
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

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestItem struct {
	name string
}

type testBatchExporter[T TestItem] struct {
	mu            sync.Mutex
	items         []*T
	sizes         []int
	batchCount    int
	shutdownCount int
	errors        []error
	droppedCount  int
	idx           int
	err           error
}

func (t *testBatchExporter[T]) ExportItems(ctx context.Context, items []*T) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.idx < len(t.errors) {
		t.droppedCount += len(items)
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

	t.items = append(t.items, items...)
	t.sizes = append(t.sizes, len(items))
	t.batchCount++

	return nil
}

func (t *testBatchExporter[T]) Shutdown(context.Context) error {
	t.shutdownCount++
	return nil
}

func (t *testBatchExporter[T]) len() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return len(t.items)
}

func (t *testBatchExporter[T]) getBatchCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.batchCount
}

func TestNewBatchItemProcessorWithNilExporter(t *testing.T) {
	bsp := NewBatchItemProcessor[TestItem](nil, nullLogger())

	bsp.Write(&TestItem{
		name: "test",
	})

	if err := bsp.ForceFlush(context.Background()); err != nil {
		t.Errorf("failed to ForceFlush the BatchItemProcessor: %v", err)
	}

	if err := bsp.Shutdown(context.Background()); err != nil {
		t.Errorf("failed to Shutdown the BatchItemProcessor: %v", err)
	}
}

type testOption struct {
	name           string
	o              []BatchItemProcessorOption
	writeNumItems  int
	genNumItems    int
	wantNumItems   int
	wantBatchCount int
}

func TestNewBatchItemProcessorWithOptions(t *testing.T) {
	schDelay := 100 * time.Millisecond
	options := []testOption{
		{
			name:           "default",
			o:              []BatchItemProcessorOption{},
			genNumItems:    2053,
			wantNumItems:   2048,
			wantBatchCount: 4,
		},
		{
			name: "non-default BatchTimeout",
			o: []BatchItemProcessorOption{
				WithBatchTimeout(schDelay),
			},
			writeNumItems:  2048,
			genNumItems:    2053,
			wantNumItems:   2053,
			wantBatchCount: 5,
		},
		{
			name: "non-default MaxQueueSize and BatchTimeout",
			o: []BatchItemProcessorOption{
				WithBatchTimeout(schDelay),
				WithMaxQueueSize(200),
			},
			writeNumItems:  200,
			genNumItems:    205,
			wantNumItems:   205,
			wantBatchCount: 1,
		},
		{
			name: "blocking option",
			o: []BatchItemProcessorOption{
				WithBatchTimeout(schDelay),
				WithMaxQueueSize(200),
				WithMaxExportBatchSize(20),
			},
			writeNumItems:  200,
			genNumItems:    205,
			wantNumItems:   205,
			wantBatchCount: 11,
		},
	}

	for _, option := range options {
		t.Run(option.name, func(t *testing.T) {
			te := testBatchExporter[TestItem]{}
			ssp := createAndRegisterBatchSP(option.o, &te)
			if ssp == nil {
				t.Fatalf("%s: Error creating new instance of BatchItemProcessor\n", option.name)
			}

			for i := 0; i < option.genNumItems; i++ {
				if option.writeNumItems > 0 && i%option.writeNumItems == 0 {
					time.Sleep(10 * time.Millisecond)
				}
				ssp.Write(&TestItem{
					name: "test",
				})
			}

			time.Sleep(schDelay * 2)

			gotNumOfItems := te.len()
			if option.wantNumItems > 0 && option.wantNumItems != gotNumOfItems {
				t.Errorf("number of exported items: got %+v, want %+v\n",
					gotNumOfItems, option.wantNumItems)
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

type stuckExporter[T TestItem] struct {
	testBatchExporter[T]
}

// ExportItems waits for ctx to expire and returns that error.
func (e *stuckExporter[T]) ExportItems(ctx context.Context, _ []*T) error {
	<-ctx.Done()
	e.err = ctx.Err()

	return ctx.Err()
}

func TestBatchItemProcessorExportTimeout(t *testing.T) {
	exp := new(stuckExporter[TestItem])
	bvp := NewBatchItemProcessor[TestItem](
		exp,
		nullLogger(),
		// Set a non-zero export timeout so a deadline is set.
		WithExportTimeout(1*time.Microsecond),
		WithBatchTimeout(1*time.Microsecond),
	)

	bvp.Write(&TestItem{
		name: "test",
	})

	time.Sleep(1 * time.Millisecond)

	if exp.err != context.DeadlineExceeded {
		t.Errorf("context deadline error not returned: got %+v", exp.err)
	}
}

func createAndRegisterBatchSP[T TestItem](options []BatchItemProcessorOption, te *testBatchExporter[T]) *BatchItemProcessor[T] {
	return NewBatchItemProcessor[T](te, nullLogger(), options...)
}

func TestBatchItemProcessorShutdown(t *testing.T) {
	var bp testBatchExporter[TestItem]
	bvp := NewBatchItemProcessor[TestItem](&bp, nullLogger())

	err := bvp.Shutdown(context.Background())
	if err != nil {
		t.Error("Error shutting the BatchItemProcessor down\n")
	}

	assert.Equal(t, 1, bp.shutdownCount, "shutdown from item exporter not called")

	// Multiple call to Shutdown() should not panic.
	err = bvp.Shutdown(context.Background())
	if err != nil {
		t.Error("Error shutting the BatchItemProcessor down\n")
	}

	assert.Equal(t, 1, bp.shutdownCount)
}

func TestBatchItemProcessorPostShutdown(t *testing.T) {
	be := testBatchExporter[TestItem]{}
	bsp := NewBatchItemProcessor[TestItem](&be, nullLogger(), WithMaxExportBatchSize(50))

	for i := 0; i < 60; i++ {
		bsp.Write(&TestItem{
			name: strconv.Itoa(i),
		})
	}

	require.NoError(t, bsp.Shutdown(context.Background()), "shutting down BatchItemProcessor")

	lenJustAfterShutdown := be.len()

	assert.NoError(t, bsp.ForceFlush(context.Background()), "force flushing BatchItemProcessor")

	assert.Equal(t, lenJustAfterShutdown, be.len(), "Write and ForceFlush should have no effect after Shutdown")
}

func TestBatchItemProcessorForceFlushSucceeds(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	option := testOption{
		name: "default BatchItemProcessorOptions",
		o: []BatchItemProcessorOption{
			WithMaxQueueSize(0),
			WithMaxExportBatchSize(3000),
		},
		genNumItems:    2053,
		wantNumItems:   2053,
		wantBatchCount: 1,
	}
	ssp := createAndRegisterBatchSP(option.o, &te)

	if ssp == nil {
		t.Fatalf("%s: Error creating new instance of BatchItemProcessor\n", option.name)
	}

	for i := 0; i < option.genNumItems; i++ {
		ssp.Write(&TestItem{
			name: strconv.Itoa(i),
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held item batches
	err := ssp.ForceFlush(context.Background())

	assertMaxItemDiff(t, te.len(), option.wantNumItems, 10)

	gotBatchCount := te.getBatchCount()
	if gotBatchCount < option.wantBatchCount {
		t.Errorf("number batches: got %+v, want >= %+v\n",
			gotBatchCount, option.wantBatchCount)
		t.Errorf("Batches %v\n", te.sizes)
	}

	assert.NoError(t, err)
}

func TestBatchItemProcessorDropBatchIfFailed(t *testing.T) {
	te := testBatchExporter[TestItem]{
		errors: []error{errors.New("fail to export")},
	}
	option := testOption{
		o: []BatchItemProcessorOption{
			WithMaxQueueSize(0),
			WithMaxExportBatchSize(2000),
		},
		genNumItems:    1000,
		wantNumItems:   1000,
		wantBatchCount: 1,
	}
	ssp := createAndRegisterBatchSP(option.o, &te)

	if ssp == nil {
		t.Fatalf("%s: Error creating new instance of BatchItemProcessor\n", option.name)
	}

	for i := 0; i < option.genNumItems; i++ {
		ssp.Write(&TestItem{
			name: strconv.Itoa(i),
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held item batches
	err := ssp.ForceFlush(context.Background())
	assert.Error(t, err)
	assert.EqualError(t, err, "fail to export")

	// First flush will fail, nothing should be exported.
	assertMaxItemDiff(t, te.droppedCount, option.wantNumItems, 10)
	assert.Equal(t, 0, te.len())
	assert.Equal(t, 0, te.getBatchCount())

	// Generate a new batch, this will succeed
	for i := 0; i < option.genNumItems; i++ {
		ssp.Write(&TestItem{
			name: strconv.Itoa(i),
		})
		time.Sleep(1 * time.Millisecond)
	}

	// Force flush any held item batches
	err = ssp.ForceFlush(context.Background())
	assert.NoError(t, err)

	assertMaxItemDiff(t, te.len(), option.wantNumItems, 10)
	gotBatchCount := te.getBatchCount()

	if gotBatchCount < option.wantBatchCount {
		t.Errorf("number batches: got %+v, want >= %+v\n",
			gotBatchCount, option.wantBatchCount)
		t.Errorf("Batches %v\n", te.sizes)
	}
}

func assertMaxItemDiff(t *testing.T, got, want, maxDif int) {
	t.Helper()

	itemDifference := want - got
	if itemDifference < 0 {
		itemDifference *= -1
	}

	if itemDifference > maxDif {
		t.Errorf("number of exported item not equal to or within %d less than: got %+v, want %+v\n",
			maxDif, got, want)
	}
}

type indefiniteExporter[T TestItem] struct{}

func (indefiniteExporter[T]) Shutdown(context.Context) error { return nil }
func (indefiniteExporter[T]) ExportItems(ctx context.Context, _ []*T) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestBatchItemProcessorForceFlushTimeout(t *testing.T) {
	// Add timeout to context to test deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	bsp := NewBatchItemProcessor[TestItem](indefiniteExporter[TestItem]{}, nullLogger())
	if got, want := bsp.ForceFlush(ctx), context.DeadlineExceeded; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func TestBatchItemProcessorForceFlushCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context
	cancel()

	bsp := NewBatchItemProcessor[TestItem](indefiniteExporter[TestItem]{}, nullLogger())
	if got, want := bsp.ForceFlush(ctx), context.Canceled; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func nullLogger() *logrus.Logger {
	log := logrus.New()
	log.Out = io.Discard

	return log
}
