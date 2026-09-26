package db

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n-r-w/pgh/v2/txmgr"
)

func (p *PxDB) beginTxHelper(ctx context.Context, opts txmgr.Options) (*pgxpool.Conn, pgx.Tx, error) {
	session, pinned, err := p.pinnedSessionForBegin(ctx)
	if err != nil {
		return nil, nil, err
	}

	pgxOpts := pgx.TxOptions{ //nolint:exhaustruct // external type, only set necessary fields
		IsoLevel:   getPgxLevel(opts.Level),
		AccessMode: getPgxMode(opts.Mode),
	}
	if pinned {
		tx, beginErr := session.begin(ctx, pgxOpts)
		return session.conn, tx, beginErr
	}

	con, err := p.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire connection: %w", err)
	}

	if p.testHookAfterAcquire != nil {
		p.testHookAfterAcquire()
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		con.Release()
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", ctxErr)
	}

	tx, err := con.BeginTx(ctx, pgxOpts)
	if err != nil {
		con.Release()
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return con, tx, nil
}

// pinnedSessionForBegin validates pinned-session context state before a transaction begins.
func (p *PxDB) pinnedSessionForBegin(ctx context.Context) (*pinnedSession, bool, error) {
	session, ok := pinnedSessionFromContext(ctx)
	if !ok {
		return nil, false, nil
	}
	if session.db != p {
		return nil, false, errors.New("pinned session belongs to another database")
	}
	if session.isEnded() {
		return nil, false, ErrPinnedSessionClosed
	}

	existing, ok := txFromContext(ctx)
	if !ok {
		return session, true, nil
	}
	if existing.isFinished() {
		return nil, false, pgx.ErrTxClosed
	}
	return nil, false, ErrPinnedSessionTransactionActive
}

// Begin runs a function within a transaction.
func (p *PxDB) Begin(ctx context.Context, f func(ctxTr context.Context) error, opts txmgr.Options) (err error) {
	ctxTr, finisher, err := p.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		rec := recover()
		rollbackErr := finisher.Rollback(ctxTr)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, rollbackErr)
		}
		if rec != nil {
			panic(rec)
		}
	}()

	if err = f(ctxTr); err != nil {
		return err
	}
	return finisher.Commit(ctxTr)
}

// BeginTx begins a new transaction with the provided options.
func (p *PxDB) BeginTx(ctx context.Context, opts txmgr.Options) (context.Context, txmgr.ITransactionFinisher, error) {
	con, pgxTx, err := p.beginTxHelper(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	session, pinned := pinnedSessionFromContext(ctx)
	tx := newTransaction(p, pgxTx, opts, session)
	if pinned {
		session.setActiveTransaction(tx)
	}
	txCtx := tx.toContext(ctx)

	return txCtx, &transactionFinisher{
		con:     con,
		tx:      tx,
		session: session,
	}, nil
}

// InTransaction returns true if transaction is started and active.
func (*PxDB) InTransaction(ctx context.Context) bool {
	tx, ok := txFromContext(ctx)
	if !ok {
		return false
	}
	if tx.session == nil {
		return true
	}
	return !tx.isFinished() && !tx.session.conn.Conn().IsClosed()
}

// TransactionOptions returns transaction parameters. If transaction is not started, returns defaults.
func (*PxDB) TransactionOptions(ctx context.Context) txmgr.Options {
	tx, ok := txFromContext(ctx)
	if !ok {
		return txmgr.Options{}
	}
	if tx.session != nil && tx.isFinished() {
		return txmgr.Options{}
	}
	return tx.opts
}

// WithoutTransaction returns context without transaction.
func (*PxDB) WithoutTransaction(ctx context.Context) context.Context {
	return WithoutTransaction(ctx)
}

// getPgxLevel returns pgx isolation level.
func getPgxLevel(level txmgr.TransactionLevel) pgx.TxIsoLevel {
	switch level {
	case txmgr.TxReadUncommitted:
		return pgx.ReadUncommitted
	case txmgr.TxReadCommitted, txmgr.TxLevelDefault:
		return pgx.ReadCommitted
	case txmgr.TxRepeatableRead:
		return pgx.RepeatableRead
	case txmgr.TxSerializable:
		return pgx.Serializable
	default:
		panic("internal error")
	}
}

// getPgxMode returns pgx transaction mode.
func getPgxMode(mode txmgr.TransactionMode) pgx.TxAccessMode {
	switch mode {
	case txmgr.TxReadOnly:
		return pgx.ReadOnly
	case txmgr.TxReadWrite, txmgr.TxModeDefault:
		return pgx.ReadWrite
	default:
		panic("internal error")
	}
}

type txKeyType int

// txKey is the key for storing a transaction in a context.
const txKey txKeyType = 0

// transaction stores transaction information and lifecycle state.
type transaction struct {
	mu       sync.Mutex
	db       *PxDB
	tx       pgx.Tx
	opts     txmgr.Options
	session  *pinnedSession
	finished bool
}

func newTransaction(db *PxDB, tx pgx.Tx, opts txmgr.Options, session *pinnedSession) *transaction {
	if db == nil || tx == nil {
		panic("invalid arguments")
	}

	return &transaction{
		mu:       sync.Mutex{},
		db:       db,
		tx:       tx,
		opts:     opts,
		session:  session,
		finished: false,
	}
}

// toContext puts the transaction in a context.
func (t *transaction) toContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, txKey, t)
}

// removeFromContext removes the transaction from a context.
func (*transaction) removeFromContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, txKey, nil)
}

// isFinished reports whether commit or rollback was attempted.
func (t *transaction) isFinished() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.finished
}

// markFinished records that the transaction cannot be used again.
func (t *transaction) markFinished() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.finished = true
}

// txFromContext extracts a transaction from a context.
func txFromContext(ctx context.Context) (*transaction, bool) {
	it, ok := ctx.Value(txKey).(*transaction)
	return it, ok && it != nil
}

// WithoutTransaction returns a context without transaction state.
func WithoutTransaction(ctx context.Context) context.Context {
	tx, ok := txFromContext(ctx)
	if !ok {
		return ctx
	}
	return tx.removeFromContext(ctx)
}

// transactionFinisher implements txmgr.ITransactionFinisher.
type transactionFinisher struct {
	con     *pgxpool.Conn
	tx      *transaction
	session *pinnedSession
}

// Commit commits the transaction.
func (t *transactionFinisher) Commit(ctx context.Context) error {
	return t.finish(ctx, true)
}

// Rollback rolls back the transaction.
func (t *transactionFinisher) Rollback(ctx context.Context) error {
	return t.finish(ctx, false)
}

// finish finalizes the transaction and releases only non-session connections.
func (t *transactionFinisher) finish(ctx context.Context, commit bool) error {
	if t.tx.isFinished() {
		return pgx.ErrTxClosed
	}

	var err error
	if commit {
		err = t.tx.tx.Commit(ctx)
	} else {
		err = t.tx.tx.Rollback(ctx)
	}
	t.tx.markFinished()

	if t.session != nil {
		t.session.finishTransaction(t.tx)
	} else {
		t.con.Release()
	}
	return err
}
