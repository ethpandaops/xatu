package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the Service interface for testing
type MockService struct {
	mock.Mock
	name      Name
	callbacks []func(ctx context.Context) error
}

func NewMockService(name Name) *MockService {
	return &MockService{
		name:      name,
		callbacks: make([]func(ctx context.Context) error, 0),
	}
}

func (m *MockService) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockService) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockService) Ready(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockService) OnReady(ctx context.Context, cb func(ctx context.Context) error) {
	m.Called(ctx, cb)
	m.callbacks = append(m.callbacks, cb)
}

func (m *MockService) Name() Name {
	m.Called()
	return m.name
}

// TriggerReadyCallbacks simulates the service becoming ready and triggers all callbacks
func (m *MockService) TriggerReadyCallbacks(ctx context.Context) error {
	for _, callback := range m.callbacks {
		if err := callback(ctx); err != nil {
			return err
		}
	}
	return nil
}

func TestName_Type(t *testing.T) {
	tests := []struct {
		name     string
		value    Name
		expected string
	}{
		{
			name:     "empty_name",
			value:    Name(""),
			expected: "",
		},
		{
			name:     "beacon_service",
			value:    Name("beacon"),
			expected: "beacon",
		},
		{
			name:     "metadata_service",
			value:    Name("metadata"),
			expected: "metadata",
		},
		{
			name:     "duties_service",
			value:    Name("duties"),
			expected: "duties",
		},
		{
			name:     "long_service_name",
			value:    Name("very-long-service-name-with-dashes"),
			expected: "very-long-service-name-with-dashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.value))
			assert.IsType(t, Name(""), tt.value)
		})
	}
}

func TestService_Interface(t *testing.T) {
	tests := []struct {
		name        string
		serviceName Name
	}{
		{
			name:        "beacon_service",
			serviceName: Name("beacon"),
		},
		{
			name:        "metadata_service",
			serviceName: Name("metadata"),
		},
		{
			name:        "duties_service",
			serviceName: Name("duties"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMockService(tt.serviceName)

			// Test the Name method
			service.On("Name").Return()
			assert.Equal(t, tt.serviceName, service.Name())

			service.AssertExpectations(t)
		})
	}
}

func TestService_Lifecycle(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockService)
		testFunc  func(*testing.T, *MockService)
	}{
		{
			name: "successful_start_and_stop",
			setupMock: func(service *MockService) {
				service.On("Start", mock.Anything).Return(nil)
				service.On("Stop", mock.Anything).Return(nil)
			},
			testFunc: func(t *testing.T, service *MockService) {
				ctx := context.Background()

				err := service.Start(ctx)
				assert.NoError(t, err)

				err = service.Stop(ctx)
				assert.NoError(t, err)
			},
		},
		{
			name: "start_fails_with_error",
			setupMock: func(service *MockService) {
				service.On("Start", mock.Anything).Return(assert.AnError)
			},
			testFunc: func(t *testing.T, service *MockService) {
				ctx := context.Background()

				err := service.Start(ctx)
				assert.Error(t, err)
				assert.Equal(t, assert.AnError, err)
			},
		},
		{
			name: "stop_fails_with_error",
			setupMock: func(service *MockService) {
				service.On("Stop", mock.Anything).Return(assert.AnError)
			},
			testFunc: func(t *testing.T, service *MockService) {
				ctx := context.Background()

				err := service.Stop(ctx)
				assert.Error(t, err)
				assert.Equal(t, assert.AnError, err)
			},
		},
		{
			name: "ready_check_passes",
			setupMock: func(service *MockService) {
				service.On("Ready", mock.Anything).Return(nil)
			},
			testFunc: func(t *testing.T, service *MockService) {
				ctx := context.Background()

				err := service.Ready(ctx)
				assert.NoError(t, err)
			},
		},
		{
			name: "ready_check_fails",
			setupMock: func(service *MockService) {
				service.On("Ready", mock.Anything).Return(assert.AnError)
			},
			testFunc: func(t *testing.T, service *MockService) {
				ctx := context.Background()

				err := service.Ready(ctx)
				assert.Error(t, err)
				assert.Equal(t, assert.AnError, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMockService(Name("test"))
			tt.setupMock(service)
			tt.testFunc(t, service)
			service.AssertExpectations(t)
		})
	}
}

func TestService_OnReadyCallbacks(t *testing.T) {
	service := NewMockService(Name("test"))

	// Mock the OnReady call
	service.On("OnReady", mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	callbackExecuted := false

	callback := func(ctx context.Context) error {
		callbackExecuted = true
		return nil
	}

	// Register the callback
	service.OnReady(ctx, callback)

	// Verify callback was registered by triggering it
	err := service.TriggerReadyCallbacks(ctx)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted, "OnReady callback should have been executed")

	service.AssertExpectations(t)
}

func TestService_MultipleOnReadyCallbacks(t *testing.T) {
	service := NewMockService(Name("test"))

	// Mock multiple OnReady calls
	service.On("OnReady", mock.Anything, mock.Anything).Return().Times(3)

	ctx := context.Background()
	executionOrder := []int{}

	// Register multiple callbacks
	callback1 := func(ctx context.Context) error {
		executionOrder = append(executionOrder, 1)
		return nil
	}

	callback2 := func(ctx context.Context) error {
		executionOrder = append(executionOrder, 2)
		return nil
	}

	callback3 := func(ctx context.Context) error {
		executionOrder = append(executionOrder, 3)
		return nil
	}

	service.OnReady(ctx, callback1)
	service.OnReady(ctx, callback2)
	service.OnReady(ctx, callback3)

	// Trigger all callbacks
	err := service.TriggerReadyCallbacks(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, executionOrder, "OnReady callbacks should execute in registration order")

	service.AssertExpectations(t)
}

func TestService_OnReadyCallbackError(t *testing.T) {
	service := NewMockService(Name("test"))

	service.On("OnReady", mock.Anything, mock.Anything).Return()

	ctx := context.Background()

	// Register a callback that returns an error
	callback := func(ctx context.Context) error {
		return assert.AnError
	}

	service.OnReady(ctx, callback)

	// Trigger callback and expect error
	err := service.TriggerReadyCallbacks(ctx)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)

	service.AssertExpectations(t)
}

func TestService_ContextCancellation(t *testing.T) {
	service := NewMockService(Name("test"))

	// Test that services properly handle context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	service.On("Start", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Err() != nil // Context should be cancelled
	})).Return(context.Canceled)

	service.On("Ready", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Err() != nil // Context should be cancelled
	})).Return(context.Canceled)

	err := service.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	err = service.Ready(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	service.AssertExpectations(t)
}

func TestService_NameComparisons(t *testing.T) {
	// Test that service names can be compared
	name1 := Name("service1")
	name2 := Name("service2")
	name3 := Name("service1")

	assert.NotEqual(t, name1, name2)
	assert.Equal(t, name1, name3)
	assert.True(t, name1 == name3)
	assert.False(t, name1 == name2)
}

func TestService_ServiceInterfaceCompliance(t *testing.T) {
	// Test that MockService implements the Service interface
	var _ Service = &MockService{}

	// This test verifies interface compliance at compile time
	service := NewMockService(Name("test"))
	assert.NotNil(t, service)
	assert.Implements(t, (*Service)(nil), service)
}
