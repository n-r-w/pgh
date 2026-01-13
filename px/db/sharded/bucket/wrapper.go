package bucket

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// bucketWrapper wrapper over IConnection that implements access to a specific bucket.
type bucketWrapper struct {
	db       conn.IConnection
	bucketID BucketID
}

func newBucketWrapper(db conn.IConnection, bucketID BucketID) *bucketWrapper {
	return &bucketWrapper{
		db:       db,
		bucketID: bucketID,
	}
}

// InTransaction returns true if a transaction has been started.
func (b *bucketWrapper) InTransaction() bool {
	return b.db.InTransaction()
}

// TransactionOptions returns transaction parameters. If transaction hasn't started, returns false.
func (b *bucketWrapper) TransactionOptions() txmgr.Options {
	return b.db.TransactionOptions()
}

// WithoutTransaction returns context without transaction.
func (b *bucketWrapper) WithoutTransaction(ctx context.Context) context.Context {
	return b.db.WithoutTransaction(ctx)
}

// Exec executes a query without returning data.
func (b *bucketWrapper) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	ctx = ToContext(ctx, b.bucketID)
	return b.db.Exec(ctx, PrepareBucketSQL(sql, b.bucketID), arguments...)
}

// Query executes a query and returns the result.
func (b *bucketWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx = ToContext(ctx, b.bucketID)
	return b.db.Query(ctx, PrepareBucketSQL(sql, b.bucketID), args...)
}

// QueryRow gets a connection and executes a query that should return no more than one row.
// Errors are deferred until pgx.Row.Scan method is called. If the query doesn't select a row,
// pgx.Row.Scan will return pgx.ErrNoRows.
// Otherwise, pgx.Row.Scan scans the first selected row and discards the rest.
func (b *bucketWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx = ToContext(ctx, b.bucketID)
	return b.db.QueryRow(ctx, PrepareBucketSQL(sql, b.bucketID), args...)
}

// SendBatch sends a set of queries for execution, combining all queries into one package.
// For correct operation, you need to create a batch instance using buckets.NewBatch function, not pgx.Batch.
// Then add queries to Batch using buckets.Batch.Queue function, after which call
// buckets.Batch.PgxBatch() function.
func (b *bucketWrapper) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults {
	ctx = ToContext(ctx, b.bucketID)
	return b.db.SendBatch(ctx, batch)
}

// LargeObjects supports working with large objects and is only available within a transaction
// (this is a postgresql limitation)
// Outside of a transaction, it will panic.
func (b *bucketWrapper) LargeObjects() pgx.LargeObjects {
	return b.db.LargeObjects()
}

// CopyFrom implements bulk data insertion into a table.
func (b *bucketWrapper) CopyFrom(ctx context.Context, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource,
) (n int64, err error) {
	ctx = ToContext(ctx, b.bucketID)
	return b.db.CopyFrom(ctx, tableName, columnNames, rowSrc)
}
