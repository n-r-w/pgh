package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

type telemetryHelperFunc func(ctx context.Context, command, details string, arguments []any, f func() error)

// ConnectionWrapper wrapper over iConnection with added telemetry and logs.
type ConnectionWrapper struct {
	con   conn.IConnection
	tFunc telemetryHelperFunc
}

// NewWrapper creates a new instance of Wrapper.
func newWrapper(con conn.IConnection, tFunc telemetryHelperFunc) *ConnectionWrapper {
	return &ConnectionWrapper{
		con:   con,
		tFunc: tFunc,
	}
}

// Exec executes a query without returning data.
func (p *ConnectionWrapper) Exec(ctx context.Context, sql string, arguments ...any) (tag pgconn.CommandTag, err error) {
	p.tFunc(ctx, "exec", sql, arguments, func() error {
		tag, err = p.con.Exec(ctx, sql, arguments...)
		return err
	})

	return tag, err
}

// Query executes a query and returns the result.
func (p *ConnectionWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	var (
		rows pgx.Rows
		err  error
	)

	p.tFunc(ctx, "query", sql, args, func() error {
		rows, err = p.con.Query(ctx, sql, args...) //nolint:sqlclosecheck // будет закрыто вызывающим
		return err
	})

	return rows, err
}

// QueryRow executes a query and returns the result.
func (p *ConnectionWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	var row pgx.Row

	p.tFunc(ctx, "query", sql, args, func() error {
		row = p.con.QueryRow(ctx, sql, args...)
		return nil
	})

	return row
}

// SendBatch sends a set of queries for execution, combining all queries into one batch.
func (p *ConnectionWrapper) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	var res pgx.BatchResults

	p.tFunc(ctx, "send batch", "batch", nil, func() error {
		res = p.con.SendBatch(ctx, b)
		return nil
	})

	return res
}

// CopyFrom implements bulk data insertion into a table.
func (p *ConnectionWrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource,
) (n int64, err error) {
	details := fmt.Sprintf("table: %s; columns: %s", tableName, strings.Join(columnNames, ", "))

	p.tFunc(ctx, "copy from", details, nil, func() error {
		n, err = p.con.CopyFrom(ctx, tableName, columnNames, rowSrc)
		return err
	})

	return n, err
}

// LargeObjects supports working with large objects and is only available in trace mode.
func (p *ConnectionWrapper) LargeObjects() pgx.LargeObjects {
	return p.con.LargeObjects()
}

// InTransaction returns true if a transaction has been started.
func (p *ConnectionWrapper) InTransaction() bool {
	return p.con.InTransaction()
}

// TransactionOptions returns transaction parameters. If the transaction has not been started, it returns false.
func (p *ConnectionWrapper) TransactionOptions() txmgr.Options {
	return p.con.TransactionOptions()
}

// WithoutTransaction returns a context without a transaction.
func (p *ConnectionWrapper) WithoutTransaction(ctx context.Context) context.Context {
	return p.con.WithoutTransaction(ctx)
}
