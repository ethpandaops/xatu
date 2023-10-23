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
	bsp, err := NewBatchItemProcessor[TestItem](nil, "processor", nullLogger())
	require.NoError(t, err)

	if err := bsp.Write(context.Background(), []*TestItem{{
		name: "test",
	}}); err != nil {
		t.Errorf("failed to Write to the BatchItemProcessor: %v", err)
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
			ssp, err := createAndRegisterBatchSP(option.o, &te)
			if err != nil {
				t.Fatalf("%s: Error creating new instance of BatchItemProcessor\n", option.name)
			}
			if ssp == nil {
				t.Fatalf("%s: Error creating new instance of BatchItemProcessor\n", option.name)
			}

			for i := 0; i < option.genNumItems; i++ {
				if option.writeNumItems > 0 && i%option.writeNumItems == 0 {
					time.Sleep(10 * time.Millisecond)
				}
				if err := ssp.Write(context.Background(), []*TestItem{{
					name: "test",
				}}); err != nil {
					t.Errorf("%s: Error writing to BatchItemProcessor\n", option.name)
				}
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
	bvp, err := NewBatchItemProcessor[TestItem](
		exp,
		"processor",
		nullLogger(),
		// Set a non-zero export timeout so a deadline is set.
		WithExportTimeout(1*time.Microsecond),
		WithBatchTimeout(1*time.Microsecond),
	)
	require.NoError(t, err)

	if err := bvp.Write(context.Background(), []*TestItem{{
		name: "test",
	}}); err != nil {
		t.Errorf("failed to Write to the BatchItemProcessor: %v", err)
	}

	time.Sleep(1 * time.Millisecond)

	if exp.err != context.DeadlineExceeded {
		t.Errorf("context deadline error not returned: got %+v", exp.err)
	}
}

func createAndRegisterBatchSP[T TestItem](options []BatchItemProcessorOption, te *testBatchExporter[T]) (*BatchItemProcessor[T], error) {
	return NewBatchItemProcessor[T](te, "processor", nullLogger(), options...)
}

func TestBatchItemProcessorShutdown(t *testing.T) {
	var bp testBatchExporter[TestItem]
	bvp, err := NewBatchItemProcessor[TestItem](&bp, "processor", nullLogger())
	require.NoError(t, err)

	err = bvp.Shutdown(context.Background())
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

func TestBatchItemProcessorDrainQueue(t *testing.T) {
	be := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&be, "processor", nullLogger(), WithMaxExportBatchSize(5), WithBatchTimeout(5*time.Minute))
	require.NoError(t, err)

	itemsToExport := 500

	for i := 0; i < itemsToExport; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{
			name: strconv.Itoa(i),
		}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	require.NoError(t, bsp.Shutdown(context.Background()), "shutting down BatchItemProcessor")

	assert.Equal(t, itemsToExport, be.len(), "Queue should have been drained on shutdown")
}

func TestBatchItemProcessorPostShutdown(t *testing.T) {
	be := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&be, "processor", nullLogger(), WithMaxExportBatchSize(50), WithBatchTimeout(5*time.Millisecond))
	require.NoError(t, err)

	for i := 0; i < 60; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{
			name: strconv.Itoa(i),
		}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	require.NoError(t, bsp.Shutdown(context.Background()), "shutting down BatchItemProcessor")

	lenJustAfterShutdown := be.len()

	for i := 0; i < 60; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{
			name: strconv.Itoa(i),
		}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	assert.Equal(t, lenJustAfterShutdown, be.len(), "Write should have no effect after Shutdown")
}

type slowExporter[T TestItem] struct {
	itemsExported int
}

func (slowExporter[T]) Shutdown(context.Context) error { return nil }
func (t *slowExporter[T]) ExportItems(ctx context.Context, items []*T) error {
	time.Sleep(100 * time.Millisecond)

	t.itemsExported += len(items)

	<-ctx.Done()

	return ctx.Err()
}

func TestMultipleWorkersConsumeConcurrently(t *testing.T) {
	te := slowExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithBatchTimeout(5*time.Minute), WithWorkers(20))
	require.NoError(t, err)

	itemsToExport := 100

	for i := 0; i < itemsToExport; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{name: strconv.Itoa(i)}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	time.Sleep(1 * time.Second) // give some time for workers to process

	if te.itemsExported != itemsToExport {
		t.Errorf("Expected all items to be exported, got: %v", te.itemsExported)
	}
}

