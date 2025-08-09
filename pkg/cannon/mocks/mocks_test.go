package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for Config struct
func TestConfig_Structure(t *testing.T) {
	config := Config{
		Name:         "test-cannon",
		LoggingLevel: "debug",
		MetricsAddr:  ":9090",
		NTPServer:    "pool.ntp.org",
		Outputs: []output.Config{
			{
				Name:     "stdout",
				SinkType: output.SinkTypeStdOut,
			},
		},
	}

	assert.Equal(t, "test-cannon", config.Name)
	assert.Equal(t, "debug", config.LoggingLevel)
	assert.Equal(t, ":9090", config.MetricsAddr)
	assert.Equal(t, "pool.ntp.org", config.NTPServer)
	assert.Len(t, config.Outputs, 1)
	assert.Equal(t, "stdout", config.Outputs[0].Name)
}

func TestConfig_ZeroValue(t *testing.T) {
	var config Config

	assert.Empty(t, config.Name)
	assert.Empty(t, config.LoggingLevel)
	assert.Empty(t, config.MetricsAddr)
	assert.Empty(t, config.NTPServer)
	assert.Empty(t, config.Outputs)
}

func TestTestConfig_Factory(t *testing.T) {
	config := TestConfig()

	require.NotNil(t, config)
	assert.Equal(t, "test-cannon", config.Name)
	assert.Equal(t, "info", config.LoggingLevel)
	assert.Equal(t, ":9090", config.MetricsAddr)
	assert.Equal(t, "time.google.com", config.NTPServer)
	assert.Len(t, config.Outputs, 1)
	assert.Equal(t, "stdout", config.Outputs[0].Name)
	assert.Equal(t, output.SinkTypeStdOut, config.Outputs[0].SinkType)
}

func TestTestLogger_Factory(t *testing.T) {
	logger := TestLogger()

	require.NotNil(t, logger)
	assert.IsType(t, &logrus.Entry{}, logger)

	// Should be configured to only log fatal errors
	// This is hard to test directly, but we can verify it's properly configured
	entry, ok := logger.(*logrus.Entry)
	require.True(t, ok)
	assert.Equal(t, logrus.FatalLevel, entry.Logger.Level)
}

// Tests for MockTimeProvider
func TestMockTimeProvider_Now(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MockTimeProvider)
		expected time.Time
	}{
		{
			name: "returns_set_time",
			setup: func(m *MockTimeProvider) {
				fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
				m.SetCurrentTime(fixedTime)
			},
			expected: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "returns_mock_expectation",
			setup: func(m *MockTimeProvider) {
				customTime := time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC)
				m.On("Now").Return(customTime)
			},
			expected: time.Date(2024, 6, 15, 9, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTimeProvider{}
			tt.setup(mock)

			result := mock.Now()
			assert.Equal(t, tt.expected, result)

			mock.AssertExpectations(t)
		})
	}
}

func TestMockTimeProvider_Since(t *testing.T) {
	mock := &MockTimeProvider{}
	pastTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedDuration := time.Hour

	mock.On("Since", pastTime).Return(expectedDuration)

	result := mock.Since(pastTime)
	assert.Equal(t, expectedDuration, result)

	mock.AssertExpectations(t)
}

func TestMockTimeProvider_Until(t *testing.T) {
	mock := &MockTimeProvider{}
	futureTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	expectedDuration := time.Hour * 24

	mock.On("Until", futureTime).Return(expectedDuration)

	result := mock.Until(futureTime)
	assert.Equal(t, expectedDuration, result)

	mock.AssertExpectations(t)
}

func TestMockTimeProvider_Sleep(t *testing.T) {
	mock := &MockTimeProvider{}
	duration := time.Millisecond * 100

	mock.On("Sleep", duration).Return()

	mock.Sleep(duration)

	mock.AssertExpectations(t)
}

func TestMockTimeProvider_After(t *testing.T) {
	mock := &MockTimeProvider{}
	duration := time.Second
	expectedChan := make(<-chan time.Time)

	mock.On("After", duration).Return(expectedChan)

	result := mock.After(duration)
	assert.Equal(t, expectedChan, result)

	mock.AssertExpectations(t)
}

func TestMockTimeProvider_SetCurrentTime(t *testing.T) {
	mock := &MockTimeProvider{}
	fixedTime := time.Date(2023, 5, 10, 14, 30, 0, 0, time.UTC)

	mock.SetCurrentTime(fixedTime)

	// Should return the set time
	result := mock.Now()
	assert.Equal(t, fixedTime, result)

	// Should have set up the mock expectation
	mock.AssertExpectations(t)
}

