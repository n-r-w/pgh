// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go
//
// Generated by this command:
//
//	mockgen -source interface.go -destination interface_mock.go -package txmgr
//

// Package txmgr is a generated GoMock package.
package txmgr

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockITransactionInformer is a mock of ITransactionInformer interface.
type MockITransactionInformer struct {
	ctrl     *gomock.Controller
	recorder *MockITransactionInformerMockRecorder
}

// MockITransactionInformerMockRecorder is the mock recorder for MockITransactionInformer.
type MockITransactionInformerMockRecorder struct {
	mock *MockITransactionInformer
}

// NewMockITransactionInformer creates a new mock instance.
func NewMockITransactionInformer(ctrl *gomock.Controller) *MockITransactionInformer {
	mock := &MockITransactionInformer{ctrl: ctrl}
	mock.recorder = &MockITransactionInformerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockITransactionInformer) EXPECT() *MockITransactionInformerMockRecorder {
	return m.recorder
}

// InTransaction mocks base method.
func (m *MockITransactionInformer) InTransaction(ctx context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InTransaction", ctx)
	ret0, _ := ret[0].(bool)
	return ret0
}

// InTransaction indicates an expected call of InTransaction.
func (mr *MockITransactionInformerMockRecorder) InTransaction(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InTransaction", reflect.TypeOf((*MockITransactionInformer)(nil).InTransaction), ctx)
}

// TransactionOptions mocks base method.
func (m *MockITransactionInformer) TransactionOptions(ctx context.Context) Options {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TransactionOptions", ctx)
	ret0, _ := ret[0].(Options)
	return ret0
}

// TransactionOptions indicates an expected call of TransactionOptions.
func (mr *MockITransactionInformerMockRecorder) TransactionOptions(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TransactionOptions", reflect.TypeOf((*MockITransactionInformer)(nil).TransactionOptions), ctx)
}

// MockITransactionBeginner is a mock of ITransactionBeginner interface.
type MockITransactionBeginner struct {
	ctrl     *gomock.Controller
	recorder *MockITransactionBeginnerMockRecorder
}

// MockITransactionBeginnerMockRecorder is the mock recorder for MockITransactionBeginner.
type MockITransactionBeginnerMockRecorder struct {
	mock *MockITransactionBeginner
}

// NewMockITransactionBeginner creates a new mock instance.
func NewMockITransactionBeginner(ctrl *gomock.Controller) *MockITransactionBeginner {
	mock := &MockITransactionBeginner{ctrl: ctrl}
	mock.recorder = &MockITransactionBeginnerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockITransactionBeginner) EXPECT() *MockITransactionBeginnerMockRecorder {
	return m.recorder
}

// Begin mocks base method.
func (m *MockITransactionBeginner) Begin(ctx context.Context, f func(context.Context) error, opts Options) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Begin", ctx, f, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// Begin indicates an expected call of Begin.
func (mr *MockITransactionBeginnerMockRecorder) Begin(ctx, f, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Begin", reflect.TypeOf((*MockITransactionBeginner)(nil).Begin), ctx, f, opts)
}

// WithoutTransaction mocks base method.
func (m *MockITransactionBeginner) WithoutTransaction(ctx context.Context) context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithoutTransaction", ctx)
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// WithoutTransaction indicates an expected call of WithoutTransaction.
func (mr *MockITransactionBeginnerMockRecorder) WithoutTransaction(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithoutTransaction", reflect.TypeOf((*MockITransactionBeginner)(nil).WithoutTransaction), ctx)
}

// MockITransactionManager is a mock of ITransactionManager interface.
type MockITransactionManager struct {
	ctrl     *gomock.Controller
	recorder *MockITransactionManagerMockRecorder
}

// MockITransactionManagerMockRecorder is the mock recorder for MockITransactionManager.
type MockITransactionManagerMockRecorder struct {
	mock *MockITransactionManager
}

// NewMockITransactionManager creates a new mock instance.
func NewMockITransactionManager(ctrl *gomock.Controller) *MockITransactionManager {
	mock := &MockITransactionManager{ctrl: ctrl}
	mock.recorder = &MockITransactionManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockITransactionManager) EXPECT() *MockITransactionManagerMockRecorder {
	return m.recorder
}

// Begin mocks base method.
func (m *MockITransactionManager) Begin(ctx context.Context, f func(context.Context) error, opts ...Option) error {
	m.ctrl.T.Helper()
	varargs := []any{ctx, f}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Begin", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Begin indicates an expected call of Begin.
func (mr *MockITransactionManagerMockRecorder) Begin(ctx, f any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, f}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Begin", reflect.TypeOf((*MockITransactionManager)(nil).Begin), varargs...)
}

// WithoutTransaction mocks base method.
func (m *MockITransactionManager) WithoutTransaction(ctx context.Context) context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithoutTransaction", ctx)
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// WithoutTransaction indicates an expected call of WithoutTransaction.
func (mr *MockITransactionManagerMockRecorder) WithoutTransaction(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithoutTransaction", reflect.TypeOf((*MockITransactionManager)(nil).WithoutTransaction), ctx)
}
