package db

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/require"
)

// TestRunPinnedSession_AllConnectionOperationsUsePinnedBackend verifies that ordinary operations share one backend.
func TestRunPinnedSession_AllConnectionOperationsUsePinnedBackend(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	acquireCount := pool.Stat().AcquireCount()

	err := database.RunPinnedSession(t.Context(), func(ctx context.Context) error {
		connection := database.Connection(ctx)
		var sessionPID int
		require.NoError(t, connection.QueryRow(ctx, "select pg_backend_pid()").Scan(&sessionPID))
		require.NotZero(t, sessionPID)

		_, err := connection.Exec(ctx, "create temporary table pinned_values (id integer)")
		require.NoError(t, err)

		rows, err := connection.Query(ctx, "select pg_backend_pid()")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var queryPID int
		require.NoError(t, rows.Scan(&queryPID))
		rows.Close()
		require.NoError(t, rows.Err())
		require.Equal(t, sessionPID, queryPID)

		batch := &pgx.Batch{}
		batch.Queue("select pg_backend_pid()")
		batchResult := connection.SendBatch(ctx, batch)
		var batchPID int
		require.NoError(t, batchResult.QueryRow().Scan(&batchPID))
		require.NoError(t, batchResult.Close())
		require.Equal(t, sessionPID, batchPID)

		copied, err := connection.CopyFrom(
			ctx,
			pgx.Identifier{"pinned_values"},
			[]string{"id"},
			pgx.CopyFromRows([][]any{{1}, {2}}),
		)
		require.NoError(t, err)
		require.EqualValues(t, 2, copied)

		var count int
		require.NoError(t, connection.QueryRow(ctx, "select count(*) from pinned_values").Scan(&count))
		require.Equal(t, 2, count)
		require.Equal(t, acquireCount+1, pool.Stat().AcquireCount())
		return nil
	})

	require.NoError(t, err)
	require.EqualValues(t, 0, pool.Stat().AcquiredConns())
}

// TestRunPinnedSession_SequentialOperationsUseOneBackend verifies that short transactions do not end the session.
func TestRunPinnedSession_SequentialOperationsUseOneBackend(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)

	err := database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		pids := make([]int, 0, 3)
		session, ok := pinnedSessionFromContext(sessionCtx)
		require.True(t, ok)
		for step := range 3 {
			if step == 1 {
				var pid int
				require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&pid))
				pids = append(pids, pid)
				require.Equal(t, byte('I'), session.conn.Conn().PgConn().TxStatus())
				continue
			}

			require.NoError(t, tm.Begin(sessionCtx, func(txCtx context.Context) error {
				var pid int
				require.NoError(t, database.Connection(txCtx).QueryRow(txCtx, "select pg_backend_pid()").Scan(&pid))
				pids = append(pids, pid)
				return nil
			}))
			require.Equal(t, byte('I'), session.conn.Conn().PgConn().TxStatus())
		}

		require.Len(t, pids, 3)
		require.Equal(t, pids[0], pids[1])
		require.Equal(t, pids[0], pids[2])
		return nil
	})

	require.NoError(t, err)
	require.EqualValues(t, 0, pool.Stat().AcquiredConns())
}

// TestRunPinnedSession_ManualCommitAndRollbackKeepBackend verifies manual transaction results and connection ownership.
func TestRunPinnedSession_ManualCommitAndRollbackKeepBackend(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	require.NoError(t, func() error {
		_, err := pool.Exec(t.Context(), "create table pinned_manual (id integer primary key)")
		return err
	}())

	err := database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var sessionPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&sessionPID))

		commitCtx, commitFinisher, err := tm.BeginTx(sessionCtx)
		require.NoError(t, err)
		var commitPID int
		require.NoError(t, database.Connection(commitCtx).QueryRow(commitCtx, "select pg_backend_pid()").Scan(&commitPID))
		_, err = database.Connection(commitCtx).Exec(commitCtx, "insert into pinned_manual values (1)")
		require.NoError(t, err)
		require.NoError(t, commitFinisher.Commit(commitCtx))
		require.ErrorIs(t, commitFinisher.Commit(commitCtx), pgx.ErrTxClosed)

		rollbackCtx, rollbackFinisher, err := tm.BeginTx(sessionCtx)
		require.NoError(t, err)
		var rollbackPID int
		require.NoError(t, database.Connection(rollbackCtx).QueryRow(rollbackCtx, "select pg_backend_pid()").Scan(&rollbackPID))
		_, err = database.Connection(rollbackCtx).Exec(rollbackCtx, "insert into pinned_manual values (2)")
		require.NoError(t, err)
		require.NoError(t, rollbackFinisher.Rollback(rollbackCtx))
		require.ErrorIs(t, rollbackFinisher.Rollback(rollbackCtx), pgx.ErrTxClosed)

		var ids []int
		rows, err := database.Connection(sessionCtx).Query(sessionCtx, "select id from pinned_manual order by id")
		require.NoError(t, err)
		for rows.Next() {
			var id int
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		rows.Close()
		require.NoError(t, rows.Err())
		require.Equal(t, []int{1}, ids)
		require.Equal(t, sessionPID, commitPID)
		require.Equal(t, sessionPID, rollbackPID)
		return nil
	})
	require.NoError(t, err)
}

