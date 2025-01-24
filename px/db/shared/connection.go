package shared

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

//go:generate mockgen -source connection.go -destination connection_mock.go -package shared

// IConnection includes methods from pgxpool.Pool, pgx.Conn and pgx.Tx + methods for checking transaction state.
type IConnection interface {
	// Exec executes a query without returning data
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	// Query executes a query and returns the result
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	// QueryRow gets a connection and executes a query that should return no more than one row.
	// Errors are deferred until the pgx.Row.Scan method is called.
	// If the query selects no rows, pgx.Row.Scan will return pgx.ErrNoRows.
	// Otherwise, pgx.Row.Scan scans the first selected row and discards the rest.
	// The obtained connection is returned to the pool when the pgx.Row.Scan method is called.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// SendBatch sends a batch of queries for execution, combining all queries into a single batch
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	// CopyFrom implements bulk data insertion into a table
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)

	// LargeObjects supports working with large objects and is only available
	// in a transaction (this is a PostgreSQL limitation).
	// Will panic if used outside a transaction.
	LargeObjects() pgx.LargeObjects

	// InTransaction returns true if a transaction has started.
	InTransaction() bool
	// TransactionOptions returns transaction options. If no transaction has started, returns false.
	TransactionOptions() txmgr.Options
	// WithoutTransaction returns a context without a transaction.
	WithoutTransaction(ctx context.Context) context.Context
}

// IStartStopConnector - interface for a service that creates IConnection and can be started and stopped.
type IStartStopConnector interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Connection(ctx context.Context, opt ...ConnectionOption) IConnection
}

// ConnectionOption options for Connection.
type ConnectionOption func(*ConnectionOptionData)

// WithLogQueries enables query logging at the specific GetDB call level.
func WithLogQueries() ConnectionOption {
	return func(o *ConnectionOptionData) {
		o.LogQueries = true
	}
}

// ConnectionOptionData option data for Connection.
type ConnectionOptionData struct {
	LogQueries bool
}
