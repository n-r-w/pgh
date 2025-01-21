package pq

import (
	"context"
	"database/sql"
)

//go:generate mockgen -source interface.go -destination interface_mock.go -package pq
//go:generate mockgen -package pq -destination sql_mock.go database/sql/driver Rows,Result

// IQuerier - a subset of sql.DB, sql.Conn and sql.Tx for queries.
type IQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// ITransactionBeginner - a subset of sql.DB, sql.Conn for starting transactions.
type ITransactionBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