// TestRunPinnedSession_NestedTransactionsRemainUnderOuterControl verifies callback and manual nested participation.
func TestRunPinnedSession_NestedTransactionsRemainUnderOuterControl(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pinned_nested (id integer)")
	require.NoError(t, err)
	callbackErr := errors.New("outer transaction failed")

	err = database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		return tm.Begin(sessionCtx, func(outerCtx context.Context) error {
			var outerPID int
			require.NoError(t, database.Connection(outerCtx).QueryRow(outerCtx, "select pg_backend_pid()").Scan(&outerPID))

			require.NoError(t, tm.Begin(outerCtx, func(innerCtx context.Context) error {
				require.Equal(t, outerCtx, innerCtx)
				var innerPID int
				require.NoError(t, database.Connection(innerCtx).QueryRow(innerCtx, "select pg_backend_pid()").Scan(&innerPID))
				require.Equal(t, outerPID, innerPID)
				_, innerErr := database.Connection(innerCtx).Exec(innerCtx, "insert into pinned_nested values (1)")
				return innerErr
			}))

			manualCtx, finisher, manualErr := tm.BeginTx(outerCtx)
			require.NoError(t, manualErr)
			require.Equal(t, outerCtx, manualCtx)
			require.NoError(t, finisher.Commit(manualCtx))
			_, manualErr = database.Connection(manualCtx).Exec(manualCtx, "insert into pinned_nested values (2)")
			require.NoError(t, manualErr)
			return callbackErr
		})
	})
	require.ErrorIs(t, err, callbackErr)

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), "select count(*) from pinned_nested").Scan(&count))
	require.Zero(t, count)
}

// TestRunPinnedSession_CallbackErrorRollsBackAndKeepsSession verifies rollback and post-rollback hook routing.
func TestRunPinnedSession_CallbackErrorRollsBackAndKeepsSession(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pinned_rollback (id integer)")
	require.NoError(t, err)
	callbackErr := errors.New("operation failed")

	err = database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var sessionPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&sessionPID))

		txErr := tm.Begin(sessionCtx, func(txCtx context.Context) error {
			_, execErr := database.Connection(txCtx).Exec(txCtx, "insert into pinned_rollback values (1)")
			require.NoError(t, execErr)
			require.NoError(t, txmgr.AfterRollback(txCtx, func(hookCtx context.Context) {
				var hookPID int
				require.NoError(t, database.Connection(hookCtx).QueryRow(hookCtx, "select pg_backend_pid()").Scan(&hookPID))
				require.Equal(t, sessionPID, hookPID)
			}))
			return callbackErr
		})
		require.ErrorIs(t, txErr, callbackErr)

		var count int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select count(*) from pinned_rollback").Scan(&count))
		require.Zero(t, count)
		return nil
	})
	require.NoError(t, err)
}

