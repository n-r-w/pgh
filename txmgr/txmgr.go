// Package txmgr implements a database-agnostic transaction manager.
package txmgr

import (
	"context"
	"fmt"
)

// Options represents transaction manager configuration options.
type Options struct {
	// Level defines the transaction isolation level.
	Level TransactionLevel
	// Mode defines the transaction operation mode.
	Mode TransactionMode
	// Lock indicates if object locking is required.
	// This is an advisory option and the implementation decides what to lock.
	// In most cases it means SELECT ... FOR UPDATE.
	Lock bool
}

// Option transaction manager option function.
type Option func(*Options)

// WithTransactionLevel sets the transaction isolation level.
func WithTransactionLevel(level TransactionLevel) Option {
	return func(opts *Options) {
		opts.Level = level
	}
}

// WithTransactionMode sets the transaction mode.
func WithTransactionMode(mode TransactionMode) Option {
	return func(opts *Options) {
		opts.Mode = mode
	}
}

// WithLock enables object locking.
func WithLock() Option {
	return func(opts *Options) {
		opts.Lock = true
	}
}

// TransactionManager handles database transactions.
type TransactionManager struct {
	tmBeginner      ITransactionBeginner
	tmImplementator ITransactionInformer
}

// New creates a new TransactionManager.
func New(tmBeginner ITransactionBeginner, tmImplementator ITransactionInformer) *TransactionManager {
	return &TransactionManager{
		tmBeginner:      tmBeginner,
		tmImplementator: tmImplementator,
	}
}

// Begin starts a new transaction and executes the function.
func (tm *TransactionManager) Begin(ctx context.Context, f func(ctxTr context.Context) error, opts ...Option) error {
	// get options
	tmOpts := &Options{
		Level: TxLevelDefault,
		Mode:  TxModeDefault,
	}
	for _, opt := range opts {
		opt(tmOpts)
	}

	if tm.tmImplementator.InTransaction(ctx) { // transaction is already started
		cOpt := tm.tmImplementator.TransactionOptions(ctx)

		// we cannot change transaction level and mode
		if cOpt.Level != tmOpts.Level {
			return fmt.Errorf("transaction level mismatch: %d != %d", cOpt.Level, tmOpts.Level)
		}
		if cOpt.Mode != tmOpts.Mode {
			return fmt.Errorf("transaction mode mismatch: %d != %d", cOpt.Mode, tmOpts.Mode)
		}

		// just execute the function
		return f(ctx)
	}

	// transaction is not started yet
	return tm.tmBeginner.Begin(ctx, f, *tmOpts)
}

// WithoutTransaction returns context without transaction.
func (tm *TransactionManager) WithoutTransaction(ctx context.Context) context.Context {
	return tm.tmBeginner.WithoutTransaction(ctx)
}
