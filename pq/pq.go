// Package pq provides database/sql compatibility layer for PostgreSQL operations.
package pq

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/sqlscan"
	"github.com/n-r-w/pgh/v2"
)

// ExecPlain - executes a modification query. Querier can be either sql.Tx or sql.DB.
func ExecPlain(ctx context.Context, db IQuerier, query string, args pgh.Args) (sql.Result, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sql exec: %w [%s]", err, pgh.TruncSQL(query))
	}
	return result, nil
}

// SelectPlain - executes a query. Querier can be either sql.Tx or sql.DB.
func SelectPlain[T any](ctx context.Context, db IQuerier, query string, dst *[]T, args pgh.Args) error {
	if err := sqlscan.Select(ctx, db, dst, query, args...); err != nil {
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(query))
	}
	return nil
}

// SelectFuncPlain - executes a query and passes each row to function f.
// Querier can be either sql.Tx or sql.DB.
func SelectFuncPlain(
	ctx context.Context,
	db IQuerier,
	query string,
	args pgh.Args,
	f func(*sql.Rows) error,
) (err error) {
	var rows *sql.Rows
	rows, err = db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(query))
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		if err = f(rows); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql rows: %w [%s]", err, pgh.TruncSQL(query))
	}

	return nil
}

// SelectOnePlain - executes a query. Querier can be either sql.Tx or sql.DB.
// dst must contain a variable, not a slice.
func SelectOnePlain[T any](ctx context.Context, db IQuerier, query string, dst *T, args pgh.Args) error {
	if err := sqlscan.Get(ctx, db, dst, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(query))
	}
	return nil
}

// InsertValuesPlain - executes a query to insert a group of values.
// query should be in the form "INSERT INTO table (col1, col2)" without VALUES.
// VALUES is added automatically.
func InsertValuesPlain(ctx context.Context, db IQuerier, query string, values []pgh.Args) error {
	if len(values) == 0 {
		return nil
	}

	var (
		args        = make(pgh.Args, 0, len(values[0])*len(values))
		sqlBuilder  strings.Builder
		columnCount = len(values[0])
	)
	sqlBuilder.WriteString(query)
	sqlBuilder.WriteString(" VALUES ")
	for _, v := range values {
		if len(v) != columnCount {
			return fmt.Errorf("pq.InsertValues: all values must have the same length. sql: %s", pgh.TruncSQL(query))
		}
		args = append(args, v...)
	}

	for i := range values {
		if i != 0 {
			sqlBuilder.WriteString(",")
		}
		sqlBuilder.WriteString("(")
		for j := range columnCount {
			if j != 0 {
				sqlBuilder.WriteString(",")
			}
			sqlBuilder.WriteString(fmt.Sprintf("$%d", i*columnCount+j+1))
		}
		sqlBuilder.WriteString(")")
	}

	targetSQL := sqlBuilder.String()

	if _, err := db.ExecContext(ctx, targetSQL, args...); err != nil {
		return fmt.Errorf("pq.InsertValues: %w [%s]", err, pgh.TruncSQL(targetSQL))
	}

	return nil
}