// TestRunPinnedSession_EndedContextAndWrappersStayEnded verifies that saved values never fall back to the pool.
func TestRunPinnedSession_EndedContextAndWrappersStayEnded(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	var savedCtx context.Context
	var savedWrapper conn.IConnection
	var savedTxCtx context.Context
	var savedTxWrapper conn.IConnection

	require.NoError(t, database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		savedCtx = sessionCtx
		savedWrapper = database.Connection(sessionCtx)
		var err error
		var finisher txmgr.ITransactionFinisher
		savedTxCtx, finisher, err = tm.BeginTx(sessionCtx)
		require.NoError(t, err)
		savedTxWrapper = database.Connection(savedTxCtx)
		require.NoError(t, finisher.Commit(savedTxCtx))
		_, err = savedTxWrapper.Exec(savedTxCtx, "select 1")
		require.ErrorIs(t, err, pgx.ErrTxClosed)
		return nil
	}))

	acquireCount := pool.Stat().AcquireCount()
	_, err := savedWrapper.Exec(savedCtx, "select 1")
	require.ErrorIs(t, err, ErrPinnedSessionClosed)
	_, err = savedTxWrapper.Exec(savedTxCtx, "select 1")
	require.ErrorIs(t, err, ErrPinnedSessionClosed)
	require.ErrorIs(t, database.Connection(savedCtx).QueryRow(savedCtx, "select 1").Scan(new(int)), ErrPinnedSessionClosed)
	batchRequest := &pgx.Batch{}
	batchRequest.Queue("select 1")
	batch := database.Connection(savedCtx).SendBatch(savedCtx, batchRequest)
	require.ErrorIs(t, batch.Close(), ErrPinnedSessionClosed)
	require.ErrorIs(t, tm.Begin(savedCtx, func(context.Context) error { return nil }), ErrPinnedSessionClosed)
	require.Equal(t, acquireCount, pool.Stat().AcquireCount())
}

// TestRunPinnedSession_WithoutTransactionPreservesPin verifies active-transaction exclusion and hook routing.
func TestRunPinnedSession_WithoutTransactionPreservesPin(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)

	require.NoError(t, database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var sessionPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&sessionPID))
		return tm.Begin(sessionCtx, func(txCtx context.Context) error {
			_, err := database.Connection(sessionCtx).Exec(sessionCtx, "select 1")
			require.ErrorIs(t, err, ErrPinnedSessionTransactionActive)
			withoutTx := tm.WithoutTransaction(txCtx)
			_, err = database.Connection(withoutTx).Exec(withoutTx, "select 1")
			require.ErrorIs(t, err, ErrPinnedSessionTransactionActive)
			require.NoError(t, txmgr.AfterCommit(txCtx, func(hookCtx context.Context) {
				var hookPID int
				require.NoError(t, database.Connection(hookCtx).QueryRow(hookCtx, "select pg_backend_pid()").Scan(&hookPID))
				require.Equal(t, sessionPID, hookPID)
			}))
			return nil
		})
	}))
}

// TestRunPinnedSession_LostConnectionDoesNotSwitchBackend verifies that a broken pin is not replaced.
func TestRunPinnedSession_LostConnectionDoesNotSwitchBackend(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pinned_lost (id integer)")
	require.NoError(t, err)

	err = database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var pid int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&pid))
		_, err := pool.Exec(t.Context(), "select pg_terminate_backend($1)", pid)
		require.NoError(t, err)
		acquireCount := pool.Stat().AcquireCount()

		_, err = database.Connection(sessionCtx).Exec(sessionCtx, "insert into pinned_lost values (1)")
		require.Error(t, err)
		require.Error(t, tm.Begin(sessionCtx, func(context.Context) error { return nil }))
		require.Equal(t, acquireCount, pool.Stat().AcquireCount())
		return nil
	})
	require.NoError(t, err)

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), "select count(*) from pinned_lost").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, database.RunPinnedSession(t.Context(), func(ctx context.Context) error {
		return database.Connection(ctx).QueryRow(ctx, "select 1").Scan(new(int))
	}))
}

// TestRunPinnedSession_CommitErrorIsReturnedWithoutReacquire verifies deterministic server-side commit failure.
func TestRunPinnedSession_CommitErrorIsReturnedWithoutReacquire(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), `create table pinned_commit (
		id integer,
		constraint pinned_commit_unique unique (id) deferrable initially deferred
	)`)
	require.NoError(t, err)

	require.NoError(t, database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var pid int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&pid))
		commitErr := tm.Begin(sessionCtx, func(txCtx context.Context) error {
			_, execErr := database.Connection(txCtx).Exec(txCtx, "insert into pinned_commit values (1), (1)")
			return execErr
		})
		require.Error(t, commitErr)
		acquireCount := pool.Stat().AcquireCount()
		var followupPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&followupPID))
		require.Equal(t, pid, followupPID)
		require.Equal(t, acquireCount, pool.Stat().AcquireCount())
		return nil
	}))

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), "select count(*) from pinned_commit").Scan(&count))
	require.Zero(t, count)
}

