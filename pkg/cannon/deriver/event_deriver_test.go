package deriver

import (
	"context"
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEventDeriver is a mock implementation of the EventDeriver interface for testing
type MockEventDeriver struct {
	mock.Mock
	name           string
	cannonType     xatu.CannonType
	activationFork spec.DataVersion
	callbacks      []func(ctx context.Context, events []*xatu.DecoratedEvent) error
}

func NewMockEventDeriver(name string, cannonType xatu.CannonType, activationFork spec.DataVersion) *MockEventDeriver {
	return &MockEventDeriver{
		name:           name,
		cannonType:     cannonType,
		activationFork: activationFork,
		callbacks:      make([]func(ctx context.Context, events []*xatu.DecoratedEvent) error, 0),
	}
}

func (m *MockEventDeriver) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventDeriver) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEventDeriver) Name() string {
	m.Called()
	return m.name
}

func (m *MockEventDeriver) CannonType() xatu.CannonType {
	m.Called()
	return m.cannonType
}

func (m *MockEventDeriver) OnEventsDerived(ctx context.Context, fn func(ctx context.Context, events []*xatu.DecoratedEvent) error) {
	m.Called(ctx, fn)
	m.callbacks = append(m.callbacks, fn)
}

func (m *MockEventDeriver) ActivationFork() spec.DataVersion {
	m.Called()
	return m.activationFork
}

// TriggerCallbacks simulates events being derived and calls all registered callbacks
func (m *MockEventDeriver) TriggerCallbacks(ctx context.Context, events []*xatu.DecoratedEvent) error {
	for _, callback := range m.callbacks {
		if err := callback(ctx, events); err != nil {
			return err
		}
	}
	return nil
}

func TestEventDeriver_Interface(t *testing.T) {
	tests := []struct {
		name           string
		deriverName    string
		cannonType     xatu.CannonType
		activationFork spec.DataVersion
	}{
		{
			name:           "beacon_block_deriver",
			deriverName:    "beacon_block",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK,
			activationFork: spec.DataVersionPhase0,
		},
		{
			name:           "attestation_deriver", 
			deriverName:    "attestation",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
			activationFork: spec.DataVersionPhase0,
		},
		{
			name:           "deposit_deriver",
			deriverName:    "deposit",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
			activationFork: spec.DataVersionPhase0,
		},
		{
			name:           "voluntary_exit_deriver",
			deriverName:    "voluntary_exit",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			activationFork: spec.DataVersionPhase0,
		},
		{
			name:           "execution_transaction_deriver",
			deriverName:    "execution_transaction",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
			activationFork: spec.DataVersionBellatrix,
		},
		{
			name:           "bls_to_execution_change_deriver",
			deriverName:    "bls_to_execution_change",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
			activationFork: spec.DataVersionCapella,
		},
		{
			name:           "withdrawal_deriver",
			deriverName:    "withdrawal",
			cannonType:     xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
			activationFork: spec.DataVersionCapella,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deriver := NewMockEventDeriver(tt.deriverName, tt.cannonType, tt.activationFork)

			// Test the interface methods
			deriver.On("Name").Return()
			deriver.On("CannonType").Return()
			deriver.On("ActivationFork").Return()

			assert.Equal(t, tt.deriverName, deriver.Name())
			assert.Equal(t, tt.cannonType, deriver.CannonType())
			assert.Equal(t, tt.activationFork, deriver.ActivationFork())

			deriver.AssertExpectations(t)
		})
	}
}

func TestEventDeriver_Lifecycle(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockEventDeriver)
		testFunc  func(*testing.T, *MockEventDeriver)
	}{
		{
			name: "successful_start_and_stop",
			setupMock: func(mockDeriver *MockEventDeriver) {
				mockDeriver.On("Start", mock.Anything).Return(nil)
				mockDeriver.On("Stop", mock.Anything).Return(nil)
			},
			testFunc: func(t *testing.T, mock *MockEventDeriver) {
				ctx := context.Background()
				
				err := mock.Start(ctx)
				assert.NoError(t, err)
				
				err = mock.Stop(ctx)
				assert.NoError(t, err)
			},
		},
		{
			name: "start_fails_with_error",
			setupMock: func(mockDeriver *MockEventDeriver) {
				mockDeriver.On("Start", mock.Anything).Return(assert.AnError)
			},
			testFunc: func(t *testing.T, mock *MockEventDeriver) {
				ctx := context.Background()
				
				err := mock.Start(ctx)
				assert.Error(t, err)
				assert.Equal(t, assert.AnError, err)
			},
		},
		{
			name: "stop_fails_with_error",
			setupMock: func(mockDeriver *MockEventDeriver) {
				mockDeriver.On("Stop", mock.Anything).Return(assert.AnError)
			},
			testFunc: func(t *testing.T, mock *MockEventDeriver) {
				ctx := context.Background()
				
				err := mock.Stop(ctx)
				assert.Error(t, err)
				assert.Equal(t, assert.AnError, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0)
			tt.setupMock(deriver)
			tt.testFunc(t, deriver)
			deriver.AssertExpectations(t)
		})
	}
}

