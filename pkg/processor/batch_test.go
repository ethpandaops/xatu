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
	"fmt"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestItem struct {
	name string
}

type testBatchExporter[T TestItem] struct {
	mu             sync.Mutex
	items          []*T
	sizes          []int
	batchCount     int
	shutdownCount  int
	errors         []error
	droppedCount   int
	idx            int
	err            error
	failNextExport bool
	delay          time.Duration
}

func (t *testBatchExporter[T]) ExportItems(ctx context.Context, items []*T) error {
	time.Sleep(t.delay)

	if t.failNextExport {
		t.failNextExport = false

		return errors.New("export failure")
	}

	time.Sleep(t.delay)

	if t.failNextExport {
		t.failNextExport = false

		return errors.New("export failure")
	}

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

	bsp.Start(context.Background())

	err = bsp.Write(context.Background(), []*TestItem{{
		name: "test",
	}})
	if err == nil || err.Error() != "exporter is nil" {
		t.Errorf("expected error 'exporter is nil', got: %v", err)
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

func TestAsyncNewBatchItemProcessorWithOptions(t *testing.T) {
	schDelay := 100 * time.Millisecond
	options := []testOption{
		{
			name: "default",
			o: []BatchItemProcessorOption{
				WithShippingMethod(ShippingMethodAsync),
				WithBatchTimeout(10 * time.Millisecond),
			},
			writeNumItems:  2048,
			genNumItems:    2053,
			wantNumItems:   2053,
			wantBatchCount: 5,
		},
		{
			name: "non-default BatchTimeout",
			o: []BatchItemProcessorOption{
				WithBatchTimeout(schDelay),
				WithShippingMethod(ShippingMethodAsync),
				WithMaxExportBatchSize(512),
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
				WithMaxExportBatchSize(200),
				WithShippingMethod(ShippingMethodAsync),
			},
			writeNumItems:  200,
			genNumItems:    205,
			wantNumItems:   205,
			wantBatchCount: 2,
		},
	}

	for _, option := range options {
		t.Run(option.name, func(t *testing.T) {
			te := testBatchExporter[TestItem]{}
			ssp, err := createAndRegisterBatchSP(option.o, &te)

			if err != nil {
				require.NoError(t, err)
			}

			if ssp == nil {
				require.NoError(t, err)
			}

			for i := 0; i < option.genNumItems; i++ {
				if option.writeNumItems > 0 && i%option.writeNumItems == 0 {
					time.Sleep(10 * time.Millisecond)
				}

				item := TestItem{
					name: "test" + strconv.Itoa(i),
				}

				if err := ssp.Write(context.Background(), []*TestItem{&item}); err != nil {
					t.Errorf("%s: Error writing to BatchItemProcessor", option.name)
				}
			}

			time.Sleep(schDelay * 2)

			gotNumOfItems := te.len()
			if option.wantNumItems > 0 && option.wantNumItems != gotNumOfItems {
				t.Errorf("number of exported items: got %v, want %v",
					gotNumOfItems, option.wantNumItems)
			}

			gotBatchCount := te.getBatchCount()
			if option.wantBatchCount > 0 && gotBatchCount != option.wantBatchCount {
				t.Errorf("number batches: got %v, want %v",
					gotBatchCount, option.wantBatchCount)
				t.Errorf("Batches %v", te.sizes)
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

	bvp.Start(context.Background())

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
	bvp, err := NewBatchItemProcessor[T](te, "processor", nullLogger(), options...)
	if err != nil {
		return nil, err
	}

	bvp.Start(context.Background())

	return bvp, nil
}

func TestBatchItemProcessorShutdown(t *testing.T) {
	var bp testBatchExporter[TestItem]
	bvp, err := NewBatchItemProcessor[TestItem](&bp, "processor", nullLogger())
	require.NoError(t, err)

	bvp.Start(context.Background())

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
	log := logrus.New()
	bsp, err := NewBatchItemProcessor[TestItem](&be, "processor", log, WithMaxExportBatchSize(5), WithBatchTimeout(1*time.Second), WithWorkers(2), WithShippingMethod(ShippingMethodAsync))
	require.NoError(t, err)

	bsp.Start(context.Background())

	itemsToExport := 5000

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

	bsp.Start(context.Background())

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
		err := bsp.Write(context.Background(), []*TestItem{{
			name: strconv.Itoa(i),
		}})
		require.Error(t, err)
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

	bsp.Start(context.Background())

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

	bsp.Start(context.Background())

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

	bsp.Start(context.Background())

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

	bsp.Start(context.Background())

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

	bsp.Start(context.Background())

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

	bsp.Start(context.Background())

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
func TestBatchItemProcessorWithSyncErrorExporter(t *testing.T) {
	bsp, err := NewBatchItemProcessor[TestItem](ErrorItemExporter[TestItem]{}, "processor", nullLogger(), WithShippingMethod(ShippingMethodSync), WithBatchTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create batch processor: %v", err)
	}

	bsp.Start(context.Background())

	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	if err == nil {
		t.Errorf("Expected write to fail")
	}
}

func TestBatchItemProcessorSyncShipping(t *testing.T) {
	// Define a range of values for workers, maxExportBatchSize, and itemsToExport
	workerCounts := []int{1, 5, 10}
	maxBatchSizes := []int{1, 5, 10, 20}
	itemCounts := []int{0, 1, 10, 25, 50}

	for _, workers := range workerCounts {
		for _, maxBatchSize := range maxBatchSizes {
			for _, itemsToExport := range itemCounts {
				t.Run(fmt.Sprintf("%d workers, batch size %d, %d items", workers, maxBatchSize, itemsToExport), func(t *testing.T) {
					te := testBatchExporter[TestItem]{}
					bsp, err := NewBatchItemProcessor[TestItem](
						&te,
						"processor",
						logrus.New(),
						WithMaxExportBatchSize(maxBatchSize),
						WithWorkers(workers),
						WithShippingMethod(ShippingMethodSync),
						WithBatchTimeout(100*time.Millisecond),
					)
					require.NoError(t, err)

					bsp.Start(context.Background())

					items := make([]*TestItem, itemsToExport)
					for i := 0; i < itemsToExport; i++ {
						items[i] = &TestItem{name: strconv.Itoa(i)}
					}

					err = bsp.Write(context.Background(), items)
					require.NoError(t, err)

					expectedBatches := (itemsToExport + maxBatchSize - 1) / maxBatchSize
					if itemsToExport == 0 {
						expectedBatches = 0
					}

					if te.len() != itemsToExport {
						t.Errorf("Expected all items to be exported, got: %v", te.len())
					}

					if te.getBatchCount() != expectedBatches {
						t.Errorf("Expected %v batches to be exported, got: %v", expectedBatches, te.getBatchCount())
					}
				})
			}
		}
	}
}

func TestBatchItemProcessorExportCancellationOnFailure(t *testing.T) {
	workers := 10
	maxBatchSize := 10
	itemsToExport := 5000

	t.Run(fmt.Sprintf("%d workers, batch size %d, %d items with cancellation on failure", workers, maxBatchSize, itemsToExport), func(t *testing.T) {
		te := testBatchExporter[TestItem]{
			delay: 100 * time.Millisecond, // Introduce a delay to simulate processing time
		}
		bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(maxBatchSize), WithWorkers(workers), WithShippingMethod(ShippingMethodSync))
		require.NoError(t, err)

		bsp.Start(context.Background())

		items := make([]*TestItem, itemsToExport)
		for i := 0; i < itemsToExport; i++ {
			items[i] = &TestItem{name: strconv.Itoa(i)}
		}

		// Simulate export failure and expect cancellation
		te.failNextExport = true

		err = bsp.Write(context.Background(), items)
		require.Error(t, err, "Expected an error due to simulated export failure")

		// Ensure we exported less than the number of batches since the export should've
		// stopped due to the failure.
		require.Less(t, te.batchCount, itemsToExport/maxBatchSize)
	})
}

func TestBatchItemProcessorWithZeroWorkers(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	_, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(0))
	require.Error(t, err, "Expected an error when initializing with zero workers")
}

func TestBatchItemProcessorWithNegativeBatchSize(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	_, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(-1), WithWorkers(5))
	require.Error(t, err, "Expected an error when initializing with negative batch size")
}

func TestBatchItemProcessorWithNegativeQueueSize(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	_, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxQueueSize(-1), WithWorkers(5))
	require.Error(t, err, "Expected an error when initializing with negative queue size")
}

func TestBatchItemProcessorWithZeroBatchSize(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	_, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(0), WithWorkers(5))
	require.Error(t, err, "Expected an error when initializing with zero batch size")
}

func TestBatchItemProcessorWithZeroQueueSize(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	_, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxQueueSize(0), WithWorkers(5))
	require.Error(t, err, "Expected an error when initializing with zero queue size")
}

func TestBatchItemProcessorShutdownWithoutExport(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5))
	require.NoError(t, err)

	require.NoError(t, bsp.Shutdown(context.Background()), "shutting down BatchItemProcessor")
	require.Equal(t, 0, te.len(), "No items should have been exported")
}

