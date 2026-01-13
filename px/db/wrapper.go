package db

import (
	"context"
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
	logQueries bool
	txOpts     txmgr.Options
	db         *PxDB
	tx         pgx.Tx
}

// newDatabaseWrapperNoTran creates databaseWrapper for working without transaction.
func newDatabaseWrapperNoTran(db *PxDB, logQueries bool) *Wrapper {
	return &Wrapper{
		db:         db,
		tx:         nil,
		txOpts:     txmgr.Options{Level: 0, Mode: 0, Lock: false},
		logQueries: logQueries,
	}
}

// newDatabaseWrapperWithTran creates databaseWrapper for working with transaction.
func newDatabaseWrapperWithTran(db *PxDB, tx pgx.Tx, txOpts txmgr.Options, logQueries bool) *Wrapper {
	return &Wrapper{
		db:         db,
		tx:         tx,
		txOpts:     txOpts,
		logQueries: logQueries,
	}
}

// InTransaction returns true if transaction is started.
func (i *Wrapper) InTransaction() bool {
	return i.tx != nil
}

// TransactionOptions returns transaction parameters. If transaction is not started, returns false.
func (i *Wrapper) TransactionOptions() txmgr.Options {
	if i.tx == nil {
		return txmgr.Options{Level: 0, Mode: 0, Lock: false}
	}
	return i.txOpts
}

// WithoutTransaction returns context without transaction.
func (i *Wrapper) WithoutTransaction(ctx context.Context) context.Context {
	return WithoutTransaction(ctx)
}

// CopyFrom implements bulk data insertion into a table.
func (i *Wrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource,
) (n int64, err error) {
	i.logQueryHelper(ctx, fmt.Sprintf("COPY %s (%s)", tableName, strings.Join(columnNames, ", ")), "", nil, func() error {
		if i.tx != nil {
			n, err = i.tx.CopyFrom(ctx, tableName, columnNames, rowSrc)
		} else {
			n, err = i.db.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
		}
		return err
	})

	return n, err
}

// Exec executes a query without returning data.
func (i *Wrapper) Exec(ctx context.Context, sql string, args ...any) (tag pgconn.CommandTag, err error) {
	i.logQueryHelper(ctx, "Exec", sql, args, func() error {
		if i.tx != nil {
			tag, err = i.tx.Exec(ctx, sql, args...)
		} else {
			tag, err = i.db.pool.Exec(ctx, sql, args...)
		}
		return err
	})

	return tag, err
}

// Query executes a query and returns the result.
func (i *Wrapper) Query(ctx context.Context, sql string, args ...any) (rows pgx.Rows, err error) {
	i.logQueryHelper(ctx, "Query", sql, args, func() error {
		if i.tx != nil {
			rows, err = i.tx.Query(ctx, sql, args...) //nolint:sqlclosecheck // will be closed by caller
		} else {
			rows, err = i.db.pool.Query(ctx, sql, args...) //nolint:sqlclosecheck // will be closed by caller
		}
		return err
	})

	return rows, err
}

// QueryRow gets a connection and executes a query that should return no more than one row.
// Errors are deferred until the pgx.Row.Scan method is called. If the query doesn't select a row,
// pgx.Row.Scan will return pgx.ErrNoRows.
// Otherwise, pgx.Row.Scan scans the first selected row and discards the rest.
// The obtained connection is returned to the pool when the pgx.Row.Scan method is called.
func (i *Wrapper) QueryRow(ctx context.Context, sql string, args ...any) (row pgx.Row) {
	i.logQueryHelper(ctx, "QueryRow", sql, args, func() error {
		if i.tx != nil {
			row = i.tx.QueryRow(ctx, sql, args...)
		} else {
			row = i.db.pool.QueryRow(ctx, sql, args...)
		}
		return nil
	})

	return row
}

// SendBatch sends a set of queries for execution, combining all queries into one package.
func (i *Wrapper) SendBatch(ctx context.Context, b *pgx.Batch) (res pgx.BatchResults) {
	const batchSizeLogLimit = 10
	var queries strings.Builder

	if i.logQueries {
		queries.Grow(batchSizeLogLimit)
		for i, q := range b.QueuedQueries {
			if i > batchSizeLogLimit {
				_, _ = queries.WriteString("...")
				break
			}
			// [SELECT * FROM users WHERE id IN ($1,$2); ARGS: 2,3]
			_, _ = queries.WriteString("[")
			_, _ = queries.WriteString(q.SQL)
			_, _ = queries.WriteString("; ARGS: ")
			for j, arg := range q.Arguments {
				if j > 0 {
					_, _ = queries.WriteString(",")
				}
				_, _ = queries.WriteString(fmt.Sprintf("%v", arg))
			}
			_, _ = queries.WriteString("]")
		}
	}

	i.logQueryHelper(ctx, queries.String(), "", nil, func() error {
		if i.tx != nil {
			res = i.tx.SendBatch(ctx, b)
		} else {
			res = i.db.pool.SendBatch(ctx, b)
		}
		return nil
	})

	return res
}

// LargeObjects supports working with large objects and is only available within a transaction (PostgreSQL limitation).
// Outside of a transaction, it will panic.
func (i *Wrapper) LargeObjects() pgx.LargeObjects {
	if i.tx != nil {
		return i.tx.LargeObjects()
	}

	panic("LargeObjects() is not supported without transaction")
}

// logQueryHelper performs query logging and calls function f.
func (i *Wrapper) logQueryHelper(ctx context.Context, command, query string, args []any, f func() error) {
	if !i.logQueries {
		_ = f() // we're not interested in the result since it should be handled inside f
		return
	}

	start := time.Now()

	err := f()

	attrs := []any{
		"database", i.db.name,
		"command", command,
		"latency", time.Since(start),
		"args", args,
	}

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
