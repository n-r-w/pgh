package txmgr

//go:generate mockgen -source interface.go -destination interface_mock.go -package txmgr

import "context"

// ITransactionInformer interface for transaction information. Implemented in pgdb package.
type ITransactionInformer interface {
	// InTransaction returns true if transaction is started.
	InTransaction(ctx context.Context) bool
	// TransactionOptions returns transaction parameters.
	TransactionOptions(ctx context.Context) Options
}

// ITransactionBeginner interface for starting transactions. Implemented in pgdb package.
type ITransactionBeginner interface {
	Begin(ctx context.Context, f func(ctxTr context.Context) error, opts Options) error

	// WithoutTransaction returns context without transaction.
	WithoutTransaction(ctx context.Context) context.Context
}

// ITransactionManager interface for managing database transactions.
// Located here at the implementation point for convenient use in other packages.
type ITransactionManager interface {
	// Begin starts a transaction. If transaction is already started - increment nested level.
	Begin(ctx context.Context, f func(ctxTr context.Context) error, opts ...Option) error

	// WithoutTransaction returns context without transaction.
	WithoutTransaction(ctx context.Context) context.Context
}