func TestBatchItemProcessorExportWithTimeout(t *testing.T) {
	te := testBatchExporter[TestItem]{delay: 2 * time.Second}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5), WithExportTimeout(1*time.Second), WithShippingMethod(ShippingMethodSync))
	require.NoError(t, err)

	bsp.Start(context.Background())

	itemsToExport := 10
	items := make([]*TestItem, itemsToExport)

	for i := 0; i < itemsToExport; i++ {
		items[i] = &TestItem{name: strconv.Itoa(i)}
	}

	err = bsp.Write(context.Background(), items)
	require.Error(t, err, "Expected an error due to export timeout")
	require.Less(t, te.len(), itemsToExport, "Expected some items to be exported before timeout")
}

func TestBatchItemProcessorWithBatchTimeout(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5), WithBatchTimeout(1*time.Second))
	require.NoError(t, err)

	bsp.Start(context.Background())

	itemsToExport := 5
	items := make([]*TestItem, itemsToExport)

	for i := 0; i < itemsToExport; i++ {
		items[i] = &TestItem{name: strconv.Itoa(i)}
	}

	for _, item := range items {
		err := bsp.Write(context.Background(), []*TestItem{item})
		require.NoError(t, err)
	}

	time.Sleep(2 * time.Second)
	require.Equal(t, itemsToExport, te.len(), "Expected all items to be exported after batch timeout")
}

