package cannon

import (
	"context"
	"errors"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/cannon/mocks"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCannonFactory_Build(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *CannonFactory
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *TestableCannon)
	}{
		{
			name: "successful_build_with_all_components",
			setup: func() *CannonFactory {
				mockBeacon := &mocks.MockBeaconNode{}
				mockCoordinator := &mocks.MockCoordinator{}
				mockScheduler := &mocks.MockScheduler{}
				mockTimeProvider := &mocks.MockTimeProvider{}

				config := &Config{
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
				return NewTestCannonFactory().
					WithConfig(config).
					WithLogger(mocks.TestLogger()).
					WithBeaconNode(mockBeacon).
					WithCoordinator(mockCoordinator).
					WithScheduler(mockScheduler).
					WithTimeProvider(mockTimeProvider).
					WithID(uuid.New())
			},
			expectError: false,
			validate: func(t *testing.T, cannon *TestableCannon) {
				assert.NotNil(t, cannon.GetBeacon())
				assert.NotNil(t, cannon.GetCoordinator())
				assert.NotNil(t, cannon.GetScheduler())
				assert.NotNil(t, cannon.GetTimeProvider())
				assert.NotNil(t, cannon.GetNTPClient())
			},
		},
		{
			name: "build_fails_without_config",
			setup: func() *CannonFactory {
				return NewTestCannonFactory()
			},
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "build_with_defaults_creates_missing_components",
			setup: func() *CannonFactory {
				config := &Config{
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
				return NewTestCannonFactory().
					WithConfig(config).
					WithLogger(mocks.TestLogger())
			},
			expectError: false,
			validate: func(t *testing.T, cannon *TestableCannon) {
				assert.NotNil(t, cannon.GetScheduler())
				assert.NotNil(t, cannon.GetTimeProvider())
				assert.NotNil(t, cannon.GetNTPClient())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := tt.setup()
			
			cannon, err := factory.Build()
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, cannon)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cannon)
				if tt.validate != nil {
					tt.validate(t, cannon)
				}
			}
		})
	}
}

func TestTestableCannon_Start(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler)
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *TestableCannon)
	}{
		{
			name: "successful_start",
			setupMocks: func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler) {
				mockBeacon := &mocks.MockBeaconNode{}
				mockCoordinator := &mocks.MockCoordinator{}
				mockScheduler := &mocks.MockScheduler{}

				// Only beacon.Start() and scheduler.Start() are called in the actual implementation
				mockBeacon.SetupStartSuccess()
				mockScheduler.On("Start").Return()
				// Note: coordinator.Start() is NOT called in the current implementation

				return mockBeacon, mockCoordinator, mockScheduler
			},
			expectError: false,
			validate: func(t *testing.T, cannon *TestableCannon) {
				// Add validation for started state
				assert.NotNil(t, cannon.GetBeacon())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBeacon, mockCoordinator, mockScheduler := tt.setupMocks()

			config := &Config{
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

			factory := NewTestCannonFactory().
				WithConfig(config).
				WithLogger(mocks.TestLogger()).
				WithBeaconNode(mockBeacon).
				WithCoordinator(mockCoordinator).
				WithScheduler(mockScheduler).
				WithTimeProvider(&mocks.MockTimeProvider{}).
				WithSinks([]output.Sink{mocks.NewMockSink("test", "stdout")})

			cannon, err := factory.Build()
			require.NoError(t, err)

			err = cannon.Start(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, cannon)
				}
			}

			// Verify mock expectations only for components that should be called
			mockBeacon.AssertExpectations(t)
			mockScheduler.AssertExpectations(t)
			// Note: coordinator is not started by the current implementation, so no assertions needed
		})
	}
}

