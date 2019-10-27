// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener/pkg/utils/kubernetes/client (interfaces: Cleaner,GoneEnsurer)

// Package client is a generated GoMock package.
package client

import (
	context "context"
	client "github.com/gardener/gardener/pkg/utils/kubernetes/client"
	gomock "github.com/golang/mock/gomock"
	runtime "k8s.io/apimachinery/pkg/runtime"
	reflect "reflect"
	client0 "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockCleaner is a mock of Cleaner interface
type MockCleaner struct {
	ctrl     *gomock.Controller
	recorder *MockCleanerMockRecorder
}

// MockCleanerMockRecorder is the mock recorder for MockCleaner
type MockCleanerMockRecorder struct {
	mock *MockCleaner
}

// NewMockCleaner creates a new mock instance
func NewMockCleaner(ctrl *gomock.Controller) *MockCleaner {
	mock := &MockCleaner{ctrl: ctrl}
	mock.recorder = &MockCleanerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCleaner) EXPECT() *MockCleanerMockRecorder {
	return m.recorder
}

// Clean mocks base method
func (m *MockCleaner) Clean(arg0 context.Context, arg1 client0.Client, arg2 runtime.Object, arg3 ...client.CleanOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Clean", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Clean indicates an expected call of Clean
func (mr *MockCleanerMockRecorder) Clean(arg0, arg1, arg2 interface{}, arg3 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1, arg2}, arg3...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Clean", reflect.TypeOf((*MockCleaner)(nil).Clean), varargs...)
}

// MockGoneEnsurer is a mock of GoneEnsurer interface
type MockGoneEnsurer struct {
	ctrl     *gomock.Controller
	recorder *MockGoneEnsurerMockRecorder
}

// MockGoneEnsurerMockRecorder is the mock recorder for MockGoneEnsurer
type MockGoneEnsurerMockRecorder struct {
	mock *MockGoneEnsurer
}

// NewMockGoneEnsurer creates a new mock instance
func NewMockGoneEnsurer(ctrl *gomock.Controller) *MockGoneEnsurer {
	mock := &MockGoneEnsurer{ctrl: ctrl}
	mock.recorder = &MockGoneEnsurerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGoneEnsurer) EXPECT() *MockGoneEnsurerMockRecorder {
	return m.recorder
}

// EnsureGone mocks base method
func (m *MockGoneEnsurer) EnsureGone(arg0 context.Context, arg1 client0.Client, arg2 runtime.Object, arg3 ...client0.ListOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1, arg2}
	for _, a := range arg3 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "EnsureGone", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureGone indicates an expected call of EnsureGone
func (mr *MockGoneEnsurerMockRecorder) EnsureGone(arg0, arg1, arg2 interface{}, arg3 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1, arg2}, arg3...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureGone", reflect.TypeOf((*MockGoneEnsurer)(nil).EnsureGone), varargs...)
}