// TestRunPinnedSession_RollbackErrorReturnsBothErrors verifies callback and rollback error composition.
func TestRunPinnedSession_RollbackErrorReturnsBothErrors(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		run  func(*PxDB, *txmgr.TransactionManager, context.Context, func(context.Context) error) error
	}{
		{
			name: "transaction manager",
			run: func(_ *PxDB, tm *txmgr.TransactionManager, ctx context.Context, callback func(context.Context) error) error {
				return tm.Begin(ctx, callback)
			},
		},
		{
			name: "database beginner",
			run: func(database *PxDB, _ *txmgr.TransactionManager, ctx context.Context, callback func(context.Context) error) error {
				return database.Begin(ctx, callback, txmgr.Options{})
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
			database := New(WithPool(pool))
			tm := txmgr.New(database, database)
			callbackErr := errors.New("callback failed")
			var rollbackErr error

			sessionErr := database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
				txErr := testCase.run(database, tm, sessionCtx, func(txCtx context.Context) error {
					var pid int
					require.NoError(t, database.Connection(txCtx).QueryRow(txCtx, "select pg_backend_pid()").Scan(&pid))
					_, terminateErr := pool.Exec(t.Context(), "select pg_terminate_backend($1)", pid)
					require.NoError(t, terminateErr)
					return callbackErr
				})
				require.ErrorIs(t, txErr, callbackErr)
				require.Error(t, txErr)
				joined, ok := txErr.(interface{ Unwrap() []error })
				require.True(t, ok)
				for _, joinedErr := range joined.Unwrap() {
					if !errors.Is(joinedErr, callbackErr) {
						rollbackErr = joinedErr
					}
				}
				require.Error(t, rollbackErr)
				require.ErrorIs(t, txErr, rollbackErr)
				acquireCount := pool.Stat().AcquireCount()
				_, followupErr := database.Connection(sessionCtx).Exec(sessionCtx, "select 1")
				require.Error(t, followupErr)
				require.Equal(t, acquireCount, pool.Stat().AcquireCount())
				return nil
			})
			require.NoError(t, sessionErr)
		})
	}
}

// TestRunPinnedSession_ReleasesOnSuccessErrorAndCancellation verifies default cleanup for all callback outcomes.
func TestRunPinnedSession_ReleasesOnSuccessErrorAndCancellation(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		run  func(context.Context, context.CancelFunc) error
	}{
		{name: "success", run: func(context.Context, context.CancelFunc) error { return nil }},
		{name: "error", run: func(context.Context, context.CancelFunc) error { return errors.New("callback failed") }},
		{name: "cancellation", run: func(ctx context.Context, cancel context.CancelFunc) error {
			cancel()
			return ctx.Err()
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
			database := New(WithPool(pool))
			ctx, cancel := context.WithCancel(t.Context())
			var savedCtx context.Context

			err := database.RunPinnedSession(ctx, func(sessionCtx context.Context) error {
				savedCtx = sessionCtx
				return testCase.run(sessionCtx, cancel)
			})
			if testCase.name == "success" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.EqualValues(t, 0, pool.Stat().AcquiredConns())
			_, staleErr := database.Connection(savedCtx).Exec(savedCtx, "select 1")
			require.ErrorIs(t, staleErr, ErrPinnedSessionClosed)
			require.NoError(t, pool.Ping(t.Context()))
		})
	}
}

// TestRunPinnedSession_AcquireCancellationDoesNotRunCallbackOrLeak verifies cancellation while waiting for the pin.
func TestRunPinnedSession_AcquireCancellationDoesNotRunCallbackOrLeak(t *testing.T) {
	t.Parallel()

	basePool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	config := basePool.Config().Copy()
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	held, err := pool.Acquire(t.Context())
	require.NoError(t, err)

	database := New(WithPool(pool))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	callbackRan := false
	err = database.RunPinnedSession(ctx, func(context.Context) error {
		callbackRan = true
		return nil
	})
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, callbackRan)
	require.EqualValues(t, 1, pool.Stat().AcquiredConns())
	held.Release()
	require.NoError(t, pool.Ping(t.Context()))
}

