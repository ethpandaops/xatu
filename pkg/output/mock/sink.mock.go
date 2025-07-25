// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ethpandaops/xatu/pkg/output (interfaces: Sink)
//
// Generated by this command:
//
//	mockgen -package mock -destination mock/sink.mock.go github.com/ethpandaops/xatu/pkg/output Sink
//

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	xatu "github.com/ethpandaops/xatu/pkg/proto/xatu"
	gomock "go.uber.org/mock/gomock"
)

// MockSink is a mock of Sink interface.
type MockSink struct {
	ctrl     *gomock.Controller
	recorder *MockSinkMockRecorder
	isgomock struct{}
}

// MockSinkMockRecorder is the mock recorder for MockSink.
type MockSinkMockRecorder struct {
	mock *MockSink
}

// NewMockSink creates a new mock instance.
func NewMockSink(ctrl *gomock.Controller) *MockSink {
	mock := &MockSink{ctrl: ctrl}
	mock.recorder = &MockSinkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSink) EXPECT() *MockSinkMockRecorder {
	return m.recorder
}

// HandleNewDecoratedEvent mocks base method.
func (m *MockSink) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleNewDecoratedEvent", ctx, event)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleNewDecoratedEvent indicates an expected call of HandleNewDecoratedEvent.
func (mr *MockSinkMockRecorder) HandleNewDecoratedEvent(ctx, event any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleNewDecoratedEvent", reflect.TypeOf((*MockSink)(nil).HandleNewDecoratedEvent), ctx, event)
}

// HandleNewDecoratedEvents mocks base method.
func (m *MockSink) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HandleNewDecoratedEvents", ctx, events)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleNewDecoratedEvents indicates an expected call of HandleNewDecoratedEvents.
func (mr *MockSinkMockRecorder) HandleNewDecoratedEvents(ctx, events any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleNewDecoratedEvents", reflect.TypeOf((*MockSink)(nil).HandleNewDecoratedEvents), ctx, events)
}

// Name mocks base method.
func (m *MockSink) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockSinkMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockSink)(nil).Name))
}

// Start mocks base method.
func (m *MockSink) Start(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockSinkMockRecorder) Start(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockSink)(nil).Start), ctx)
}

// Stop mocks base method.
func (m *MockSink) Stop(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockSinkMockRecorder) Stop(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockSink)(nil).Stop), ctx)
}

// Type mocks base method.
func (m *MockSink) Type() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Type")
	ret0, _ := ret[0].(string)
	return ret0
}

// Type indicates an expected call of Type.
func (mr *MockSinkMockRecorder) Type() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Type", reflect.TypeOf((*MockSink)(nil).Type))
}