func TestTestableCannon_Shutdown(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler, []output.Sink)
		validate   func(*testing.T, error)
	}{
		{
			name: "successful_shutdown",
			setupMocks: func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler, []output.Sink) {
				mockBeacon := &mocks.MockBeaconNode{}
				mockCoordinator := &mocks.MockCoordinator{}
				mockScheduler := &mocks.MockScheduler{}
				
				mockSink := mocks.NewMockSink("test", "stdout")
				mockSink.On("Stop", mock.Anything).Return(nil)
				mockScheduler.On("Shutdown").Return(nil)

				return mockBeacon, mockCoordinator, mockScheduler, []output.Sink{mockSink}
			},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "shutdown_with_sink_error",
			setupMocks: func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler, []output.Sink) {
				mockBeacon := &mocks.MockBeaconNode{}
				mockCoordinator := &mocks.MockCoordinator{}
				mockScheduler := &mocks.MockScheduler{}
				
				mockSink := mocks.NewMockSink("test", "stdout")
				mockSink.On("Stop", mock.Anything).Return(errors.New("sink error"))

				return mockBeacon, mockCoordinator, mockScheduler, []output.Sink{mockSink}
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "sink error")
			},
		},
		{
			name: "shutdown_with_scheduler_error",
			setupMocks: func() (*mocks.MockBeaconNode, *mocks.MockCoordinator, *mocks.MockScheduler, []output.Sink) {
				mockBeacon := &mocks.MockBeaconNode{}
				mockCoordinator := &mocks.MockCoordinator{}
				mockScheduler := &mocks.MockScheduler{}
				
				mockSink := mocks.NewMockSink("test", "stdout")
				mockSink.On("Stop", mock.Anything).Return(nil)
				mockScheduler.On("Shutdown").Return(errors.New("scheduler error"))

				return mockBeacon, mockCoordinator, mockScheduler, []output.Sink{mockSink}
			},
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "scheduler error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBeacon, mockCoordinator, mockScheduler, sinks := tt.setupMocks()

			config := &Config{
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

			factory := NewTestCannonFactory().
				WithConfig(config).
				WithLogger(mocks.TestLogger()).
				WithBeaconNode(mockBeacon).
				WithCoordinator(mockCoordinator).
				WithScheduler(mockScheduler).
				WithTimeProvider(&mocks.MockTimeProvider{}).
				WithSinks(sinks)

			cannon, err := factory.Build()
			require.NoError(t, err)

			err = cannon.Shutdown(context.Background())

			tt.validate(t, err)

			// Verify mock expectations
			for _, sink := range sinks {
				if mockSink, ok := sink.(*mocks.MockSink); ok {
					mockSink.AssertExpectations(t)
				}
			}
			mockScheduler.AssertExpectations(t)
		})
	}
}

func TestTestableCannon_GettersSetters(t *testing.T) {
	mockBeacon := &mocks.MockBeaconNode{}
	mockCoordinator := &mocks.MockCoordinator{}
	mockScheduler := &mocks.MockScheduler{}
	mockTimeProvider := &mocks.MockTimeProvider{}
	
	mockSink := mocks.NewMockSink("test", "stdout")

	config := &Config{
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

	cannon, err := NewTestCannonFactory().
		WithConfig(config).
		WithLogger(mocks.TestLogger()).
		WithBeaconNode(mockBeacon).
		WithCoordinator(mockCoordinator).
		WithScheduler(mockScheduler).
		WithTimeProvider(mockTimeProvider).
		WithSinks([]output.Sink{mockSink}).
		Build()

	require.NoError(t, err)

	// Test getters
	assert.Equal(t, mockBeacon, cannon.GetBeacon())
	assert.Equal(t, mockCoordinator, cannon.GetCoordinator())
	assert.Equal(t, mockScheduler, cannon.GetScheduler())
	assert.Equal(t, mockTimeProvider, cannon.GetTimeProvider())
	assert.NotNil(t, cannon.GetNTPClient())
	assert.Equal(t, []output.Sink{mockSink}, cannon.GetSinks())
	assert.Empty(t, cannon.GetEventDerivers())

	// Test setter for event derivers
	mockDeriver := &MockDeriver{}
	cannon.SetEventDerivers([]Deriver{mockDeriver})
	assert.Equal(t, []Deriver{mockDeriver}, cannon.GetEventDerivers())
}

// MockDeriver for testing
type MockDeriver struct {
	mock.Mock
}

func (m *MockDeriver) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockDeriver) ActivationFork() spec.DataVersion {
	args := m.Called()
	return args.Get(0).(spec.DataVersion)
}

func (m *MockDeriver) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDeriver) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDeriver) OnEventsDerived(ctx context.Context, callback func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	m.Called(ctx, callback)
}