package pgh

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// BeginFunc starts a transaction, executes the callback function, and handles commit/rollback automatically.
// If the callback returns an error, the transaction is rolled back. Otherwise, it is committed.
// If commit/rollback fails, the error is wrapped with the original error if any.
func BeginFunc(ctx context.Context, conn ITransactionBeginner,
	f func(ctx context.Context, tx pgx.Tx) error,
) error {
	tx, err := conn.Begin(ctx)
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

	err = f(ctx, tx)
	if err != nil {
		rbErr := tx.Rollback(ctx)
		if rbErr != nil {
			return fmt.Errorf("rollback failed: %v: %w", rbErr, err) //nolint:errorlint // ok for 2 errors
		}
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
