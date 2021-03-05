// Code generated by MockGen. DO NOT EDIT.
// Source: podmon/test/ssh (interfaces: ClientWrapper)

// Package mocks is a generated GoMock package.
package mocks

import (
	os "os"
	ssh "podmon/test/ssh"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockClientWrapper is a mock of ClientWrapper interface.
type MockClientWrapper struct {
	ctrl     *gomock.Controller
	recorder *MockClientWrapperMockRecorder
}

// MockClientWrapperMockRecorder is the mock recorder for MockClientWrapper.
type MockClientWrapperMockRecorder struct {
	mock *MockClientWrapper
}

// NewMockClientWrapper creates a new mock instance.
func NewMockClientWrapper(ctrl *gomock.Controller) *MockClientWrapper {
	mock := &MockClientWrapper{ctrl: ctrl}
	mock.recorder = &MockClientWrapperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientWrapper) EXPECT() *MockClientWrapperMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockClientWrapper) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockClientWrapperMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockClientWrapper)(nil).Close))
}

// Copy mocks base method.
func (m *MockClientWrapper) Copy(arg0 os.File, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Copy", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Copy indicates an expected call of Copy.
func (mr *MockClientWrapperMockRecorder) Copy(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Copy", reflect.TypeOf((*MockClientWrapper)(nil).Copy), arg0, arg1, arg2)
}

// GetSession mocks base method.
func (m *MockClientWrapper) GetSession(arg0 string) (ssh.SessionWrapper, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSession", arg0)
	ret0, _ := ret[0].(ssh.SessionWrapper)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSession indicates an expected call of GetSession.
func (mr *MockClientWrapperMockRecorder) GetSession(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSession", reflect.TypeOf((*MockClientWrapper)(nil).GetSession), arg0)
}

// SendRequest mocks base method.
func (m *MockClientWrapper) SendRequest(arg0 string, arg1 bool, arg2 []byte) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendRequest", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendRequest indicates an expected call of SendRequest.
func (mr *MockClientWrapperMockRecorder) SendRequest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendRequest", reflect.TypeOf((*MockClientWrapper)(nil).SendRequest), arg0, arg1, arg2)
}