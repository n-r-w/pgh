package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n-r-w/pgh/v2/txmgr"
)

func (p *PxDB) beginTxHelper(ctx context.Context, opts txmgr.Options) (*pgxpool.Conn, pgx.Tx, error) {
	con, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	if p.testHookAfterAcquire != nil {
		p.testHookAfterAcquire()
	}

	//nolint:exhaustruct // external type, only set necessary fields
	tx, err := con.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   getPgxLevel(opts.Level),
		AccessMode: getPgxMode(opts.Mode),
	})
	if err != nil {
		con.Release()
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return con, tx, nil
}

// Begin runs a function within a transaction.
func (p *PxDB) Begin(ctx context.Context, f func(ctxTr context.Context) error, opts txmgr.Options) (err error) {
	con, tx, err := p.beginTxHelper(ctx, opts)
	if err != nil {
		return err
	}

	// If panic occurs, rollback the transaction.
	defer func() {
		defer con.Release()

		if rec := recover(); rec != nil {
			_ = tx.Rollback(ctx)
			panic(rec) // Re-throw panic after rollback.
		}
	}()

	defer func() {
		errRollback := tx.Rollback(ctx)
		if errRollback != nil && !errors.Is(errRollback, pgx.ErrTxClosed) {
			if err != nil {
				err = fmt.Errorf("%w (rollback error: %v)", err, errRollback) //nolint:errorlint // ok for 2 errors
			} else {
				err = errRollback
			}
		}
	}()

	// Create transaction object and put it in context
	tCtx := newTransaction(p, tx, opts).toContext(ctx)

	if err = f(tCtx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// BeginTx begins a new transaction with the provided options.
func (p *PxDB) BeginTx(ctx context.Context, opts txmgr.Options) (context.Context, txmgr.ITransactionFinisher, error) {
	con, tx, err := p.beginTxHelper(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	// Create transaction object and put it in context
	tCtx := newTransaction(p, tx, opts).toContext(ctx)

	return tCtx, &transactionFinisher{
		con: con,
		tx:  tx,
	}, nil
}

// InTransaction returns true if transaction is started.
func (p *PxDB) InTransaction(ctx context.Context) bool {
	_, ok := txFromContext(ctx)
	return ok
}

// TransactionOptions returns transaction parameters. If transaction is not started, returns false.
func (p *PxDB) TransactionOptions(ctx context.Context) txmgr.Options {
	tx, ok := txFromContext(ctx)
	if !ok {
		//nolint:exhaustruct // external type, zero values are acceptable defaults
		return txmgr.Options{}
	}

	return tx.opts
}

// WithoutTransaction returns context without transaction.
func (p *PxDB) WithoutTransaction(ctx context.Context) context.Context {
	return WithoutTransaction(ctx)
}

// getPgxLevel returns pgx isolation level.
func getPgxLevel(level txmgr.TransactionLevel) pgx.TxIsoLevel {
	switch level {
	case txmgr.TxReadUncommitted:
		return pgx.ReadUncommitted
	case txmgr.TxReadCommitted:
		return pgx.ReadCommitted
	case txmgr.TxRepeatableRead:
		return pgx.RepeatableRead
	case txmgr.TxSerializable:
		return pgx.Serializable
	case txmgr.TxLevelDefault:
		return pgx.ReadCommitted
	default:
		panic("internal error")
	}
}

// getPgxMode returns pgx transaction mode.
func getPgxMode(mode txmgr.TransactionMode) pgx.TxAccessMode {
	switch mode {
	case txmgr.TxReadOnly:
		return pgx.ReadOnly
	case txmgr.TxReadWrite:
		return pgx.ReadWrite
	case txmgr.TxModeDefault:
		return pgx.ReadWrite
	default:
		panic("internal error")
	}
}

type txKeyType int

// txKey key for storing transaction in context.
const txKey txKeyType = 0

// transaction stores transaction information.
type transaction struct {
	db   *PxDB
	tx   pgx.Tx
	opts txmgr.Options
}

func newTransaction(db *PxDB, tx pgx.Tx, opts txmgr.Options) *transaction {
	if db == nil || tx == nil {
		panic("invalid arguments") // just in case
	}

	return &transaction{
		db:   db,
		tx:   tx,
		opts: opts,
	}
}

// toContext puts transaction in context.
func (t *transaction) toContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, txKey, t)
}

// removeFromContext removes transaction from context.
func (t *transaction) removeFromContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, txKey, nil)
}

// txFromContext extracts transaction from context.
func txFromContext(ctx context.Context) (*transaction, bool) {
	it, ok := ctx.Value(txKey).(*transaction)

	if !ok || it == nil {
		return nil, false
	}

	return it, true
}

// WithoutTransaction returns context without transaction.
func WithoutTransaction(ctx context.Context) context.Context {
	tx, ok := txFromContext(ctx)
	if !ok {
		return ctx
	}

	return tx.removeFromContext(ctx)
}

// transactionFinisher implements txmgr.ITransactionFinisher.
type transactionFinisher struct {
	con *pgxpool.Conn
	tx  pgx.Tx
}

// Commit commits the transaction.
func (t *transactionFinisher) Commit(ctx context.Context) error {
	defer t.con.Release()
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (t *transactionFinisher) Rollback(ctx context.Context) error {
	defer t.con.Release()
	return t.tx.Rollback(ctx)
}
