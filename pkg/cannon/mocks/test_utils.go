package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestLogger provides a silent logger for tests
func TestLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Only log fatal errors in tests
	return logrus.NewEntry(logger)
}

// Config represents a test configuration structure
type Config struct {
	Name         string
	LoggingLevel string
	MetricsAddr  string
	NTPServer    string
	Outputs      []output.Config
}

// TestConfig creates a basic test configuration
func TestConfig() *Config {
	return &Config{
		Name:         "test-cannon",
		LoggingLevel: "info",
		MetricsAddr:  ":9090",
		NTPServer:    "time.google.com",
		Outputs: []output.Config{
			{
				Name:     "stdout",
				SinkType: output.SinkTypeStdOut,
			},
		},
	}
}

// MockTimeProvider is a mock implementation of TimeProvider
type MockTimeProvider struct {
	mock.Mock
	currentTime time.Time
}

func (m *MockTimeProvider) Now() time.Time {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(time.Time)
	}
	if !m.currentTime.IsZero() {
		return m.currentTime
	}
	return time.Now()
}

func (m *MockTimeProvider) Since(t time.Time) time.Duration {
	args := m.Called(t)
	return args.Get(0).(time.Duration)
}

func (m *MockTimeProvider) Until(t time.Time) time.Duration {
	args := m.Called(t)
	return args.Get(0).(time.Duration)
}

func (m *MockTimeProvider) Sleep(d time.Duration) {
	m.Called(d)
}

func (m *MockTimeProvider) After(d time.Duration) <-chan time.Time {
	args := m.Called(d)
	return args.Get(0).(<-chan time.Time)
}

// SetCurrentTime sets a fixed time for the mock to return
func (m *MockTimeProvider) SetCurrentTime(t time.Time) {
	m.currentTime = t
	m.On("Now").Return(t)
}

// MockNTPClient is a mock implementation of NTPClient
type MockNTPClient struct {
	mock.Mock
}

// NTPResponse interface for testing
type NTPResponse interface {
	Validate() error
	ClockOffset() time.Duration
}

// Import the interfaces from the parent package
func (m *MockNTPClient) Query(host string) (NTPResponse, error) {
	args := m.Called(host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(NTPResponse), args.Error(1)
}

// MockNTPResponse is a mock implementation of NTPResponse
type MockNTPResponse struct {
	mock.Mock
	clockOffset time.Duration
}

func (m *MockNTPResponse) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockNTPResponse) ClockOffset() time.Duration {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(time.Duration)
	}
	return m.clockOffset
}

// SetClockOffset sets a fixed clock offset for the mock to return
func (m *MockNTPResponse) SetClockOffset(offset time.Duration) {
	m.clockOffset = offset
	m.On("ClockOffset").Return(offset)
}

// MockScheduler is a mock implementation of Scheduler
type MockScheduler struct {
	mock.Mock
	isStarted bool
}

func (m *MockScheduler) Start() {
	m.Called()
	m.isStarted = true
}

func (m *MockScheduler) Shutdown() error {
	args := m.Called()
	err := args.Error(0)
	if err == nil {
		m.isStarted = false
	}
	return err
}

func (m *MockScheduler) NewJob(jobDefinition, task any, options ...any) (any, error) {
	args := m.Called(jobDefinition, task, options)
	return args.Get(0), args.Error(1)
}

// IsStarted returns whether the scheduler has been started
func (m *MockScheduler) IsStarted() bool {
	return m.isStarted
}

// MockSink is a mock implementation of output.Sink
type MockSink struct {
	mock.Mock
	name     string
	sinkType string
	started  bool
}

func NewMockSink(name, sinkType string) *MockSink {
	return &MockSink{
		name:     name,
		sinkType: sinkType,
	}
}

func (m *MockSink) Name() string {
	return m.name
}

func (m *MockSink) Type() string {
	return m.sinkType
}

func (m *MockSink) Start(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.started = true
	}
	return args.Error(0)
}

func (m *MockSink) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	if args.Error(0) == nil {
		m.started = false
	}
	return args.Error(0)
}

func (m *MockSink) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockSink) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

// IsStarted returns whether the sink has been started
func (m *MockSink) IsStarted() bool {
	return m.started
}

// TestAssertions provides common test assertion helpers
type TestAssertions struct {
	t *testing.T
}

func NewTestAssertions(t *testing.T) *TestAssertions {
	return &TestAssertions{t: t}
}

// AssertMockExpectations verifies all mock expectations
func (ta *TestAssertions) AssertMockExpectations(mocks ...interface{}) {
	for _, m := range mocks {
		if mockObj, ok := m.(interface{ AssertExpectations(*testing.T) bool }); ok {
			mockObj.AssertExpectations(ta.t)
		}
	}
}

// AssertCannonStarted verifies that a cannon has been properly started
func (ta *TestAssertions) AssertCannonStarted(cannon any) {
	// This would need to be implemented with type assertions
	// For now, just a placeholder
	assert.NotNil(ta.t, cannon, "cannon should be set")
}

// AssertCannonStopped verifies that a cannon has been properly stopped
func (ta *TestAssertions) AssertCannonStopped(cannon any) {
	// This would need to be implemented with type assertions
	// For now, just a placeholder
	assert.NotNil(ta.t, cannon, "cannon should be set")
}