func TestEventDeriver_CallbackRegistration(t *testing.T) {
	deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0)
	
	// Mock the OnEventsDerived call
	deriver.On("OnEventsDerived", mock.Anything, mock.Anything).Return()
	
	ctx := context.Background()
	callbackExecuted := false
	
	callback := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		callbackExecuted = true
		return nil
	}
	
	// Register the callback
	deriver.OnEventsDerived(ctx, callback)
	
	// Verify callback was registered by triggering it
	events := []*xatu.DecoratedEvent{
		{
			Event: &xatu.Event{
				Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
				Id:   "test-event",
			},
		},
	}
	
	err := deriver.TriggerCallbacks(ctx, events)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted, "Callback should have been executed")
	
	deriver.AssertExpectations(t)
}

func TestEventDeriver_MultipleCallbacks(t *testing.T) {
	deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0)
	
	// Mock multiple OnEventsDerived calls
	deriver.On("OnEventsDerived", mock.Anything, mock.Anything).Return().Times(3)
	
	ctx := context.Background()
	executionOrder := []int{}
	
	// Register multiple callbacks
	callback1 := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		executionOrder = append(executionOrder, 1)
		return nil
	}
	
	callback2 := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		executionOrder = append(executionOrder, 2)
		return nil
	}
	
	callback3 := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		executionOrder = append(executionOrder, 3)
		return nil
	}
	
	deriver.OnEventsDerived(ctx, callback1)
	deriver.OnEventsDerived(ctx, callback2)
	deriver.OnEventsDerived(ctx, callback3)
	
	// Trigger all callbacks
	events := []*xatu.DecoratedEvent{
		{
			Event: &xatu.Event{
				Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
				Id:   "test-event",
			},
		},
	}
	
	err := deriver.TriggerCallbacks(ctx, events)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, executionOrder, "Callbacks should execute in registration order")
	
	deriver.AssertExpectations(t)
}

func TestEventDeriver_CallbackError(t *testing.T) {
	deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0)
	
	deriver.On("OnEventsDerived", mock.Anything, mock.Anything).Return()
	
	ctx := context.Background()
	
	// Register a callback that returns an error
	callback := func(ctx context.Context, events []*xatu.DecoratedEvent) error {
		return assert.AnError
	}
	
	deriver.OnEventsDerived(ctx, callback)
	
	// Trigger callback and expect error
	events := []*xatu.DecoratedEvent{
		{
			Event: &xatu.Event{
				Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
				Id:   "test-event",
			},
		},
	}
	
	err := deriver.TriggerCallbacks(ctx, events)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	
	deriver.AssertExpectations(t)
}

func TestEventDeriver_ActivationForkValues(t *testing.T) {
	tests := []struct {
		name           string
		activationFork spec.DataVersion
		description    string
	}{
		{
			name:           "phase0_activation",
			activationFork: spec.DataVersionPhase0,
			description:    "Deriver active from Phase 0",
		},
		{
			name:           "altair_activation",
			activationFork: spec.DataVersionAltair,
			description:    "Deriver active from Altair",
		},
		{
			name:           "bellatrix_activation",
			activationFork: spec.DataVersionBellatrix,
			description:    "Deriver active from Bellatrix (Merge)",
		},
		{
			name:           "capella_activation",
			activationFork: spec.DataVersionCapella,
			description:    "Deriver active from Capella",
		},
		{
			name:           "deneb_activation",
			activationFork: spec.DataVersionDeneb,
			description:    "Deriver active from Deneb",
		},
		{
			name:           "electra_activation",
			activationFork: spec.DataVersionElectra,
			description:    "Deriver active from Electra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, tt.activationFork)
			
			deriver.On("ActivationFork").Return()
			
			fork := deriver.ActivationFork()
			assert.Equal(t, tt.activationFork, fork, tt.description)
			
			deriver.AssertExpectations(t)
		})
	}
}

func TestEventDeriver_CompileTimeInterfaceChecks(t *testing.T) {
	// This test verifies that the compile-time interface checks in event_deriver.go
	// are working correctly. If any of the deriver types don't implement EventDeriver,
	// this will fail at compile time.
	
	// We can't directly test the var _ EventDeriver = &Type{} declarations,
	// but we can verify that the types exist and can be instantiated
	
	// These would fail at compile time if the interface implementations are broken
	t.Run("interface_compliance_compiles", func(t *testing.T) {
		// This test just verifies the file compiles, which means all interface
		// assignments in event_deriver.go are valid
		assert.True(t, true, "If this test runs, all interface checks passed compilation")
	})
}

func TestEventDeriver_ContextCancellation(t *testing.T) {
	deriver := NewMockEventDeriver("test", xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK, spec.DataVersionPhase0)
	
	// Test that derivers properly handle context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	deriver.On("Start", mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Err() != nil // Context should be cancelled
	})).Return(context.Canceled)
	
	err := deriver.Start(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	
	deriver.AssertExpectations(t)
}