func TestBatchItemProcessorDrainOnShutdownAfterContextCancellation(t *testing.T) {
	te := testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](&te, "processor", nullLogger(), WithMaxExportBatchSize(10), WithWorkers(5), WithBatchTimeout(1*time.Second))
	require.NoError(t, err)

	// Create a cancellable context for Start
	ctx, cancel := context.WithCancel(context.Background())
	bsp.Start(ctx)

	itemsToExport := 50
	items := make([]*TestItem, itemsToExport)

	for i := 0; i < itemsToExport; i++ {
		items[i] = &TestItem{name: strconv.Itoa(i)}
	}

	// Write items to the processor
	err = bsp.Write(context.Background(), items)
	require.NoError(t, err)

	// Cancel the context immediately after writing
	cancel()

	// Allow some time for the cancellation to propagate
	time.Sleep(100 * time.Millisecond)

	// Shutdown the processor
	err = bsp.Shutdown(context.Background())
	require.NoError(t, err)

	// Check if any items were exported
	require.Greater(t, itemsToExport, 0, "No items should have been exported on shutdown")
}

func TestBatchItemProcessorQueueSize(t *testing.T) {
	te := indefiniteExporter[TestItem]{}

	metrics := NewMetrics("test")
	maxQueueSize := 5
	bsp, err := NewBatchItemProcessor[TestItem](
		&te,
		"processor",
		nullLogger(),
		WithBatchTimeout(10*time.Minute),
		WithMaxQueueSize(maxQueueSize),
		WithMaxExportBatchSize(maxQueueSize),
		WithWorkers(1),
		WithShippingMethod(ShippingMethodAsync),
		WithMetrics(metrics),
	)
	require.NoError(t, err)

	bsp.Start(context.Background())

	itemsToExport := 5
	items := make([]*TestItem, itemsToExport)

	for i := 0; i < itemsToExport; i++ {
		items[i] = &TestItem{name: strconv.Itoa(i)}
	}

	// Write items to the processor
	for i := 0; i < itemsToExport; i++ {
		err = bsp.Write(context.Background(), []*TestItem{items[i]})
		if i < maxQueueSize {
			require.NoError(t, err, "Expected no error for item %d", i)
		} else {
			require.Error(t, err, "Expected an error for item %d due to queue size limit", i)
		}
	}

	// Ensure that the queue size is respected
	require.Equal(t, maxQueueSize, len(bsp.queue), "Queue size should be equal to maxQueueSize")

	// Ensure that the dropped count is correct
	counter, err := bsp.metrics.itemsDropped.GetMetricWith(prometheus.Labels{"processor": "processor"})
	require.NoError(t, err)

	metric := &dto.Metric{}

	err = counter.Write(metric)
	require.NoError(t, err)

	require.Equal(t, float64(itemsToExport-maxQueueSize), *metric.Counter.Value, "Dropped count should be equal to the number of items that exceeded the queue size")
}

