package sentry

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	"github.com/sirupsen/logrus"
)

// Summary is a struct that holds the summary of the sentry.
type Summary struct {
	log           logrus.FieldLogger
	printInterval time.Duration

	beacon *ethereum.BeaconNode

	eventStreamEvents sync.Map
	eventsExported    atomic.Uint64
	failedEvents      atomic.Uint64
}

// NewSummary creates a new summary with the given print interval.
func NewSummary(log logrus.FieldLogger, printInterval time.Duration, beacon *ethereum.BeaconNode) *Summary {
	return &Summary{
		log:           log,
		printInterval: printInterval,
		beacon:        beacon,
	}
}

func (s *Summary) Start(ctx context.Context) {
	s.log.WithField("interval", s.printInterval).Info("Starting summary")
	ticker := time.NewTicker(s.printInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Print()
		}
	}
}

func (s *Summary) Print() {
	isSyncing := "unknown"
	status := s.beacon.Node().Status()

	if status != nil {
		isSyncing = strconv.FormatBool(status.Syncing())
	}

	events := s.GetEventStreamEvents()

	// Build a sorted slice of event stream topics and counts
	type topicCount struct {
		topic string
		count uint64
	}

	sortedEvents := make([]topicCount, 0, len(events))
	for topic, count := range events {
		sortedEvents = append(sortedEvents, topicCount{topic, count})
	}

	sort.Slice(sortedEvents, func(i, j int) bool {
		return sortedEvents[i].count > sortedEvents[j].count
	})

	// Create formatted strings for each topic and count
	eventTopics := make([]string, len(sortedEvents))
	for i, tc := range sortedEvents {
		eventTopics[i] = fmt.Sprintf("%s: %d", tc.topic, tc.count)
	}

	eventStream := strings.Join(eventTopics, ", ")

	s.log.WithFields(logrus.Fields{
		"events_exported":     s.GetEventsExported(),
		"events_failed":       s.GetFailedEvents(),
		"node_is_healthy":     s.beacon.Node().Healthy(),
		"node_is_syncing":     isSyncing,
		"event_stream_events": eventStream,
	}).Infof("Summary of the last %s", s.printInterval)

	s.Reset()
}

func (s *Summary) AddEventsExported(count uint64) {
	s.eventsExported.Add(count)
}

func (s *Summary) GetEventsExported() uint64 {
	return s.eventsExported.Load()
}

func (s *Summary) AddFailedEvents(count uint64) {
	s.failedEvents.Add(count)
}

func (s *Summary) GetFailedEvents() uint64 {
	return s.failedEvents.Load()
}

func (s *Summary) AddEventStreamEvents(topic string, count uint64) {
	current, _ := s.eventStreamEvents.LoadOrStore(topic, count)

	s.eventStreamEvents.Store(topic, current.(uint64)+count)
}

func (s *Summary) GetEventStreamEvents() map[string]uint64 {
	events := make(map[string]uint64)

	s.eventStreamEvents.Range(func(key, value any) bool {
		events[key.(string)] = value.(uint64)

		return true
	})

	return events
}

func (s *Summary) Reset() {
	s.eventsExported.Store(0)
	s.failedEvents.Store(0)
	s.eventStreamEvents = sync.Map{}
}
