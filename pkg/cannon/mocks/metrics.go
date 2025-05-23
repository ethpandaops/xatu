package mocks

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

// MockMetrics provides a mock implementation that doesn't register with Prometheus
type MockMetrics struct {
	eventCount int
}

// NewMockMetrics creates a new mock metrics instance
func NewMockMetrics() *MockMetrics {
	return &MockMetrics{}
}

// AddDecoratedEvent mocks the metric recording without actually doing it
func (m *MockMetrics) AddDecoratedEvent(count int, eventType *xatu.DecoratedEvent, network string) {
	m.eventCount += count
}

// GetEventCount returns the mock event count for testing
func (m *MockMetrics) GetEventCount() int {
	return m.eventCount
}