// Tests for MockNTPClient
func TestMockNTPClient_Query(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		setupMock    func(*MockNTPClient)
		expectError  bool
		expectResult bool
	}{
		{
			name: "successful_query",
			host: "time.google.com",
			setupMock: func(m *MockNTPClient) {
				mockResponse := &MockNTPResponse{}
				m.On("Query", "time.google.com").Return(mockResponse, nil)
			},
			expectError:  false,
			expectResult: true,
		},
		{
			name: "failed_query",
			host: "invalid.host",
			setupMock: func(m *MockNTPClient) {
				m.On("Query", "invalid.host").Return(nil, errors.New("host not found"))
			},
			expectError:  true,
			expectResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockNTPClient{}
			tt.setupMock(mock)

			result, err := mock.Query(tt.host)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.expectResult {
					assert.NotNil(t, result)
				}
			}

			mock.AssertExpectations(t)
		})
	}
}

// Tests for MockNTPResponse
func TestMockNTPResponse_Validate(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockNTPResponse)
		expectError bool
	}{
		{
			name: "validation_passes",
			setupMock: func(m *MockNTPResponse) {
				m.On("Validate").Return(nil)
			},
			expectError: false,
		},
		{
			name: "validation_fails",
			setupMock: func(m *MockNTPResponse) {
				m.On("Validate").Return(errors.New("validation failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockNTPResponse{}
			tt.setupMock(mock)

			err := mock.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mock.AssertExpectations(t)
		})
	}
}

func TestMockNTPResponse_ClockOffset(t *testing.T) {
	mock := &MockNTPResponse{}
	expectedOffset := time.Millisecond * 50

	mock.SetClockOffset(expectedOffset)

	result := mock.ClockOffset()
	assert.Equal(t, expectedOffset, result)

	mock.AssertExpectations(t)
}

func TestMockNTPResponse_SetClockOffset(t *testing.T) {
	mock := &MockNTPResponse{}
	offset := time.Second * 2

	mock.SetClockOffset(offset)

	// Should return the set offset
	result := mock.ClockOffset()
	assert.Equal(t, offset, result)

	// Should have set up the mock expectation
	mock.AssertExpectations(t)
}

// Tests for MockScheduler
func TestMockScheduler_Start(t *testing.T) {
	mock := &MockScheduler{}
	mock.On("Start").Return()

	assert.False(t, mock.IsStarted(), "Should not be started initially")

	mock.Start()

	assert.True(t, mock.IsStarted(), "Should be started after Start() call")
	mock.AssertExpectations(t)
}

func TestMockScheduler_Shutdown(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockScheduler)
		expectError bool
	}{
		{
			name: "successful_shutdown",
			setupMock: func(m *MockScheduler) {
				m.On("Shutdown").Return(nil)
			},
			expectError: false,
		},
		{
			name: "failed_shutdown",
			setupMock: func(m *MockScheduler) {
				m.On("Shutdown").Return(errors.New("shutdown failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockScheduler{}
			mock.isStarted = true // Start it first
			tt.setupMock(mock)

			err := mock.Shutdown()

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, mock.IsStarted(), "Should still be started on error")
			} else {
				assert.NoError(t, err)
				assert.False(t, mock.IsStarted(), "Should be stopped after successful shutdown")
			}

			mock.AssertExpectations(t)
		})
	}
}

func TestMockScheduler_NewJob(t *testing.T) {
	mock := &MockScheduler{}
	jobDef := "every 5 minutes"
	task := "backup"

	// Test with empty options slice (variadic parameter with no options)
	mock.On("NewJob", jobDef, task, []interface{}(nil)).Return("job-id", nil)

	result, err := mock.NewJob(jobDef, task)

	assert.NoError(t, err)
	assert.Equal(t, "job-id", result)

	mock.AssertExpectations(t)
}

// Tests for MockSink
func TestNewMockSink(t *testing.T) {
	sink := NewMockSink("test-sink", "stdout")

	assert.Equal(t, "test-sink", sink.Name())
	assert.Equal(t, "stdout", sink.Type())
	assert.False(t, sink.IsStarted())
}

