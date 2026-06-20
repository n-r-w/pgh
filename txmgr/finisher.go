package txmgr

import (
	"context"
	"sync"
)

// transactionHookFinisher runs transaction hooks after successful transaction finalization.
type transactionHookFinisher struct {
	tm    *TransactionManager
	next  ITransactionFinisher
	hooks *transactionHooks

	mu       sync.Mutex
	finished bool
}

// Commit commits the transaction and runs commit hooks after a successful commit.
func (f *transactionHookFinisher) Commit(ctx context.Context) error {
	if err := f.next.Commit(ctx); err != nil {
		return err
	}

	if f.markFinished() {
		f.hooks.runAfterCommit(f.hookContext(ctx))
	}

	return nil
}

// Rollback rolls back the transaction and runs rollback hooks after a successful rollback.
func (f *transactionHookFinisher) Rollback(ctx context.Context) error {
	if err := f.next.Rollback(ctx); err != nil {
		return err
	}

	if f.markFinished() {
		f.hooks.runAfterRollback(f.hookContext(ctx))
	}

	return nil
}

// markFinished returns true only for the first successful finalization.
func (f *transactionHookFinisher) markFinished() bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.finished {
		return false
	}

	f.finished = true
	return true
}

// hookContext returns context without transaction state and without active hook registration.
func (f *transactionHookFinisher) hookContext(ctx context.Context) context.Context {
	return withoutHooks(f.tm.WithoutTransaction(ctx))
}