// TestRunPinnedSession_UnfinishedManualTransactionIsRolledBack verifies cleanup of abandoned transactions.
func TestRunPinnedSession_UnfinishedManualTransactionIsRolledBack(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pinned_unfinished (id integer)")
	require.NoError(t, err)
	var finisher txmgr.ITransactionFinisher
	var txCtx context.Context

	err = database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var beginErr error
		txCtx, finisher, beginErr = tm.BeginTx(sessionCtx)
		require.NoError(t, beginErr)
		_, beginErr = database.Connection(txCtx).Exec(txCtx, "insert into pinned_unfinished values (1)")
		return beginErr
	})
	require.ErrorIs(t, err, ErrPinnedSessionTransactionActive)
	require.EqualValues(t, 0, pool.Stat().AcquiredConns())
	require.ErrorIs(t, finisher.Commit(txCtx), pgx.ErrTxClosed)

	var count int
	require.NoError(t, pool.QueryRow(t.Context(), "select count(*) from pinned_unfinished").Scan(&count))
	require.Zero(t, count)
}

// TestRunPinnedSession_CloseOptionClosesInsteadOfReleasing verifies connection disposal without closing the pool.
func TestRunPinnedSession_CloseOptionClosesInsteadOfReleasing(t *testing.T) {
	t.Parallel()

	basePool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	config := basePool.Config().Copy()
	config.MaxConns = 1
	config.MinConns = 0
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	database := New(WithPool(pool))
	var sessionPID int

	require.NoError(t, database.RunPinnedSession(t.Context(), func(ctx context.Context) error {
		return database.Connection(ctx).QueryRow(ctx, "select pg_backend_pid()").Scan(&sessionPID)
	}, WithPinnedSessionCloseConnection()))
	require.EqualValues(t, 0, pool.Stat().AcquiredConns())

	var backendCount int
	require.NoError(t, basePool.QueryRow(
		t.Context(),
		"select count(*) from pg_stat_activity where pid = $1",
		sessionPID,
	).Scan(&backendCount))
	require.Zero(t, backendCount)

	var nextPID int
	require.NoError(t, pool.QueryRow(t.Context(), "select pg_backend_pid()").Scan(&nextPID))
	require.NotEqual(t, sessionPID, nextPID)
}

