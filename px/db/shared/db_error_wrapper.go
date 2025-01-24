package shared

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/txmgr"
)

// ErrorWrapper implements the IConnection interface with error return.
type ErrorWrapper struct {
	err error
}

// NewDatabaseErrorWrapper creates a databaseErrorWrapper.
func NewDatabaseErrorWrapper(err error) *ErrorWrapper {
	return &ErrorWrapper{
		err: err,
	}
}

// InTransaction always returns false.
func (i *ErrorWrapper) InTransaction() bool {
	return false
}

// TransactionOptions always returns an empty object.
func (i *ErrorWrapper) TransactionOptions() txmgr.Options {
	return txmgr.Options{}
}

// WithoutTransaction returns a context without transaction.
func (i *ErrorWrapper) WithoutTransaction(ctx context.Context) context.Context {
	return ctx
}

// CopyFrom returns an error.
func (i *ErrorWrapper) CopyFrom(_ context.Context, _ pgx.Identifier,
	_ []string, _ pgx.CopyFromSource,
) (n int64, err error) {
	return 0, i.err
}

// Exec returns an error.
func (i *ErrorWrapper) Exec(_ context.Context, _ string, _ ...any) (tag pgconn.CommandTag, err error) {
	return pgconn.CommandTag{}, i.err
}

// Query returns an error.
func (i *ErrorWrapper) Query(_ context.Context, _ string, _ ...any) (rows pgx.Rows, err error) {
	return nil, i.err
}

// QueryRow returns an error.
func (i *ErrorWrapper) QueryRow(_ context.Context, _ string, _ ...any) (row pgx.Row) {
	return NewErrRow(i.err)
}

// SendBatch returns an error.
func (i *ErrorWrapper) SendBatch(_ context.Context, _ *pgx.Batch) (res pgx.BatchResults) {
	return newErrBatchResults(i.err)
}

// LargeObjects panics because it needs to return pgx.LargeObjects which contains private fields and has no constructor.
func (i *ErrorWrapper) LargeObjects() pgx.LargeObjects {
	// we can't do anything here because we need to return pgx.LargeObjects,
	// which contains private fields and has no constructor
	// we must either panic here or panic will occur later when using pgx.LargeObjects
	panic(fmt.Sprintf("failed to get large objects: %v", i.err))
}

// ErrRow wrapper around pgx.Row that always returns an error.
type ErrRow struct {
	err error
}

// NewErrRow creates an ErrRow.
func NewErrRow(err error) *ErrRow {
	return &ErrRow{
		err: err,
	}
}

// Scan returns an error.
func (e ErrRow) Scan(_ ...any) error {
	return e.err
}

type errBatchResults struct {
	err error
}

func newErrBatchResults(err error) *errBatchResults {
	return &errBatchResults{
		err: err,
	}
}

func (e errBatchResults) Exec() (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, e.err
}

func (e errBatchResults) Query() (pgx.Rows, error) {
	return nil, e.err
}

func (e errBatchResults) QueryRow() pgx.Row {
	return NewErrRow(e.err)
}

func (e errBatchResults) Close() error {
	return e.err
}
