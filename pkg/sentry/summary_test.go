package sentry

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
)

const (
	summaryHeadTopic  = "head"
	summaryBlockTopic = "block"
)

func TestSummaryEventStreamCounts(t *testing.T) {
	t.Parallel()

	summary := NewSummary(nil, time.Minute, nil)
	summary.AddEventStreamEvents(summaryHeadTopic, 1)
	require.Equal(t, map[string]uint64{summaryHeadTopic: 1}, summary.GetEventStreamEvents())

	summary.AddEventStreamEvents(summaryHeadTopic, 3)
	summary.AddEventStreamEvents(summaryBlockTopic, 2)
	require.Equal(t, map[string]uint64{summaryHeadTopic: 4, summaryBlockTopic: 2}, summary.GetEventStreamEvents())

	// A caller must not be able to mutate the live counters through a getter.
	events := summary.GetEventStreamEvents()
	events[summaryHeadTopic] = 100
	delete(events, summaryBlockTopic)
	require.Equal(t, map[string]uint64{summaryHeadTopic: 4, summaryBlockTopic: 2}, summary.GetEventStreamEvents())

	summary.AddEventsExported(3)
	summary.AddFailedEvents(2)
	summary.Reset()
	require.Empty(t, summary.GetEventStreamEvents())
	require.Zero(t, summary.GetEventsExported())
	require.Zero(t, summary.GetFailedEvents())

	summary.AddEventStreamEvents(summaryHeadTopic, 1)
	require.Equal(t, map[string]uint64{summaryHeadTopic: 1}, summary.GetEventStreamEvents())
}

func TestSummaryConcurrentIncrements(t *testing.T) {
	t.Parallel()

	const (
		workers    = 16
		iterations = 1000
	)

	summary := &Summary{}
	start := make(chan struct{})

	var wg sync.WaitGroup

	for range workers {
		wg.Go(func() {
			<-start

			for range iterations {
				summary.AddEventStreamEvents(summaryHeadTopic, 1)
				summary.AddEventStreamEvents(summaryBlockTopic, 2)
				summary.AddEventsExported(3)
				summary.AddFailedEvents(4)
			}
		})
	}

	close(start)
	wg.Wait()

	require.Equal(t, map[string]uint64{
		summaryHeadTopic:  workers * iterations,
		summaryBlockTopic: 2 * workers * iterations,
	}, summary.GetEventStreamEvents())
	require.EqualValues(t, 3*workers*iterations, summary.GetEventsExported())
	require.EqualValues(t, 4*workers*iterations, summary.GetFailedEvents())
}

func TestSummaryConcurrentReset(t *testing.T) {
	t.Parallel()

	summary := &Summary{}
	start := make(chan struct{})

	var wg sync.WaitGroup

	wg.Go(func() {
		<-start

		for range 1000 {
			summary.AddEventStreamEvents(summaryHeadTopic, 1)
			summary.AddEventsExported(1)
			summary.AddFailedEvents(1)
		}
	})
	wg.Go(func() {
		<-start

		for range 1000 {
			summary.Reset()
		}
	})
	wg.Go(func() {
		<-start

		for range 1000 {
			summary.GetEventStreamEvents()
			summary.GetEventsExported()
			summary.GetFailedEvents()
		}
	})

	close(start)
	wg.Wait()

	summary.Reset()
	require.Empty(t, summary.GetEventStreamEvents())
	require.Zero(t, summary.GetEventsExported())
	require.Zero(t, summary.GetFailedEvents())
}

func TestSummaryConcurrentIntervals(t *testing.T) {
	t.Parallel()

	const (
		workers     = 8
		intervals   = 100
		perInterval = 50
	)

	summary := &Summary{}

	var (
		ready, finished               sync.WaitGroup
		head, block, exported, failed uint64
	)

	for range intervals {
		start := make(chan struct{})

		for range workers {
			ready.Add(1)
			finished.Go(func() {
				ready.Done()
				<-start

				for range perInterval {
					summary.AddEventStreamEvents(summaryHeadTopic, 1)
					summary.AddEventStreamEvents(summaryBlockTopic, 2)
					summary.AddEventsExported(3)
					summary.AddFailedEvents(4)
				}
			})
		}

		ready.Wait()
		close(start)

		// Snapshot/reset competes with writers released by the same barrier.
		// Every increment must land in either this snapshot or the next one.
		events, intervalExported, intervalFailed := summary.snapshotAndReset()
		head += events[summaryHeadTopic]
		block += events[summaryBlockTopic]
		exported += intervalExported
		failed += intervalFailed

		// Reads must also be safe while callbacks are updating the new map.
		summary.GetEventStreamEvents()
		summary.GetEventsExported()
		summary.GetFailedEvents()
		finished.Wait()
	}

	events, intervalExported, intervalFailed := summary.snapshotAndReset()
	head += events[summaryHeadTopic]
	block += events[summaryBlockTopic]
	exported += intervalExported
	failed += intervalFailed

	require.EqualValues(t, workers*intervals*perInterval, head)
	require.EqualValues(t, 2*workers*intervals*perInterval, block)
	require.EqualValues(t, 3*workers*intervals*perInterval, exported)
	require.EqualValues(t, 4*workers*intervals*perInterval, failed)
	require.Empty(t, summary.GetEventStreamEvents())
}

type summaryLogHook struct {
	callback func(*logrus.Entry)
}

func (h summaryLogHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.InfoLevel}
}

func (h summaryLogHook) Fire(entry *logrus.Entry) error {
	h.callback(entry)

	return nil
}

func TestSummaryPrintPreservesEventsDuringLogging(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Construct an unstarted node; Print only reads its local health status.
	node, err := ethereum.NewBeaconNode(context.Background(), "summary-test", &ethereum.Config{}, logger, &ethereum.Options{})
	require.NoError(t, err)

	summary := NewSummary(logger, time.Minute, node)
	// Pre-create the topic so this test isolates the print/reset boundary from
	// the separate first-insertion counting regression.
	summary.AddEventStreamEvents(summaryHeadTopic, 0)
	summary.AddEventStreamEvents(summaryHeadTopic, 1)
	summary.AddEventsExported(2)
	summary.AddFailedEvents(3)

	logger.AddHook(summaryLogHook{callback: func(entry *logrus.Entry) {
		require.Equal(t, "head: 1", entry.Data["event_stream_events"])
		require.Equal(t, uint64(2), entry.Data["events_exported"])
		require.Equal(t, uint64(3), entry.Data["events_failed"])

		// Re-enter from the log hook to prove logging holds no summary lock.
		summary.AddEventStreamEvents(summaryBlockTopic, 4)
		summary.AddEventsExported(5)
		summary.AddFailedEvents(6)
	}})

	summary.Print()

	require.Equal(t, map[string]uint64{summaryBlockTopic: 4}, summary.GetEventStreamEvents())
	require.Equal(t, uint64(5), summary.GetEventsExported())
	require.Equal(t, uint64(6), summary.GetFailedEvents())
}