func TestMockSink_StartStop(t *testing.T) {
	sink := NewMockSink("test-sink", "stdout")
	ctx := context.Background()

	// Test successful start
	sink.On("Start", ctx).Return(nil)
	err := sink.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, sink.IsStarted())

	// Test successful stop
	sink.On("Stop", ctx).Return(nil)
	err = sink.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, sink.IsStarted())

	sink.AssertExpectations(t)
}

func TestMockSink_StartError(t *testing.T) {
	sink := NewMockSink("test-sink", "stdout")
	ctx := context.Background()

	sink.On("Start", ctx).Return(errors.New("start failed"))

	err := sink.Start(ctx)
	assert.Error(t, err)
	assert.False(t, sink.IsStarted(), "Should not be started on error")

	sink.AssertExpectations(t)
}

func TestMockSink_HandleNewDecoratedEvent(t *testing.T) {
	sink := NewMockSink("test-sink", "stdout")
	ctx := context.Background()
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
		},
	}

	sink.On("HandleNewDecoratedEvent", ctx, event).Return(nil)

	err := sink.HandleNewDecoratedEvent(ctx, event)
	assert.NoError(t, err)

	sink.AssertExpectations(t)
}

func TestMockSink_HandleNewDecoratedEvents(t *testing.T) {
	sink := NewMockSink("test-sink", "stdout")
	ctx := context.Background()
	events := []*xatu.DecoratedEvent{
		{
			Event: &xatu.Event{
				Name: xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
			},
		},
	}

	sink.On("HandleNewDecoratedEvents", ctx, events).Return(nil)

	err := sink.HandleNewDecoratedEvents(ctx, events)
	assert.NoError(t, err)

	sink.AssertExpectations(t)
}

// Tests for TestAssertions
func TestNewTestAssertions(t *testing.T) {
	assertions := NewTestAssertions(t)

	assert.NotNil(t, assertions)
	assert.Equal(t, t, assertions.t)
}

func TestTestAssertions_AssertMockExpectations(t *testing.T) {
	assertions := NewTestAssertions(t)

	// Create some mocks
	mock1 := &MockTimeProvider{}
	mock2 := &MockNTPClient{}

	// Set up expectations
	mock1.On("Now").Return(time.Now())
	mock2.On("Query", "test").Return(nil, errors.New("test"))

	// Call the methods to satisfy expectations
	mock1.Now()
	_, _ = mock2.Query("test")

	// Should not panic when expectations are met
	assertions.AssertMockExpectations(mock1, mock2)
}

func TestTestAssertions_AssertCannonStarted(t *testing.T) {
	assertions := NewTestAssertions(t)

	// Should not panic with non-nil cannon
	cannon := &struct{ started bool }{started: true}
	assertions.AssertCannonStarted(cannon)
}

func TestTestAssertions_AssertCannonStopped(t *testing.T) {
	assertions := NewTestAssertions(t)

	// Should not panic with non-nil cannon
	cannon := &struct{ started bool }{started: false}
	assertions.AssertCannonStopped(cannon)
}

// Integration test for multiple mock components
func TestMockIntegration(t *testing.T) {
	// Create all mocks
	timeProvider := &MockTimeProvider{}
	ntpClient := &MockNTPClient{}
	ntpResponse := &MockNTPResponse{}
	scheduler := &MockScheduler{}
	sink := NewMockSink("integration-sink", "stdout")

	// Set up a realistic interaction
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	timeProvider.SetCurrentTime(fixedTime)

	ntpResponse.SetClockOffset(time.Millisecond * 10)
	ntpClient.On("Query", "time.google.com").Return(ntpResponse, nil)

	scheduler.On("Start").Return()
	scheduler.On("NewJob", "sync", "task", []interface{}(nil)).Return("job-1", nil)

	ctx := context.Background()
	sink.On("Start", ctx).Return(nil)

	// Execute the integration
	now := timeProvider.Now()
	assert.Equal(t, fixedTime, now)

	response, err := ntpClient.Query("time.google.com")
	require.NoError(t, err)
	require.NotNil(t, response)

	offset := response.ClockOffset()
	assert.Equal(t, time.Millisecond*10, offset)

	scheduler.Start()
	assert.True(t, scheduler.IsStarted())

	job, err := scheduler.NewJob("sync", "task")
	require.NoError(t, err)
	assert.Equal(t, "job-1", job)

	err = sink.Start(ctx)
	require.NoError(t, err)
	assert.True(t, sink.IsStarted())

	// Verify all expectations
	assertions := NewTestAssertions(t)
	assertions.AssertMockExpectations(timeProvider, ntpClient, ntpResponse, scheduler, sink)
}
