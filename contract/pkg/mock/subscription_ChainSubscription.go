// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cerebellum-network/cere-ddc-sdk-go/contract/pkg/subscription (interfaces: ChainSubscription)

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	types "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	gomock "github.com/golang/mock/gomock"
)

// MockChainSubscription is a mock of ChainSubscription interface.
type MockChainSubscription struct {
	ctrl     *gomock.Controller
	recorder *MockChainSubscriptionMockRecorder
}

// MockChainSubscriptionMockRecorder is the mock recorder for MockChainSubscription.
type MockChainSubscriptionMockRecorder struct {
	mock *MockChainSubscription
}

// NewMockChainSubscription creates a new mock instance.
func NewMockChainSubscription(ctrl *gomock.Controller) *MockChainSubscription {
	mock := &MockChainSubscription{ctrl: ctrl}
	mock.recorder = &MockChainSubscriptionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChainSubscription) EXPECT() *MockChainSubscriptionMockRecorder {
	return m.recorder
}

// Chan mocks base method.
func (m *MockChainSubscription) Chan() <-chan types.StorageChangeSet {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Chan")
	ret0, _ := ret[0].(<-chan types.StorageChangeSet)
	return ret0
}

// Chan indicates an expected call of Chan.
func (mr *MockChainSubscriptionMockRecorder) Chan() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Chan", reflect.TypeOf((*MockChainSubscription)(nil).Chan))
}

// Err mocks base method.
func (m *MockChainSubscription) Err() <-chan error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Err")
	ret0, _ := ret[0].(<-chan error)
	return ret0
}

// Err indicates an expected call of Err.
func (mr *MockChainSubscriptionMockRecorder) Err() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Err", reflect.TypeOf((*MockChainSubscription)(nil).Err))
}

// Unsubscribe mocks base method.
func (m *MockChainSubscription) Unsubscribe() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Unsubscribe")
}

// Unsubscribe indicates an expected call of Unsubscribe.
func (mr *MockChainSubscriptionMockRecorder) Unsubscribe() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribe", reflect.TypeOf((*MockChainSubscription)(nil).Unsubscribe))
}
