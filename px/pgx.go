package px

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2"
)

// ExecPlain executes a modification query. Querier can be either pgx.Tx or pg_types.Pool.
func ExecPlain(ctx context.Context, querier IQuerier, sql string, args pgh.Args) (pgconn.CommandTag, error) {
	var (
		tag pgconn.CommandTag
		err error
	)

	if tag, err = querier.Exec(ctx, sql, args...); err != nil {
		return tag, fmt.Errorf("sql exec: %w [%s]", err, pgh.TruncSQL(sql))
	}

	return tag, nil
}

// SelectPlain executes a query. Querier can be either pgx.Tx or pg_types.Pool.
func SelectPlain[T any](ctx context.Context, querier IQuerier, sql string, dst *[]T, args pgh.Args) error {
	if err := pgxscan.Select(ctx, querier, dst, sql, args...); err != nil {
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(sql))
	}

	return nil
}

// SelectFuncPlain executes a query and passes each row to function f.
// Querier can be either pgx.Tx or pg_types.Pool.
func SelectFuncPlain(ctx context.Context, querier IQuerier, sql string, args pgh.Args,
	f func(pgx.Row) error,
) error {
	rows, err := querier.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(sql))
	}
	defer rows.Close()

	for rows.Next() {
		if err := f(rows); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(sql))
	}

	return nil
}

// SelectOnePlain executes a query. Querier can be either pgx.Tx or pg_types.Pool.
// dst must contain a variable, not a slice.
func SelectOnePlain[T any](ctx context.Context, querier IQuerier, sql string, dst *T, args pgh.Args) error {
	if err := pgxscan.Get(ctx, querier, dst, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// we don't need the original error, it contains extra service information that will come in the response
			return pgx.ErrNoRows
		}

		return fmt.Errorf("sql select: %w [%s]", err, pgh.TruncSQL(sql))
	}

	return nil
}

// ExecSplitPlain splits queries into groups of splitSize and executes them separately within a transaction.
// tx can be either pgx.Tx or pg_types.Pool.
func ExecSplitPlain(
	ctx context.Context,
	tx IBatcher,
	queries []string,
	args []pgh.Args,
	splitSize int,
) (rowsAffected int64, err error) {
	if splitSize <= 0 {
		return 0, errors.New("ExecSplitPlain: splitSize must be greater than zero")
	}

	var (
		l     = len(queries)
		idxTo int
		ra    int64
	)

	for idx := 0; idx < l; idx += splitSize {
		if idxTo = idx + splitSize; idxTo > l {
			idxTo = l
		}
		if ra, err = execSplitPlainHelper(ctx, tx, queries[idx:idxTo], args[idx:idxTo]); err != nil {
			return 0, err
		}
		rowsAffected += ra
	}
	return rowsAffected, nil
}

func execSplitPlainHelper(
	ctx context.Context,
	tx IBatcher,
	queries []string,
	args []pgh.Args,
) (rowsAffected int64, err error) {
	//nolint:exhaustruct // external type, QueuedQueries is managed by Queue method
	batch := pgx.Batch{}

	for idx, query := range queries {
		batch.Queue(query, args[idx]...)
	}

	return SendBatch(ctx, tx, &batch)
}

// InsertSplitPlain splits queries into groups of splitSize and executes them separately within a transaction.
// values - rows with the same number of values in each. tx can be either pgx.Tx or pg_types.Pool.
// Use this when COPY cannot be used, for example, when ON CONFLICT is required.
func InsertSplitPlain(
	ctx context.Context,
	tx IBatcher,
	sql string,
	values []pgh.Args,
	splitSize int,
) (rowsAffected int64, err error) {
	if splitSize <= 0 {
		return 0, errors.New("ExecSplitPlain: splitSize must be greater than zero")
	}

	var (
		l     = len(values)
		idxTo int
	)

	//nolint:exhaustruct // external type, QueuedQueries is managed by Queue method
	batch := pgx.Batch{}
	for idx := 0; idx < l; idx += splitSize {
		if idxTo = idx + splitSize; idxTo > l {
			idxTo = l
		}

		batch.Queue(sql, values[idx:idxTo])
	}

	return SendBatch(ctx, tx, &batch)
}

// SendBatch executes a batch of queries with error checking. tx can be either pgx.Tx or pg_types.Pool.
func SendBatch(ctx context.Context, tx IBatcher, batch *pgx.Batch) (rowsAffected int64, err error) {
	if batch.Len() == 0 {
		return 0, nil
	}

	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := range batch.Len() {
		tag, err := br.Exec()
		if err != nil {
			return 0, fmt.Errorf("pgx.SendBatch exec at index %d: %w", i, err)
		}
		rowsAffected += tag.RowsAffected()
	}
	return rowsAffected, nil
}

// SendBatchQuery executes a batch of queries with error checking.
// tx can be either pgx.Tx or pg_types.Pool.
// Returns query results as a single slice.
func SendBatchQuery[T any](ctx context.Context, tx IBatcher, batch *pgx.Batch, dst *[]T) error {
	if batch.Len() == 0 {
		return nil
	}

	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	for i := range batch.Len() {
		rows, err := br.Query()
		if err != nil {
			return fmt.Errorf("pgx.SendBatchQuery query at index %d: %w", i, err)
		}

		var dstBatch []T
		if err := pgxscan.ScanAll(&dstBatch, rows); err != nil {
			return fmt.Errorf("pgx.SendBatchQuery scan at index %d: %w", i, err)
		}

		*dst = append(*dst, dstBatch...)
	}
	return nil
}

// InsertValuesPlain executes a query to insert a group of values.
// sql should be in the form "INSERT INTO table (col1, col2)" without VALUES.
// VALUES is added automatically.
func InsertValuesPlain(ctx context.Context, querier IQuerier, sql string, values []pgh.Args) error {
	if len(values) == 0 {
		return nil
	}

	var (
		args        = make(pgh.Args, 0, len(values[0])*len(values))
		sqlBuilder  strings.Builder
		columnCount = len(values[0])
	)
	sqlBuilder.WriteString(sql)
	sqlBuilder.WriteString(" VALUES ")
	for _, v := range values {
		if len(v) != columnCount {
			return fmt.Errorf("pgx.InsertValues: all values must have the same length. sql: %s", pgh.TruncSQL(sql))
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

	if _, err := querier.Exec(ctx, targetSQL, args...); err != nil {
		return fmt.Errorf("pgx.InsertValues: %w [%s]", err, pgh.TruncSQL(targetSQL))
	}

	return nil
}
