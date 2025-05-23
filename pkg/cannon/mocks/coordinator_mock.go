package mocks

import (
	"context"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/mock"
)

// MockCoordinator is a mock implementation of Coordinator
type MockCoordinator struct {
	mock.Mock
}

func (m *MockCoordinator) GetCannonLocation(ctx context.Context, typ xatu.CannonType, networkID string) (*xatu.CannonLocation, error) {
	args := m.Called(ctx, typ, networkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*xatu.CannonLocation), args.Error(1)
}

func (m *MockCoordinator) UpsertCannonLocationRequest(ctx context.Context, location *xatu.CannonLocation) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *MockCoordinator) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCoordinator) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper methods for common mock setups
func (m *MockCoordinator) SetupGetCannonLocationResponse(typ xatu.CannonType, networkID string, resp *xatu.CannonLocation) {
	m.On("GetCannonLocation", mock.Anything, typ, networkID).Return(resp, nil)
}

func (m *MockCoordinator) SetupGetCannonLocationError(typ xatu.CannonType, networkID string, err error) {
	m.On("GetCannonLocation", mock.Anything, typ, networkID).Return(nil, err)
}

func (m *MockCoordinator) SetupUpsertCannonLocationSuccess(location *xatu.CannonLocation) {
	m.On("UpsertCannonLocationRequest", mock.Anything, location).Return(nil)
}

func (m *MockCoordinator) SetupUpsertCannonLocationError(location *xatu.CannonLocation, err error) {
	m.On("UpsertCannonLocationRequest", mock.Anything, location).Return(err)
}

func (m *MockCoordinator) SetupStartSuccess() {
	m.On("Start", mock.Anything).Return(nil)
}

func (m *MockCoordinator) SetupStartError(err error) {
	m.On("Start", mock.Anything).Return(err)
}

func (m *MockCoordinator) SetupStopSuccess() {
	m.On("Stop", mock.Anything).Return(nil)
}

func (m *MockCoordinator) SetupStopError(err error) {
	m.On("Stop", mock.Anything).Return(err)
}

// SetupAnyGetCannonLocationResponse sets up a response for any GetCannonLocation call
func (m *MockCoordinator) SetupAnyGetCannonLocationResponse(resp *xatu.CannonLocation) {
	m.On("GetCannonLocation", mock.Anything, mock.Anything, mock.Anything).Return(resp, nil)
}

// SetupAnyUpsertCannonLocationSuccess sets up success for any UpsertCannonLocationRequest call
func (m *MockCoordinator) SetupAnyUpsertCannonLocationSuccess() {
	m.On("UpsertCannonLocationRequest", mock.Anything, mock.Anything).Return(nil)
}