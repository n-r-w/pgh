package px

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BeginTxFunc starts a transaction, executes the callback function, and handles commit/rollback automatically.
// If the callback returns an error, the transaction is rolled back. Otherwise, it is committed.
// If commit/rollback fails, the error is wrapped with the original error if any.
// Unlike pgx.BeginTxFunc, it passes the context to the function f.
func BeginTxFunc(ctx context.Context,
	conn ITransactionBeginner,
	options pgx.TxOptions,
	f func(ctx context.Context, tx pgx.Tx) error,
) (err error) {
	var tx pgx.Tx
	tx, err = conn.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// If panic occurs, rollback the transaction.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p) // Re-throw panic after rollback.
		}
	}()

	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			if err != nil {
				err = fmt.Errorf("%w (rollback error: %v)", err, rollbackErr) //nolint:errorlint // ok for 2 errors
			} else {
				err = rollbackErr
			}
		}
	}()

	err = f(ctx, tx)
	if err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// BeginFunc starts a transaction, executes the callback function, and handles commit/rollback automatically.
// If the callback returns an error, the transaction is rolled back. Otherwise, it is committed.
// If commit/rollback fails, the error is wrapped with the original error if any.
// Unlike pgx.BeginFunc, it passes the context to the function f.
func BeginFunc(ctx context.Context,
	conn ITransactionBeginner,
	f func(ctx context.Context, tx pgx.Tx) error,
) error {
	//nolint:exhaustruct // external type, zero values are acceptable defaults
	return BeginTxFunc(ctx, conn, pgx.TxOptions{}, f)
}
