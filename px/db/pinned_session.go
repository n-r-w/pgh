package db

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPinnedSessionClosed reports use of a pinned session after its scope ended.
var ErrPinnedSessionClosed = errors.New("pinned session is closed")

// ErrPinnedSessionTransactionActive reports use of a pinned session outside its active transaction.
var ErrPinnedSessionTransactionActive = errors.New("pinned session transaction is active")

// PinnedSessionOption configures a pinned session scope.
type PinnedSessionOption func(*pinnedSessionOptions)

// pinnedSessionOptions contains pinned session cleanup options.
type pinnedSessionOptions struct {
	closeConnection bool
}

// WithPinnedSessionCloseConnection closes the acquired connection when the pinned session scope ends.
func WithPinnedSessionCloseConnection() PinnedSessionOption {
	return func(opts *pinnedSessionOptions) {
		opts.closeConnection = true
	}
}

type pinnedSessionKeyType int

const pinnedSessionKey pinnedSessionKeyType = 0

// pinnedSession owns one pool connection for the callback lifetime.
type pinnedSession struct {
	mu       sync.Mutex
	db       *PxDB
	conn     *pgxpool.Conn
	ended    bool
	activeTx *transaction
}

// RunPinnedSession acquires one connection and pins database work in f to it.
func (p *PxDB) RunPinnedSession(
	ctx context.Context,
	f func(context.Context) error,
	opts ...PinnedSessionOption,
) (err error) {
	if session, ok := pinnedSessionFromContext(ctx); ok {
		if session.isEnded() {
			return ErrPinnedSessionClosed
		}
		return errors.New("pinned session is already active")
	}
	if _, ok := txFromContext(ctx); ok {
		return errors.New("pinned session cannot start from a transaction context")
	}

	options := pinnedSessionOptions{}
	for _, option := range opts {
		option(&options)
	}

	connection, err := p.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire pinned session connection: %w", err)
	}

	session := &pinnedSession{
		mu:       sync.Mutex{},
		db:       p,
		conn:     connection,
		ended:    false,
		activeTx: nil,
	}
	sessionCtx := context.WithValue(ctx, pinnedSessionKey, session)

	defer func() {
		cleanupErr := session.end(ctx, options.closeConnection)
		if rec := recover(); rec != nil {
			panic(rec)
		}
		err = errors.Join(err, cleanupErr)
	}()

	return f(sessionCtx)
}

// begin starts a transaction on the pinned connection and records its owner.
func (s *pinnedSession) begin(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return nil, ErrPinnedSessionClosed
	}
	if s.activeTx != nil {
		return nil, ErrPinnedSessionTransactionActive
	}

	tx, err := s.conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// setActiveTransaction records the transaction that currently borrows the session connection.
func (s *pinnedSession) setActiveTransaction(tx *transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeTx = tx
}

// finishTransaction releases transaction ownership without releasing the session connection.
func (s *pinnedSession) finishTransaction(tx *transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTx == tx {
		s.activeTx = nil
	}
}

// operationError validates session and transaction state before an operation.
func (s *pinnedSession) operationError(tx *transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ended {
		return ErrPinnedSessionClosed
	}
	if tx != nil {
		if tx.isFinished() {
			return pgx.ErrTxClosed
		}
		return nil
	}
	if s.activeTx != nil {
		return ErrPinnedSessionTransactionActive
	}
	return nil
}

// isEnded reports whether the session scope has ended.
func (s *pinnedSession) isEnded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ended
}

// end invalidates the session, rolls back unfinished work, and disposes of the connection.
func (s *pinnedSession) end(ctx context.Context, closeConnection bool) error {
	s.mu.Lock()
	s.ended = true
	activeTx := s.activeTx
	s.activeTx = nil
	s.mu.Unlock()

	var cleanupErr error
	if activeTx != nil && !activeTx.isFinished() {
		rollbackErr := activeTx.tx.Rollback(ctx)
		activeTx.markFinished()
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			cleanupErr = errors.Join(ErrPinnedSessionTransactionActive, rollbackErr)
		} else {
			cleanupErr = ErrPinnedSessionTransactionActive
		}
	}

	if closeConnection {
		connection := s.conn.Hijack()
		cleanupErr = errors.Join(cleanupErr, connection.Close(ctx))
	} else {
		s.conn.Release()
	}

	return cleanupErr
}

// pinnedSessionFromContext extracts a pinned session from a context.
func pinnedSessionFromContext(ctx context.Context) (*pinnedSession, bool) {
	session, ok := ctx.Value(pinnedSessionKey).(*pinnedSession)
	return session, ok && session != nil
}
