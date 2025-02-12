package px

//go:generate mockgen -source interface.go -destination interface_mock.go -package px
//go:generate mockgen -package px -destination pgx_mock.go github.com/jackc/pgx/v5 BatchResults,Row,Rows,Tx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// IQuerier is a subset of pgxpool.Pool, pgx.Conn and pgx.Tx interfaces for queries
type IQuerier interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
}

// IBatcher is a subset of pgxpool.Pool, pgx.Conn and pgx.Tx interfaces for batches
type IBatcher interface {
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

// ITransactionBeginner is a subset of pgxpool.Pool, pgx.Conn for starting transactions.
type ITransactionBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
