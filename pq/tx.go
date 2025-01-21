package pq

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// BeginTxFunc - starts a transaction, executes the callback function, and handles commit/rollback automatically.
// If the callback returns an error, the transaction is rolled back. Otherwise, it is committed.
// If commit/rollback fails, the error is wrapped with the original error if any.
func BeginTxFunc(ctx context.Context, conn ITransactionBeginner, opts *sql.TxOptions,
	f func(ctx context.Context, tx *sql.Tx) error,
) (err error) {
	var tx *sql.Tx
	tx, err = conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// If panic occurs, rollback the transaction.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // Re-throw panic after rollback.
		}
	}()

	defer func() {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			if err != nil {
				err = fmt.Errorf("%w (rollback error: %v)", err, rollbackErr) //nolint:errorlint // ok for 2 errors
			} else {
				err = rollbackErr
			}
		}
	}()

	err = f(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// BeginFunc - starts a transaction, executes the callback function, and handles commit/rollback automatically.
// If the callback returns an error, the transaction is rolled back. Otherwise, it is committed.
// If commit/rollback fails, the error is wrapped with the original error if any.
func BeginFunc(ctx context.Context, conn ITransactionBeginner,
	f func(ctx context.Context, tx *sql.Tx) error,
) error {
	return BeginTxFunc(ctx, conn, nil, f)
}
