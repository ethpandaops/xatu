package processor

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// TestEnqueueOrDropReportsDropOnClosedQueue verifies that enqueueOrDrop reports an error
// and records a drop when the queue send panics on a closed channel during shutdown.
// Previously the panic was recovered into an unnamed return value, so the function
// returned nil and the caller treated a lost item as successfully enqueued.
func TestEnqueueOrDropReportsDropOnClosedQueue(t *testing.T) {
	metrics := NewMetrics("enqueue_closed_queue_test")

	be := testBatchExporter[TestItem]{}
	bvp, err := NewBatchItemProcessor[TestItem](&be, "processor", nullLogger(), WithMetrics(metrics))
	if err != nil {
		t.Fatalf("constructor: %v", err)
	}

	droppedBefore := droppedCount(t, metrics, "processor")

	// Reproduce the shutdown state: the stopCh check has already passed and the queue is
	// now closed, so the send inside enqueueOrDrop panics.
	close(bvp.queue)

	item := &TraceableItem[TestItem]{item: &TestItem{name: "x"}}

	if err := bvp.enqueueOrDrop(context.Background(), item); err == nil {
		t.Fatal("enqueueOrDrop returned nil after the queue was closed; a dropped item must be reported as an error")
	}

	if got, want := droppedCount(t, metrics, "processor"), droppedBefore+1; got != want {
		t.Fatalf("items dropped counter = %v, want %v", got, want)
	}
}

func droppedCount(t *testing.T, m *Metrics, processor string) float64 {
	t.Helper()

	c, err := m.itemsDropped.GetMetricWith(prometheus.Labels{"processor": processor})
	if err != nil {
		t.Fatalf("metric: %v", err)
	}

	metric := &dto.Metric{}
	if err := c.Write(metric); err != nil {
		t.Fatalf("metric write: %v", err)
	}

	if metric.Counter == nil || metric.Counter.Value == nil {
		return 0
	}

	return *metric.Counter.Value
}
