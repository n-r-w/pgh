package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/txmgr"
	"go.opentelemetry.io/otel/trace"
)

// Wrapper is a wrapper over pgx.
type Wrapper struct {
	logQueries  bool
	txOpts      txmgr.Options
	db          *PxDB
	tx          pgx.Tx
	transaction *transaction
	session     *pinnedSession
}

// queryExecutor contains the operations shared by a pool, connection, and transaction.
type queryExecutor interface {
	CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

// newDatabaseWrapperNoTran creates a database wrapper without a transaction.
func newDatabaseWrapperNoTran(db *PxDB, session *pinnedSession, logQueries bool) *Wrapper {
	return &Wrapper{
		db: db, session: session, tx: nil, transaction: nil,
		txOpts: txmgr.Options{Level: 0, Mode: 0, Lock: false}, logQueries: logQueries,
	}
}

// newDatabaseWrapperWithTran creates a database wrapper with a transaction.
func newDatabaseWrapperWithTran(db *PxDB, session *pinnedSession, tx *transaction, logQueries bool) *Wrapper {
	return &Wrapper{
		db: db, session: session, tx: tx.tx, transaction: tx,
		txOpts: tx.opts, logQueries: logQueries,
	}
}

// InTransaction returns true if the wrapper has an active transaction.
func (i *Wrapper) InTransaction() bool {
	if i.transaction == nil {
		return false
	}
	if i.session == nil {
		return true
	}
	return !i.transaction.isFinished() && !i.session.conn.Conn().IsClosed()
}

// TransactionOptions returns transaction parameters or defaults when no transaction is active.
func (i *Wrapper) TransactionOptions() txmgr.Options {
	if !i.InTransaction() {
		return txmgr.Options{Level: 0, Mode: 0, Lock: false}
	}
	return i.txOpts
}

// WithoutTransaction returns a context without transaction state.
func (*Wrapper) WithoutTransaction(ctx context.Context) context.Context {
	return WithoutTransaction(ctx)
}

// operationError validates wrapper lifecycle state.
func (i *Wrapper) operationError() error {
	if i.session != nil {
		if i.session.db != i.db {
			return errors.New("pinned session belongs to another database")
		}
		return i.session.operationError(i.transaction)
	}
	return nil
}

// executor returns the operation target after lifecycle validation.
func (i *Wrapper) executor() (queryExecutor, error) {
	if err := i.operationError(); err != nil {
		return nil, err
	}
	if i.tx != nil {
		return i.tx, nil
	}
	if i.session != nil {
		return i.session.conn, nil
	}
	return i.db.pool, nil
}

// CopyFrom implements bulk data insertion into a table.
func (i *Wrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource,
) (n int64, err error) {
	i.logQueryHelper(ctx, fmt.Sprintf("COPY %s (%s)", tableName, strings.Join(columnNames, ", ")), "", nil, func() error {
		executor, executorErr := i.executor()
		if executorErr != nil {
			err = executorErr
			return err
		}
		n, err = executor.CopyFrom(ctx, tableName, columnNames, rowSrc)
		return err
	})
	return n, err
}

// Exec executes a query without returning data.
func (i *Wrapper) Exec(ctx context.Context, sql string, args ...any) (tag pgconn.CommandTag, err error) {
	i.logQueryHelper(ctx, "Exec", sql, args, func() error {
		executor, executorErr := i.executor()
		if executorErr != nil {
			err = executorErr
			return err
		}
		tag, err = executor.Exec(ctx, sql, args...)
		return err
	})
	return tag, err
}

// Query executes a query and returns the result.
func (i *Wrapper) Query(ctx context.Context, sql string, args ...any) (rows pgx.Rows, err error) {
	i.logQueryHelper(ctx, "Query", sql, args, func() error {
		executor, executorErr := i.executor()
		if executorErr != nil {
			err = executorErr
			return err
		}
		rows, err = executor.Query(ctx, sql, args...) //nolint:sqlclosecheck // caller owns rows
		return err
	})
	return rows, err
}

// QueryRow executes a query whose error is deferred until Scan.
func (i *Wrapper) QueryRow(ctx context.Context, sql string, args ...any) (row pgx.Row) {
	i.logQueryHelper(ctx, "QueryRow", sql, args, func() error {
		executor, err := i.executor()
		if err != nil {
			row = errorRow{err: err}
			return err
		}
		row = executor.QueryRow(ctx, sql, args...)
		return nil
	})
	return row
}

// SendBatch sends a set of queries for execution.
func (i *Wrapper) SendBatch(ctx context.Context, b *pgx.Batch) (res pgx.BatchResults) {
	const batchSizeLogLimit = 10
	var queries strings.Builder
	if i.logQueries {
		queries.Grow(batchSizeLogLimit)
		for index, query := range b.QueuedQueries {
			if index > batchSizeLogLimit {
				_, _ = queries.WriteString("...")
				break
			}
			_, _ = queries.WriteString("[")
			_, _ = queries.WriteString(query.SQL)
			_, _ = queries.WriteString("; ARGS: ")
			for argumentIndex, argument := range query.Arguments {
				if argumentIndex > 0 {
					_, _ = queries.WriteString(",")
				}
				_, _ = fmt.Fprintf(&queries, "%v", argument)
			}
			_, _ = queries.WriteString("]")
		}
	}

	i.logQueryHelper(ctx, queries.String(), "", nil, func() error {
		executor, err := i.executor()
		if err != nil {
			res = errorBatchResults{err: err}
			return err
		}
		res = executor.SendBatch(ctx, b)
		return nil
	})
	return res
}

// LargeObjects supports large objects inside a transaction.
func (i *Wrapper) LargeObjects() pgx.LargeObjects {
	if i.tx != nil {
		return i.tx.LargeObjects()
	}
	panic("LargeObjects() is not supported without transaction")
}

// errorRow defers a lifecycle error until Scan.
type errorRow struct {
	err error
}

// Scan returns the stored lifecycle error.
func (r errorRow) Scan(...any) error {
	return r.err
}

// errorBatchResults returns one lifecycle error for every batch operation.
type errorBatchResults struct {
	err error
}

// Exec returns the stored lifecycle error.
func (r errorBatchResults) Exec() (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, r.err
}

// Query returns the stored lifecycle error.
func (r errorBatchResults) Query() (pgx.Rows, error) {
	return nil, r.err
}

// QueryRow returns a row that reports the stored lifecycle error.
func (r errorBatchResults) QueryRow() pgx.Row {
	return errorRow(r)
}

// Close returns the stored lifecycle error.
func (r errorBatchResults) Close() error {
	return r.err
}

// logQueryHelper performs query logging and calls f.
func (i *Wrapper) logQueryHelper(ctx context.Context, command, query string, args []any, f func() error) {
	if !i.logQueries {
		_ = f()
		return
	}

	start := time.Now()
	err := f()
	attrs := []any{"database", i.db.name, "command", command, "latency", time.Since(start), "args", args}
	if query != "" {
		attrs = append(attrs, "query", query)
	}
	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if spanContext.TraceID().IsValid() {
		attrs = append(attrs, "trace_id", spanContext.TraceID().String())
	}
	if err != nil {
		attrs = append(attrs, "error", err)
		i.db.logger.Error(ctx, "dbquery", attrs...)
	} else {
		i.db.logger.Debug(ctx, "dbquery", attrs...)
	}
}
