package sentry

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
)

// Summary is a struct that holds the summary of the sentry.
type Summary struct {
	log           observability.ContextualLogger
	printInterval time.Duration

	beacon *ethereum.BeaconNode

	mu                sync.Mutex
	eventStreamEvents map[string]uint64
	eventsExported    uint64
	failedEvents      uint64
}

// NewSummary creates a new summary with the given print interval.
func NewSummary(log observability.ContextualLogger, printInterval time.Duration, beacon *ethereum.BeaconNode) *Summary {
	return &Summary{
		log:           log,
		printInterval: printInterval,
		beacon:        beacon,
	}
}

func (s *Summary) Start(ctx context.Context) {
	s.log.WithField("interval", s.printInterval).WithContext(ctx).Info("Starting summary")
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
	// Close the interval before logging so callbacks during the print are
	// counted in the next interval rather than discarded by a later reset.
	events, eventsExported, failedEvents := s.snapshotAndReset()

	isSyncing := "unknown"
	status := s.beacon.Node().Status()

	if status != nil {
		isSyncing = strconv.FormatBool(status.Syncing())
	}

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
		"events_exported":     eventsExported,
		"events_failed":       failedEvents,
		"node_is_healthy":     s.beacon.Node().Healthy(),
		"node_is_syncing":     isSyncing,
		"event_stream_events": eventStream,
	}).Infof("Summary of the last %s", s.printInterval)
}

func (s *Summary) AddEventsExported(count uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.eventsExported += count
}

func (s *Summary) GetEventsExported() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.eventsExported
}

func (s *Summary) AddFailedEvents(count uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failedEvents += count
}

func (s *Summary) GetFailedEvents() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.failedEvents
}

func (s *Summary) AddEventStreamEvents(topic string, count uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.eventStreamEvents == nil {
		s.eventStreamEvents = make(map[string]uint64)
	}

	s.eventStreamEvents[topic] += count
}

func (s *Summary) GetEventStreamEvents() map[string]uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	events := make(map[string]uint64, len(s.eventStreamEvents))

	for topic, count := range s.eventStreamEvents {
		events[topic] = count
	}

	return events
}

func (s *Summary) Reset() {
	s.snapshotAndReset()
}

// snapshotAndReset transfers the completed interval to the caller. Writers
// only access the new map after the lock is released, so formatting and logging
// the detached snapshot do not block event callbacks.
func (s *Summary) snapshotAndReset() (events map[string]uint64, eventsExported, failedEvents uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	events = s.eventStreamEvents
	eventsExported = s.eventsExported
	failedEvents = s.failedEvents

	s.eventStreamEvents = nil
	s.eventsExported = 0
	s.failedEvents = 0

	return events, eventsExported, failedEvents
}
