package pgh

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	sq "github.com/n-r-w/squirrel"
)

// Helpers for working with Squirrel

// Builder creates a new instance of squirrel.StatementBuilderType for building queries
func Builder() sq.StatementBuilderType {
	return sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
}

// Exec executes a modification query. Querier can be either pgx.Tx or pg_types.Pool
func Exec(ctx context.Context, querier IQuerier, sqlizer sq.Sqlizer) (pgconn.CommandTag, error) {
	var (
		sql  string
		args []any
		err  error
	)

	if sql, args, err = sqToSQL(ctx, sqlizer); err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("pgh.Exec to sql: %w", err)
	}

	return ExecPlain(ctx, querier, sql, args)
}

// Select executes a query. Querier can be either pgx.Tx or pg_types.Pool
func Select[T any](ctx context.Context, querier IQuerier, sqlizer sq.Sqlizer, dst *[]T) error {
	var (
		sql  string
		args []any
		err  error
	)

	if sql, args, err = sqToSQL(ctx, sqlizer); err != nil {
		return fmt.Errorf("pgh.Select to sql: %w", err)
	}

	return SelectPlain(ctx, querier, sql, dst, args)
}

// SelectFunc executes a query and passes each row to function f. Querier can be either pgx.Tx or pg_types.Pool.
func SelectFunc(ctx context.Context, querier IQuerier, sqlizer sq.Sqlizer, f func(row pgx.Row) error) error {
	var (
		sql  string
		args []any
		err  error
	)

	if sql, args, err = sqToSQL(ctx, sqlizer); err != nil {
		return fmt.Errorf("pgh.SelectFunc to sql: %w", err)
	}

	return SelectFuncPlain(ctx, querier, sql, args, f)
}

// SelectOne executes a query. Querier can be either pgx.Tx or pg_types.Pool.
// dst must contain a variable, not a slice.
func SelectOne[T any](ctx context.Context, querier IQuerier, sqlizer sq.Sqlizer, dst *T) error {
	var (
		sql  string
		args []any
		err  error
	)

	if sql, args, err = sqToSQL(ctx, sqlizer); err != nil {
		return fmt.Errorf("pgh.SelectOne to sql: %w", err)
	}

	return SelectOnePlain(ctx, querier, sql, dst, args)
}

// ExecSplit splits queries into groups of splitSize and executes them separately within a transaction.
// tx can be either pgx.Tx or pg_types.Pool
func ExecSplit(ctx context.Context, tx IBatcher, queries []sq.Sqlizer, splitSize int) (rowsAffected int64, err error) {
	var (
		queriesSQL = make([]string, 0, len(queries))
		args       = make([]Args, 0, len(queries))
	)

	for _, query := range queries {
		var (
			sql string
			a   []any
		)

		if sql, a, err = sqToSQL(ctx, query); err != nil {
			return 0, err
		}
		queriesSQL = append(queriesSQL, sql)
		args = append(args, a)
	}

	return ExecSplitPlain(ctx, tx, queriesSQL, args, splitSize)
}

// InsertSplit splits queries into groups of splitSize and executes them separately within a transaction.
// values - rows with the same number of values in each. tx can be either pgx.Tx or pg_types.Pool.
// Use this when COPY cannot be used, for example, when ON CONFLICT is required.
func InsertSplit(
	ctx context.Context,
	tx IBatcher,
	base sq.InsertBuilder,
	values []Args,
	splitSize int,
) (rowsAffected int64, err error) {
	var (
		sql   string
		args  []any
		l     = len(values)
		idxTo int
	)

	batch := pgx.Batch{}
	for idx := 0; idx < l; idx += splitSize {
		if idxTo = idx + splitSize; idxTo > l {
			idxTo = l
		}

		query := base
		for _, vals := range values[idx:idxTo] {
			query = query.Values(vals...)
		}

		if sql, args, err = sqToSQL(ctx, query); err != nil {
			return 0, err
		}

		batch.Queue(sql, args...)
	}

	return SendBatch(ctx, tx, &batch)
}

// InsertSplitQuery splits queries into groups of splitSize and executes them separately within a transaction.
// values - rows with the same number of values in each. tx can be either pgx.Tx or pg_types.Pool.
// Use this when COPY cannot be used, for example, when ON CONFLICT is required.
// Unlike InsertSplit, it allows using the RETURNING clause to get data into dst.
func InsertSplitQuery[T any](
	ctx context.Context,
	tx IBatcher,
	base sq.InsertBuilder,
	values []Args,
	splitSize int,
	dst *[]T,
) (err error) {
	var (
		sql   string
		args  []any
		l     = len(values)
		idxTo int
	)

	batch := pgx.Batch{}
	for idx := 0; idx < l; idx += splitSize {
		if idxTo = idx + splitSize; idxTo > l {
			idxTo = l
		}

		query := base
		for _, vals := range values[idx:idxTo] {
			query = query.Values(vals...)
		}

		if sql, args, err = sqToSQL(ctx, query); err != nil {
			return err
		}

		batch.Queue(sql, args...)
	}

	return SendBatchQuery(ctx, tx, &batch, dst)
}

// ExecBatch executes a batch of queries with error checking. tx can be either pgx.Tx or pg_types.Pool.
func ExecBatch(ctx context.Context, queries []sq.Sqlizer, tx IBatcher) (rowsAffected int64, err error) {
	batch := pgx.Batch{}

	for _, query := range queries {
		var (
			sql string
			a   []any
		)

		if sql, a, err = sqToSQL(ctx, query); err != nil {
			return 0, err
		}

		batch.Queue(sql, a...)
	}

	return SendBatch(ctx, tx, &batch)
}

// SelectBatch executes a batch of queries with error checking. tx can be either pgx.Tx or pg_types.Pool.
func SelectBatch[T any](ctx context.Context, queries []sq.Sqlizer, tx IBatcher, dst *[]T) error {
	batch := pgx.Batch{}

	for _, query := range queries {
		var (
			sql string
			a   []any
			err error
		)

		if sql, a, err = sqToSQL(ctx, query); err != nil {
			return err
		}

		batch.Queue(sql, a...)
	}

	return SendBatchQuery(ctx, tx, &batch, dst)
}

// sqToSQL converts squirrel query to SQL
func sqToSQL(_ context.Context, sqlizer sq.Sqlizer) (sql string, args []any, err error) {
	if sql, args, err = sqlizer.ToSql(); err != nil {
		return sql, args, err
	}

	return sql, args, nil
}
