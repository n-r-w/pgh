//nolint:prealloc //ok
package px

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2"
	sq "github.com/n-r-w/squirrel"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// suffixImpl реализация sq.Sqlizer для мока suffix
type suffixImpl struct {
	sq.Sqlizer
}

func (s suffixImpl) ToSql() (string, []any, error) {
	return "ON CONFLICT DO NOTHING", nil, nil
}

// Test_InsertSplitPlain_SendBatch проверяет работу InsertSplitPlain, SendBatch
func Test_InsertSplit_SendBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		rowsCount = 100
		splitSize = 9
	)

	var (
		rows           []pgh.Args
		processedBatch int
		processedExecs int
		processedCount int
		columns        = sq.Insert("test").Columns("id", "name").SuffixExpr(suffixImpl{})
		expected       int
	)

	if rowsCount%splitSize == 0 {
		expected = rowsCount / splitSize
	} else {
		expected = rowsCount/splitSize + 1
	}

	for i := range rowsCount {
		rows = append(rows, pgh.Args{i, fmt.Sprintf("test %d", i)})
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	batchMock := NewMockIBatcher(mc)
	batchMock.EXPECT().SendBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
			batchResultMock := NewMockBatchResults(mc)
			batchResultMock.EXPECT().Exec().DoAndReturn(
				func() (pgconn.CommandTag, error) {
					processedExecs++
					return pgconn.NewCommandTag("1"), nil
				}).Times(expected)
			batchResultMock.EXPECT().Close().DoAndReturn(
				func() error {
					processedCount++
					return nil
				})

			processedBatch += batch.Len()
			return batchResultMock
		})

	ra, err := InsertSplit(ctx, batchMock, columns, rows, splitSize)
	require.NoError(t, err)
	require.Equal(t, int64(expected), ra)
	require.Equal(t, expected, processedBatch)
	require.Equal(t, expected, processedExecs)
	require.Equal(t, 1, processedCount)
}

func Test_InsertSplit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		rowsCount = 100
		splitSize = 9
	)

	var (
		rows           []pgh.Args
		processedBatch int
		processedExecs int
		processedCount int
		expected       int
	)

	if rowsCount%splitSize == 0 {
		expected = rowsCount / splitSize
	} else {
		expected = rowsCount/splitSize + 1
	}

	for i := range rowsCount {
		rows = append(rows, pgh.Args{i, fmt.Sprintf("test %d", i)})
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	batchMock := NewMockIBatcher(mc)
	batchMock.EXPECT().SendBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
			batchResultMock := NewMockBatchResults(mc)
			batchResultMock.EXPECT().Exec().DoAndReturn(
				func() (pgconn.CommandTag, error) {
					processedExecs++
					return pgconn.NewCommandTag("1"), nil
				}).Times(expected)
			batchResultMock.EXPECT().Close().DoAndReturn(
				func() error {
					processedCount++
					return nil
				})

			processedBatch += batch.Len()
			return batchResultMock
		})

	ra, err := InsertSplitPlain(ctx, batchMock,
		"INSERT INTO test_table (name, age) VALUES ($1, $2)",
		rows,
		splitSize)
	require.NoError(t, err)
	require.Equal(t, int64(expected), ra)
	require.Equal(t, expected, processedBatch)
	require.Equal(t, expected, processedExecs)
	require.Equal(t, 1, processedCount)
}

func Test_InsertSplitQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		rowsCount = 100
		splitSize = 9
	)

	var (
		rows           []pgh.Args
		processedBatch int
		processedCount int
		totalParts     = int(math.Ceil(float64(rowsCount) / float64(splitSize)))
	)

	var expectedIDs []int64
	for i := range rowsCount {
		rows = append(rows, pgh.Args{i, fmt.Sprintf("test %d", i)})
		expectedIDs = append(expectedIDs, int64(i+1))
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	batchMock := NewMockIBatcher(mc)
	scannedID := int64(0)
	batchMock.EXPECT().SendBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, batch *pgx.Batch) pgx.BatchResults {
			batchResultMock := NewMockBatchResults(mc)
			batchResultMock.EXPECT().Close().DoAndReturn(
				func() error {
					processedCount++
					return nil
				})

			processedBatch += batch.Len()

			mockRows := NewMockRows(mc)
			mockRows.EXPECT().Err().Return(nil).AnyTimes()
			mockRows.EXPECT().Close().Return().AnyTimes()
			mockRows.EXPECT().FieldDescriptions().Return([]pgconn.FieldDescription{
				{Name: "id"},
			}).AnyTimes()

			mockRows.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dst any) error {
				scannedID++
				id, _ := dst.(*int64)
				*id = scannedID
				return nil
			}).AnyTimes()
			batchResultMock.EXPECT().Query().Return(mockRows, nil).AnyTimes()

			lastPartSize := int(math.Mod(float64(rowsCount), float64(splitSize)))
			for i := range totalParts {
				if i == (totalParts - 1) {
					mockRows.EXPECT().Next().Return(true).Times(lastPartSize)
					mockRows.EXPECT().Next().Return(false)
					continue
				}
				mockRows.EXPECT().Next().Return(true).Times(splitSize)
				mockRows.EXPECT().Next().Return(false)
			}
			return batchResultMock
		})

	insertSQL := pgh.Builder().Insert("test_table").Columns("name", "age").Suffix("RETURNING id")
	var actualIDs []int64
	err := InsertSplitQuery(ctx, batchMock,
		insertSQL,
		rows,
		splitSize,
		&actualIDs,
	)
	require.NoError(t, err)
	require.Equal(t, expectedIDs, actualIDs)
	require.Equal(t, 1, processedCount)
}