// TestRunPinnedSession_IndependentSessionsAndPoolWorkContinue verifies independent pins and pool ownership.
func TestRunPinnedSession_IndependentSessionsAndPoolWorkContinue(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	type sessionResult struct {
		pid int
		err error
	}
	readyA := make(chan sessionResult, 1)
	readyB := make(chan sessionResult, 1)
	followupB := make(chan sessionResult, 1)
	releaseA := make(chan struct{})
	continueB := make(chan struct{})
	doneA := make(chan error, 1)
	doneB := make(chan error, 1)

	go func() {
		doneA <- database.RunPinnedSession(ctx, func(sessionCtx context.Context) error {
			var pid int
			err := database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&pid)
			readyA <- sessionResult{pid: pid, err: err}
			select {
			case <-releaseA:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	go func() {
		doneB <- database.RunPinnedSession(ctx, func(sessionCtx context.Context) error {
			var pid int
			err := database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&pid)
			readyB <- sessionResult{pid: pid, err: err}
			select {
			case <-continueB:
			case <-ctx.Done():
				return ctx.Err()
			}
			var followupPID int
			err = database.Connection(sessionCtx).QueryRow(sessionCtx, "select pg_backend_pid()").Scan(&followupPID)
			followupB <- sessionResult{pid: followupPID, err: err}
			return nil
		})
	}()

	receiveReady := func(ready <-chan sessionResult, done <-chan error) sessionResult {
		t.Helper()
		select {
		case result := <-ready:
			return result
		case err := <-done:
			require.NoError(t, err)
			return sessionResult{}
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
			return sessionResult{}
		}
	}
	resultA := receiveReady(readyA, doneA)
	resultB := receiveReady(readyB, doneB)
	require.NoError(t, resultA.err)
	require.NoError(t, resultB.err)
	var poolPID int
	require.NoError(t, pool.QueryRow(ctx, "select pg_backend_pid()").Scan(&poolPID))
	require.NotEqual(t, resultA.pid, resultB.pid)
	require.NotEqual(t, resultA.pid, poolPID)
	require.NotEqual(t, resultB.pid, poolPID)

	close(releaseA)
	select {
	case err := <-doneA:
		require.NoError(t, err)
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	close(continueB)
	var resultBAfterA sessionResult
	select {
	case resultBAfterA = <-followupB:
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	require.NoError(t, resultBAfterA.err)
	require.Equal(t, resultB.pid, resultBAfterA.pid)
	select {
	case err := <-doneB:
		require.NoError(t, err)
	case <-ctx.Done():
		require.NoError(t, ctx.Err())
	}
	require.NoError(t, pool.Ping(ctx))
}

// TestWithoutPinnedSession_PreservesPoolAndBypassBehavior verifies compatibility outside pinned scopes.
func TestWithoutPinnedSession_PreservesPoolAndBypassBehavior(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pool_compatibility (id integer)")
	require.NoError(t, err)
	callbackErr := errors.New("rollback outer transaction")

	err = tm.Begin(t.Context(), func(txCtx context.Context) error {
		withoutTx := tm.WithoutTransaction(txCtx)
		_, execErr := database.Connection(withoutTx).Exec(withoutTx, "insert into pool_compatibility values (1)")
		require.NoError(t, execErr)
		return callbackErr
	})
	require.ErrorIs(t, err, callbackErr)

	var count int
	require.NoError(t, database.Connection(t.Context()).QueryRow(t.Context(), "select count(*) from pool_compatibility").Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, tm.Begin(t.Context(), func(context.Context) error { return nil }))
}

// TestRunPinnedSession_RejectsExistingSessionAndTransactionContexts verifies unambiguous scope ownership.
func TestRunPinnedSession_RejectsExistingSessionAndTransactionContexts(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)

	require.NoError(t, database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		acquireCount := pool.Stat().AcquireCount()
		callbackRan := false
		err := database.RunPinnedSession(sessionCtx, func(context.Context) error {
			callbackRan = true
			return nil
		})
		require.Error(t, err)
		require.False(t, callbackRan)
		require.Equal(t, acquireCount, pool.Stat().AcquireCount())
		return nil
	}))

	require.NoError(t, tm.Begin(t.Context(), func(txCtx context.Context) error {
		acquireCount := pool.Stat().AcquireCount()
		callbackRan := false
		err := database.RunPinnedSession(txCtx, func(context.Context) error {
			callbackRan = true
			return nil
		})
		require.Error(t, err)
		require.False(t, callbackRan)
		require.Equal(t, acquireCount, pool.Stat().AcquireCount())
		return nil
	}))
}

// TestRunPinnedSession_DatabaseOperationCancellationReleasesConnection verifies cancellation during SQL work.
func TestRunPinnedSession_DatabaseOperationCancellationReleasesConnection(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	ctx, cancel := context.WithCancel(t.Context())

	err := database.RunPinnedSession(ctx, func(sessionCtx context.Context) error {
		go func() {
			timer := time.NewTimer(25 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
				cancel()
			case <-sessionCtx.Done():
			}
		}()
		_, execErr := database.Connection(sessionCtx).Exec(sessionCtx, "select pg_sleep(10)")
		return execErr
	})
	require.Error(t, err)
	require.Eventually(t, func() bool {
		return pool.Stat().AcquiredConns() == 0
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, pool.Ping(t.Context()))
}

// TestRunPinnedSession_LostActiveTransactionCannotParticipate verifies nested calls fail after connection loss.
func TestRunPinnedSession_LostActiveTransactionCannotParticipate(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)

	err := database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		txCtx, _, beginErr := tm.BeginTx(sessionCtx)
		require.NoError(t, beginErr)
		var pid int
		require.NoError(t, database.Connection(txCtx).QueryRow(txCtx, "select pg_backend_pid()").Scan(&pid))
		_, terminateErr := pool.Exec(t.Context(), "select pg_terminate_backend($1)", pid)
		require.NoError(t, terminateErr)
		_, queryErr := database.Connection(txCtx).Exec(txCtx, "select 1")
		require.Error(t, queryErr)
		acquireCount := pool.Stat().AcquireCount()
		callbackRan := false
		nestedErr := tm.Begin(txCtx, func(context.Context) error {
			callbackRan = true
			return nil
		})
		require.Error(t, nestedErr)
		require.False(t, callbackRan)
		require.Equal(t, acquireCount, pool.Stat().AcquireCount())
		return nil
	})
	require.ErrorIs(t, err, ErrPinnedSessionTransactionActive)
}

