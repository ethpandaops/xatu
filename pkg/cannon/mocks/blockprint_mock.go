package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockBlockprint is a mock implementation of Blockprint
type MockBlockprint struct {
	mock.Mock
}

// BlockClassification represents a block classification from blockprint
type BlockClassification struct {
	Slot        uint64 `json:"slot"`
	Blockprint  string `json:"blockprint"`
	BlockHash   string `json:"blockHash"`
	BlockNumber uint64 `json:"blockNumber"`
}

func (m *MockBlockprint) GetBlockClassifications(ctx context.Context, slots []uint64) (map[uint64]*BlockClassification, error) {
	args := m.Called(ctx, slots)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uint64]*BlockClassification), args.Error(1)
}

func (m *MockBlockprint) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper methods for common mock setups
func (m *MockBlockprint) SetupGetBlockClassificationsResponse(slots []uint64, classifications map[uint64]*BlockClassification) {
	m.On("GetBlockClassifications", mock.Anything, slots).Return(classifications, nil)
}

func (m *MockBlockprint) SetupGetBlockClassificationsError(slots []uint64, err error) {
	m.On("GetBlockClassifications", mock.Anything, slots).Return(nil, err)
}

func (m *MockBlockprint) SetupHealthCheckSuccess() {
	m.On("HealthCheck", mock.Anything).Return(nil)
}

func (m *MockBlockprint) SetupHealthCheckError(err error) {
	m.On("HealthCheck", mock.Anything).Return(err)
}

// SetupAnyGetBlockClassificationsResponse sets up a response for any GetBlockClassifications call
func (m *MockBlockprint) SetupAnyGetBlockClassificationsResponse(classifications map[uint64]*BlockClassification) {
	m.On("GetBlockClassifications", mock.Anything, mock.Anything).Return(classifications, nil)
}
