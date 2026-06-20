package txmgr

import (
	"context"
	"errors"
	"sync"
)

// ErrNoTransactionHooks means the context has no active txmgr transaction hook registry.
var ErrNoTransactionHooks = errors.New("txmgr: transaction hooks are not available")

// transactionHooksKey stores transaction hooks in context for the outer txmgr-managed transaction.
type transactionHooksKey struct{}

// transactionHooks stores hooks that must run after a successful transaction finish.
type transactionHooks struct {
	mu            sync.Mutex
	afterCommit   []func(context.Context)
	afterRollback []func(context.Context)
}

// AfterCommit registers fn for execution after a successful transaction commit.
func AfterCommit(ctx context.Context, fn func(context.Context)) error {
	if fn == nil {
		return nil
	}

	hooks, ok := hooksFromContext(ctx)
	if !ok {
		return ErrNoTransactionHooks
	}

	hooks.addAfterCommit(fn)
	return nil
}

// AfterRollback registers fn for execution after a successful transaction rollback.
func AfterRollback(ctx context.Context, fn func(context.Context)) error {
	if fn == nil {
		return nil
	}

	hooks, ok := hooksFromContext(ctx)
	if !ok {
		return ErrNoTransactionHooks
	}

	hooks.addAfterRollback(fn)
	return nil
}

// newTransactionHooks creates an empty transaction hook registry.
func newTransactionHooks() *transactionHooks {
	return &transactionHooks{
		mu:            sync.Mutex{},
		afterCommit:   make([]func(context.Context), 0),
		afterRollback: make([]func(context.Context), 0),
	}
}

// hooksFromContext extracts transaction hooks from context.
func hooksFromContext(ctx context.Context) (*transactionHooks, bool) {
	hooks, ok := ctx.Value(transactionHooksKey{}).(*transactionHooks)
	return hooks, ok && hooks != nil
}

// withHooks returns context with a transaction hook registry.
func withHooks(ctx context.Context, hooks *transactionHooks) context.Context {
	return context.WithValue(ctx, transactionHooksKey{}, hooks)
}

// withoutHooks returns context without an active transaction hook registry.
func withoutHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, transactionHooksKey{}, (*transactionHooks)(nil))
}

// addAfterCommit registers a hook for a successful transaction commit.
func (h *transactionHooks) addAfterCommit(fn func(context.Context)) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.afterCommit = append(h.afterCommit, fn)
}

// addAfterRollback registers a hook for a successful transaction rollback.
func (h *transactionHooks) addAfterRollback(fn func(context.Context)) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.afterRollback = append(h.afterRollback, fn)
}

// runAfterCommit runs commit hooks in registration order.
func (h *transactionHooks) runAfterCommit(ctx context.Context) {
	for _, hook := range h.commitHooks() {
		hook(ctx)
	}
}

// runAfterRollback runs rollback hooks in registration order.
func (h *transactionHooks) runAfterRollback(ctx context.Context) {
	for _, hook := range h.rollbackHooks() {
		hook(ctx)
	}
}

// commitHooks returns a snapshot of registered commit hooks.
func (h *transactionHooks) commitHooks() []func(context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]func(context.Context){}, h.afterCommit...)
}

// rollbackHooks returns a snapshot of registered rollback hooks.
func (h *transactionHooks) rollbackHooks() []func(context.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]func(context.Context){}, h.afterRollback...)
}
