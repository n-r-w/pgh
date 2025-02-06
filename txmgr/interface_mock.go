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

// MockITransactionFinisher is a mock of ITransactionFinisher interface.
type MockITransactionFinisher struct {
	ctrl     *gomock.Controller
	recorder *MockITransactionFinisherMockRecorder
}

// MockITransactionFinisherMockRecorder is the mock recorder for MockITransactionFinisher.
type MockITransactionFinisherMockRecorder struct {
	mock *MockITransactionFinisher
}

// NewMockITransactionFinisher creates a new mock instance.
func NewMockITransactionFinisher(ctrl *gomock.Controller) *MockITransactionFinisher {
	mock := &MockITransactionFinisher{ctrl: ctrl}
	mock.recorder = &MockITransactionFinisherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockITransactionFinisher) EXPECT() *MockITransactionFinisherMockRecorder {
	return m.recorder
}

// Commit mocks base method.
func (m *MockITransactionFinisher) Commit(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Commit", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Commit indicates an expected call of Commit.
func (mr *MockITransactionFinisherMockRecorder) Commit(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Commit", reflect.TypeOf((*MockITransactionFinisher)(nil).Commit), ctx)
}

// Rollback mocks base method.
func (m *MockITransactionFinisher) Rollback(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Rollback", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Rollback indicates an expected call of Rollback.
func (mr *MockITransactionFinisherMockRecorder) Rollback(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Rollback", reflect.TypeOf((*MockITransactionFinisher)(nil).Rollback), ctx)
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

// BeginTx mocks base method.
func (m *MockITransactionBeginner) BeginTx(ctx context.Context, opts Options) (context.Context, ITransactionFinisher, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeginTx", ctx, opts)
	ret0, _ := ret[0].(context.Context)
	ret1, _ := ret[1].(ITransactionFinisher)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// BeginTx indicates an expected call of BeginTx.
func (mr *MockITransactionBeginnerMockRecorder) BeginTx(ctx, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeginTx", reflect.TypeOf((*MockITransactionBeginner)(nil).BeginTx), ctx, opts)
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

// BeginTx mocks base method.
func (m *MockITransactionManager) BeginTx(ctx context.Context, opts ...Option) (context.Context, ITransactionFinisher, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "BeginTx", varargs...)
	ret0, _ := ret[0].(context.Context)
	ret1, _ := ret[1].(ITransactionFinisher)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// BeginTx indicates an expected call of BeginTx.
func (mr *MockITransactionManagerMockRecorder) BeginTx(ctx any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeginTx", reflect.TypeOf((*MockITransactionManager)(nil).BeginTx), varargs...)
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
