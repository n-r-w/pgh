package pq

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/n-r-w/squirrel"
)

// Exec - executes a modification query. Querier can be either sql.Tx or sql.DB.
func Exec(ctx context.Context, db IQuerier, sqlizer sq.Sqlizer) (sql.Result, error) {
	q, args, err := sqToSQL(ctx, sqlizer)
	if err != nil {
		return nil, fmt.Errorf("pq.Exec to sql: %w", err)
	}
	return ExecPlain(ctx, db, q, args)
}

// Select - executes a query. Querier can be either sql.Tx or sql.DB.
func Select[T any](ctx context.Context, db IQuerier, sqlizer sq.Sqlizer, dst *[]T) error {
	q, args, err := sqToSQL(ctx, sqlizer)
	if err != nil {
		return fmt.Errorf("pq.Select to sql: %w", err)
	}
	return SelectPlain(ctx, db, q, dst, args)
}

// SelectFunc - executes a query and passes each row to function f.
// Querier can be either sql.Tx or sql.DB.
func SelectFunc(ctx context.Context, db IQuerier, sqlizer sq.Sqlizer, f func(*sql.Rows) error) error {
	q, args, err := sqToSQL(ctx, sqlizer)
	if err != nil {
		return fmt.Errorf("pq.SelectFunc to sql: %w", err)
	}
	return SelectFuncPlain(ctx, db, q, args, f)
}

// SelectOne - executes a query. Querier can be either sql.Tx or sql.DB.
// dst must contain a variable, not a slice.
func SelectOne[T any](ctx context.Context, db IQuerier, sqlizer sq.Sqlizer, dst *T) error {
	q, args, err := sqToSQL(ctx, sqlizer)
	if err != nil {
		return fmt.Errorf("pq.SelectOne to sql: %w", err)
	}
	return SelectOnePlain(ctx, db, q, dst, args)
}

// sqToSQL - converts squirrel query to SQL.
func sqToSQL(_ context.Context, sqlizer sq.Sqlizer) (q string, args []any, err error) {
	if q, args, err = sqlizer.ToSql(); err != nil {
		return q, args, err
	}
	return q, args, nil
}