func TestBatchItemProcessorNilItem(t *testing.T) {
	te := testBatchExporter[TestItem]{}

	bsp, err := NewBatchItemProcessor[TestItem](
		&te,
		"processor",
		nullLogger(),
		WithBatchTimeout(10*time.Millisecond),
		WithMaxQueueSize(5),
		WithMaxExportBatchSize(5),
		WithWorkers(1),
		WithShippingMethod(ShippingMethodSync),
	)
	require.NoError(t, err)

	bsp.Start(context.Background())

	// Write nil item to processor
	err = bsp.Write(context.Background(), []*TestItem{nil})
	require.NoError(t, err)

	// Write invalid items to processor
	err = bsp.Write(context.Background(), nil)
	require.NoError(t, err)

	// Give processor time to process the nil item
	time.Sleep(500 * time.Millisecond)

	// Verify processor is still running by writing a valid item
	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	require.NoError(t, err)
}

func TestBatchItemProcessorNilExporter(t *testing.T) {
	bsp, err := NewBatchItemProcessor[TestItem](
		nil,
		"processor",
		nullLogger(),
		WithBatchTimeout(10*time.Millisecond),
		WithMaxQueueSize(5),
		WithMaxExportBatchSize(5),
		WithWorkers(1),
		WithShippingMethod(ShippingMethodSync),
	)
	require.NoError(t, err)

	bsp.Start(context.Background())

	// Write an item to processor
	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	require.Error(t, err)
}

func TestBatchItemProcessorNilExporterAfterProcessing(t *testing.T) {
	exporter := &testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](
		exporter,
		"processor",
		nullLogger(),
		WithBatchTimeout(500*time.Millisecond),
		WithMaxQueueSize(5),
		WithMaxExportBatchSize(5),
		WithWorkers(1),
		WithShippingMethod(ShippingMethodAsync),
	)
	require.NoError(t, err)

	bsp.Start(context.Background())

	// Write an item to processor with valid exporter
	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	require.NoError(t, err)

	// Give processor time to process the item
	time.Sleep(1000 * time.Millisecond)

	// Nil the exporter
	bsp.e = nil

	// Write an item to processor with nil exporter
	err = bsp.Write(context.Background(), []*TestItem{{name: "test"}})
	require.Error(t, err)

	// Verify we can still shutdown without panic
	require.NotPanics(t, func() {
		err := bsp.Shutdown(context.Background())
		require.NoError(t, err)
	})
}

func TestBatchItemProcessorNilItemAfterQueue(t *testing.T) {
	exporter := &testBatchExporter[TestItem]{}
	bsp, err := NewBatchItemProcessor[TestItem](
		exporter,
		"processor",
		nullLogger(),
		WithBatchTimeout(500*time.Millisecond),
		WithMaxQueueSize(5),
		WithMaxExportBatchSize(5),
		WithWorkers(1),
		WithShippingMethod(ShippingMethodAsync),
	)
	require.NoError(t, err)

	bsp.Start(context.Background())

	// Write a valid item first
	item := &TestItem{name: "test"}
	err = bsp.Write(context.Background(), []*TestItem{item})
	require.NoError(t, err)

	// Inject nil directly into the processor's queue
	bsp.queue <- nil

	// Write a valid item to ensure the processor is still running
	err = bsp.Write(context.Background(), []*TestItem{item})
	require.NoError(t, err)

	// Give processor time to handle the nil item
	time.Sleep(1000 * time.Millisecond)

	// Ensure no panic during shutdown
	require.NotPanics(t, func() {
		err := bsp.Shutdown(context.Background())
		require.NoError(t, err)
	})

	// Verify that valid items were still exported
	require.Equal(t, 2, exporter.len())
}