func TestWorkersProcessBatches(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5))
	require.NoError(t, err)

	itemsToExport := 50

	for i := 0; i < itemsToExport; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{name: strconv.Itoa(i)}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	time.Sleep(1 * time.Second)

	if te.getBatchCount() != 5 {
		t.Errorf("Expected 5 batches, got: %v", te.getBatchCount())
	}

	if te.len() != itemsToExport {
		t.Errorf("Expected all items to be exported, got: %v", te.len())
	}
}

func TestDrainQueueWithMultipleWorkers(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5))
	require.NoError(t, err)

	itemsToExport := 100

	for i := 0; i < itemsToExport; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{name: strconv.Itoa(i)}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	if err := bsp.Shutdown(context.Background()); err != nil {
		t.Errorf("Error shutting down BatchItemProcessor\n")
	}

	if te.len() != itemsToExport {
		t.Errorf("Expected all items to be exported after drain, got: %v", te.len())
	}
}

func TestBatchItemProcessorTimerFunctionality(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	batchTimeout := 500 * time.Millisecond
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(50), WithBatchTimeout(batchTimeout), WithWorkers(5))
	require.NoError(t, err)

	// Add items less than the max batch size
	itemsToExport := 25

	for i := 0; i < itemsToExport; i++ {
		if err := bsp.Write(context.Background(), []*TestItem{{name: strconv.Itoa(i)}}); err != nil {
			t.Errorf("Error writing to BatchItemProcessor\n")
		}
	}

	// Wait for more than the batchTimeout duration
	time.Sleep(batchTimeout + 100*time.Millisecond)

	// Check if items have been exported due to timer trigger
	if te.len() != itemsToExport {
		t.Errorf("Expected %v items to be exported due to timer, but got: %v", itemsToExport, te.len())
	}

	// Ensure that it was exported as a single batch
	if te.getBatchCount() != 1 {
		t.Errorf("Expected 1 batch to be exported due to timer, but got: %v", te.getBatchCount())
	}
}

type indefiniteExporter[T TestItem] struct{}

func (indefiniteExporter[T]) Shutdown(context.Context) error { return nil }
func (indefiniteExporter[T]) ExportItems(ctx context.Context, _ []*T) error {
	<-ctx.Done()

	return ctx.Err()
}

func TestBatchItemProcessorTimeout(t *testing.T) {
	// Add timeout to context to test deadline
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	<-ctx.Done()

	bsp, err := NewBatchItemProcessor[TestItem](indefiniteExporter[TestItem]{}, "processor", nullLogger(), WithShippingMethod(ShippingMethodSync), WithExportTimeout(time.Millisecond*10))
	if err != nil {
		t.Fatalf("failed to create batch processor: %v", err)
	}

	if got, want := bsp.Write(ctx, []*TestItem{{}}), context.DeadlineExceeded; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func TestBatchItemProcessorCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel the context
	go func() {
		time.Sleep(time.Millisecond * 100)

		cancel()
	}()

	bsp, err := NewBatchItemProcessor[TestItem](indefiniteExporter[TestItem]{}, "processor", nullLogger(), WithShippingMethod(ShippingMethodSync))
	if err != nil {
		t.Fatalf("failed to create batch processor: %v", err)
	}

	if got, want := bsp.Write(ctx, []*TestItem{{}}), context.Canceled; !errors.Is(got, want) {
		t.Errorf("expected %q error, got %v", want, got)
	}
}

func nullLogger() *logrus.Logger {
	log := logrus.New()
	log.Out = io.Discard

	return log
}

// ErrorItemExporter is an ItemExporter that only returns errors when exporting.
type ErrorItemExporter[T any] struct{}

func (ErrorItemExporter[T]) Shutdown(context.Context) error { return nil }

func (ErrorItemExporter[T]) ExportItems(ctx context.Context, _ []*T) error {
	return errors.New("export error")
}

// TestBatchItemProcessorWithSyncErrorExporter tests a processor with ShippingMethod = sync and an exporter that only returns errors.
func TestBatchItemProcessorWithAsyncErrorExporter(t *testing.T) {
	bsp, err := NewBatchItemProcessor[TestItem](ErrorItemExporter[TestItem]{}, "processor", nullLogger(), WithShippingMethod(ShippingMethodSync))
	if err != nil {
		t.Fatalf("failed to create batch processor: %v", err)
	}

	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	if err == nil {
		t.Errorf("Expected write to fail")
	}
}