// Test_SendBatchQuery проверяет работу SendBatchQuery
func Test_SendBatchQuery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	batch := &pgx.Batch{}
	batch.Queue("SELECT * FROM test_table WHERE id = $1", 1)
	batch.Queue("SELECT * FROM test_table WHERE id = $1", 2)
	batch.Queue("SELECT * FROM test_table WHERE id = $1", 3)

	rowsMock := NewMockRows(mc)

	nextCount := 0
	rowsMock.EXPECT().Next().DoAndReturn(
		func() bool {
			nextCount++
			return nextCount <= 3
		},
	).AnyTimes()
	rowsMock.EXPECT().Close().Return().AnyTimes()
	rowsMock.EXPECT().Err().Return(nil).AnyTimes()
	rowsMock.EXPECT().FieldDescriptions().Return([]pgconn.FieldDescription{{Name: "id"}}).AnyTimes()
	rowsMock.EXPECT().Scan(gomock.Any()).DoAndReturn(
		func(dst ...any) error {
			require.Len(t, dst, 1)
			return nil
		},
	).AnyTimes()

	batchResultMock := NewMockBatchResults(mc)
	batchResultMock.EXPECT().Query().Return(
		rowsMock,
		nil,
	).Times(3)
	batchResultMock.EXPECT().Close().Return(nil).Times(1)

	batchMock := NewMockIBatcher(mc)
	batchMock.EXPECT().SendBatch(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, batch *pgx.Batch) (pgx.BatchResults, error) {
			require.Equal(t, 3, batch.Len())
			return batchResultMock, nil
		},
	).Times(1)

	var dst []pgh.Args
	err := SendBatchQuery(ctx, batchMock, batch, &dst)
	require.NoError(t, err)
	require.Len(t, dst, 3)
}

// Test_ExecSplit проверяет работу ExecSplitPlain, ExecSplit
func Test_ExecSplit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	batchResultMock := NewMockBatchResults(mc)
	batchResultMock.EXPECT().Exec().Return(pgconn.NewCommandTag("1"), nil).Times(6)
	batchResultMock.EXPECT().Close().Return(nil).Times(4)

	batchMock := NewMockIBatcher(mc)
	batchLen := 0
	batchMock.EXPECT().SendBatch(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, batch *pgx.Batch) (pgx.BatchResults, error) {
			batchLen += batch.Len()
			return batchResultMock, nil
		},
	).Times(4)

	var queries []string
	var args []pgh.Args
	for i := range 3 {
		queries = append(queries, "SELECT * FROM test_table WHERE id = $1")
		args = append(args, pgh.Args{i})
	}

	splitSize := 2

	rowsAffected, err := ExecSplitPlain(ctx, batchMock, queries, args, splitSize)
	require.NoError(t, err)
	require.Equal(t, int64(3), rowsAffected)
	require.Equal(t, 3, batchLen)

	// Проверяем ExecSplit, т.к. он использует ExecSplitPlain
	var queriesSQ []sq.Sqlizer
	for i := range queries {
		queriesSQ = append(queriesSQ, pgh.Builder().Select("*").From("test_table").Where(sq.Eq{"id": i}))
	}

	rowsAffected, err = ExecSplit(ctx, batchMock, queriesSQ, splitSize)
	require.NoError(t, err)
	require.Equal(t, int64(3), rowsAffected)
	require.Equal(t, 6, batchLen)
}

func TestInsertValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	querier := NewMockIQuerier(mc)

	querier.EXPECT().Exec(gomock.Any(),
		"INSERT INTO users (name, age) VALUES ($1,$2),($3,$4),($5,$6)",
		"John Doe", 30,
		"Jane Smith", 25,
		"Bob", 20,
	).Return(pgconn.CommandTag{}, nil)

	sql := "INSERT INTO users (name, age)"
	values := []pgh.Args{
		{"John Doe", 30},
		{"Jane Smith", 25},
		{"Bob", 20},
	}

	err := InsertValuesPlain(ctx, querier, sql, values)
	require.NoError(t, err)
}

func TestSelectFuncPlain(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuerier := NewMockIQuerier(ctrl)
	mockRows := NewMockRows(ctrl)

	ctx := context.Background()
	sql := "SELECT * FROM table"
	args := pgh.Args{1, "test"}

	mockQuerier.EXPECT().Query(ctx, sql, args...).Return(mockRows, nil)

	mockRows.EXPECT().Next().Return(true).Times(2)
	mockRows.EXPECT().Next().Return(false).Times(1)
	mockRows.EXPECT().Err().Return(nil)
	mockRows.EXPECT().Close()

	var called int
	err := SelectFuncPlain(ctx, mockQuerier, sql, args, func(_ pgx.Row) error {
		called++
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 2, called)
}

func TestExecBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	querier := NewMockIBatcher(mc)

	querier.EXPECT().SendBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *pgx.Batch) pgx.BatchResults {
			batchResultMock := NewMockBatchResults(mc)
			batchResultMock.EXPECT().Exec().DoAndReturn(
				func() (pgconn.CommandTag, error) {
					return pgconn.NewCommandTag("3"), nil
				}).Times(3)
			batchResultMock.EXPECT().Close().DoAndReturn(
				func() error {
					return nil
				})

			return batchResultMock
		})

	queries := []sq.Sqlizer{
		pgh.Builder().Insert("users").Columns("name", "age").Values("John Doe", 30),
		pgh.Builder().Insert("users").Columns("name", "age").Values("Jane Smith", 25),
		pgh.Builder().Insert("users").Columns("name", "age").Values("Bob", 20),
	}

	_, err := ExecBatch(ctx, queries, querier)
	require.NoError(t, err)
}