// TestBegin_GoexitReleasesConnection verifies unconditional cleanup when a callback exits its goroutine.
func TestBegin_GoexitReleasesConnection(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	callbackExited := make(chan struct{})

	go func() {
		_ = database.Begin(t.Context(), func(context.Context) error {
			defer close(callbackExited)
			runtime.Goexit()
			return nil
		}, txmgr.Options{})
	}()
	<-callbackExited
	require.Eventually(t, func() bool {
		return pool.Stat().AcquiredConns() == 0
	}, time.Second, 10*time.Millisecond)
}

// TestWithoutPinnedSession_PreservesFinishedContextBehavior verifies legacy stale-context routing outside sessions.
func TestWithoutPinnedSession_PreservesFinishedContextBehavior(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	firstCtx, firstFinisher, err := database.BeginTx(t.Context(), txmgr.Options{})
	require.NoError(t, err)
	require.NoError(t, firstFinisher.Commit(firstCtx))
	require.True(t, database.InTransaction(firstCtx))
	require.True(t, database.Connection(firstCtx).InTransaction())

	secondCtx, secondFinisher, err := database.BeginTx(firstCtx, txmgr.Options{})
	require.NoError(t, err)
	require.NoError(t, secondFinisher.Rollback(secondCtx))
}

// TestRunPinnedSession_TransactionManagerGoexitRollsBackAndKeepsSession verifies fallback rollback after callback Goexit.
func TestRunPinnedSession_TransactionManagerGoexitRollsBackAndKeepsSession(t *testing.T) {
	t.Parallel()

	pool, _ := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	database := New(WithPool(pool))
	tm := txmgr.New(database, database)
	_, err := pool.Exec(t.Context(), "create table pinned_goexit (id integer)")
	require.NoError(t, err)

	err = database.RunPinnedSession(t.Context(), func(sessionCtx context.Context) error {
		var sessionPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(
			sessionCtx,
			"select pg_backend_pid()",
		).Scan(&sessionPID))

		transactionDone := make(chan struct{})
		unexpectedResult := make(chan error, 1)
		go func() {
			defer close(transactionDone)
			beginErr := tm.Begin(sessionCtx, func(txCtx context.Context) error {
				_, execErr := database.Connection(txCtx).Exec(txCtx, "insert into pinned_goexit values (1)")
				if execErr != nil {
					return execErr
				}
				runtime.Goexit()
				return nil
			})
			unexpectedResult <- beginErr
		}()
		<-transactionDone
		select {
		case beginErr := <-unexpectedResult:
			require.Fail(t, "transaction callback returned instead of exiting its goroutine", beginErr)
		default:
		}

		var followupPID int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(
			sessionCtx,
			"select pg_backend_pid()",
		).Scan(&followupPID))
		require.Equal(t, sessionPID, followupPID)

		var count int
		require.NoError(t, database.Connection(sessionCtx).QueryRow(
			sessionCtx,
			"select count(*) from pinned_goexit",
		).Scan(&count))
		require.Zero(t, count)

		require.NoError(t, tm.Begin(sessionCtx, func(txCtx context.Context) error {
			var transactionPID int
			require.NoError(t, database.Connection(txCtx).QueryRow(
				txCtx,
				"select pg_backend_pid()",
			).Scan(&transactionPID))
			require.Equal(t, sessionPID, transactionPID)
			_, execErr := database.Connection(txCtx).Exec(txCtx, "insert into pinned_goexit values (2)")
			return execErr
		}))

		require.NoError(t, database.Connection(sessionCtx).QueryRow(
			sessionCtx,
			"select count(*) from pinned_goexit",
		).Scan(&count))
		require.Equal(t, 1, count)
		return nil
	})
	require.NoError(t, err)
}
