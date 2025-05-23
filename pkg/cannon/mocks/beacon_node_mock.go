package mocks

import (
	"context"

	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/cannon/ethereum/services"
	"github.com/stretchr/testify/mock"
)

// MockBeaconNode is a mock implementation of BeaconNode
type MockBeaconNode struct {
	mock.Mock
}

func (m *MockBeaconNode) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockBeaconNode) GetBeaconBlock(ctx context.Context, identifier string, ignoreMetrics ...bool) (*spec.VersionedSignedBeaconBlock, error) {
	args := m.Called(ctx, identifier, ignoreMetrics)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*spec.VersionedSignedBeaconBlock), args.Error(1)
}

func (m *MockBeaconNode) GetValidators(ctx context.Context, identifier string) (map[phase0.ValidatorIndex]*apiv1.Validator, error) {
	args := m.Called(ctx, identifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[phase0.ValidatorIndex]*apiv1.Validator), args.Error(1)
}

func (m *MockBeaconNode) Node() beacon.Node {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(beacon.Node)
}

func (m *MockBeaconNode) Metadata() *services.MetadataService {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*services.MetadataService)
}

func (m *MockBeaconNode) Duties() *services.DutiesService {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*services.DutiesService)
}

func (m *MockBeaconNode) OnReady(ctx context.Context, callback func(ctx context.Context) error) {
	m.Called(ctx, callback)
}

func (m *MockBeaconNode) Synced(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper methods for common mock setups
func (m *MockBeaconNode) SetupSyncedSuccess() {
	m.On("Synced", mock.Anything).Return(nil)
}

func (m *MockBeaconNode) SetupSyncedError(err error) {
	m.On("Synced", mock.Anything).Return(err)
}

func (m *MockBeaconNode) SetupBlockResponse(identifier string, block *spec.VersionedSignedBeaconBlock) {
	m.On("GetBeaconBlock", mock.Anything, identifier, mock.Anything).Return(block, nil)
}

func (m *MockBeaconNode) SetupBlockError(identifier string, err error) {
	m.On("GetBeaconBlock", mock.Anything, identifier, mock.Anything).Return(nil, err)
}

func (m *MockBeaconNode) SetupStartSuccess() {
	m.On("Start", mock.Anything).Return(nil)
}

func (m *MockBeaconNode) SetupStartError(err error) {
	m.On("Start", mock.Anything).Return(err)
}

func (m *MockBeaconNode) SetupStopSuccess() {
	m.On("Stop", mock.Anything).Return(nil)
}

func (m *MockBeaconNode) SetupOnReadyCallback() {
	m.On("OnReady", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return()
}

func (m *MockBeaconNode) TriggerOnReadyCallback(ctx context.Context) error {
	// This helper method allows tests to manually trigger the OnReady callback
	calls := m.Calls
	for _, call := range calls {
		if call.Method == "OnReady" && len(call.Arguments) >= 2 {
			if callback, ok := call.Arguments[1].(func(context.Context) error); ok {
				return callback(ctx)
			}
		}
	}
	return nil
}