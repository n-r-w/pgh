package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/require"
)

func TestPxDB(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, informer := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)

	pgdbImpl := New(
		WithName("test"),
		WithDSN(informer.DSN()),
		WithLogQueries(),
	)

	require.Equal(t, "test", pgdbImpl.name)

	// запуск
	ctxStart, cancelStart := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancelStart)
	require.NoError(t, pgdbImpl.Start(ctxStart))

	// проверка реализации txmgr.ITransactionManager

	// транзакция не начата
	require.False(t, pgdbImpl.InTransaction(ctx))

	// проверка databaseWrapper
	iwrapper := pgdbImpl.Connection(ctx, conn.WithLogQueries())
	wrapper, _ := iwrapper.(*Wrapper)
	require.NotNil(t, wrapper)
	require.NotNil(t, wrapper.db)
	require.Nil(t, wrapper.tx)
	require.True(t, wrapper.logQueries)

	// начало транзакции
	require.Error(t, pgdbImpl.Begin(ctx, func(ctxTr context.Context) error {
		require.True(t, pgdbImpl.InTransaction(ctxTr))

		// проверка databaseWrapper
		iwrapperTran := pgdbImpl.Connection(ctxTr)
		require.NotNil(t, iwrapperTran)
		wrapperTran, _ := iwrapperTran.(*Wrapper)
		require.NotNil(t, wrapperTran.db)
		require.NotNil(t, wrapperTran.tx)
		require.True(t, wrapperTran.logQueries)

		// создаем таблицу в БД
		_, errTran := wrapperTran.Exec(ctxTr, "CREATE TABLE test (id int)")
		require.NoError(t, errTran)

		// вставляем данные в БД
		_, errTran = wrapperTran.Exec(ctxTr, "INSERT INTO test (id) VALUES (1), (2), (3)")
		require.NoError(t, errTran)

		// выбираем данные из БД через QueryRow
		var id int
		require.NoError(t, pgxscan.Get(ctxTr, wrapperTran, &id, "SELECT id FROM test WHERE id=1"))
		require.Equal(t, 1, id)

		// выбираем все данные из БД через Query
		var ids []int

		errTran = pgxscan.Select(ctxTr, wrapperTran, &ids, "SELECT id FROM test")
		require.NoError(t, errTran)
		require.Equal(t, []int{1, 2, 3}, ids)

		// проверяем SendBatch
		batch := &pgx.Batch{}
		batch.Queue("SELECT id FROM test WHERE id=1")
		batch.Queue("SELECT id FROM test WHERE id=2")
		batch.Queue("SELECT id FROM test WHERE id=3")

		ids = []int{}
		batchResult := wrapperTran.SendBatch(ctxTr, batch)
		for range batch.Len() {
			var rows pgx.Rows
			rows, errTran = batchResult.Query()
			require.NoError(t, errTran)
			require.NoError(t, rows.Err())
			var idsBatch []int
			require.NoError(t, pgxscan.ScanAll(&idsBatch, rows))
			ids = append(ids, idsBatch...)
		}
		require.NoError(t, batchResult.Close())
		require.Equal(t, []int{1, 2, 3}, ids)

		// проверяем CopyFrom
		_, errTran = wrapperTran.CopyFrom(ctxTr, pgx.Identifier{"test"}, []string{"id"}, pgx.CopyFromRows([][]any{{4}, {5}, {6}}))
		require.NoError(t, errTran)
		ids = []int{}
		require.NoError(t, pgxscan.Select(ctxTr, wrapperTran, &ids, "SELECT id FROM test"))
		require.Equal(t, []int{1, 2, 3, 4, 5, 6}, ids)

		// проверяем LargeObjects
		lObjects := wrapperTran.LargeObjects()
		require.NotNil(t, lObjects)

		loID, errTran := lObjects.Create(ctxTr, 0)
		require.NoError(t, errTran)
		require.NotZero(t, loID)

		lObj, errTran := lObjects.Open(ctxTr, loID, pgx.LargeObjectModeWrite)
		require.NoError(t, errTran)

		count, errTran := lObj.Write([]byte("test"))
		require.NoError(t, errTran)
		require.Equal(t, 4, count)
		require.NoError(t, lObj.Close())

		lObj, errTran = lObjects.Open(ctxTr, loID, pgx.LargeObjectModeRead)
		require.NoError(t, errTran)
		data := make([]byte, 4)
		count, errTran = lObj.Read(data)
		require.NoError(t, errTran)
		require.Equal(t, 4, count)
		require.Equal(t, []byte("test"), data)
		require.NoError(t, lObj.Close())

		require.NoError(t, lObjects.Unlink(ctxTr, loID))
		_, errTran = lObjects.Open(ctxTr, loID, pgx.LargeObjectModeRead)
		require.Error(t, errTran)

		return errors.New("rollback")
	}, txmgr.Options{}))

	// транзакция не начата
	require.False(t, pgdbImpl.InTransaction(ctx))

	// данные в БД не попали (нет таблицы test)
	_, err := wrapper.Exec(ctx, "SELECT * FROM test")
	require.Error(t, err)

	// остановка
	ctxStop, cancelStop := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancelStop)
	require.NoError(t, pgdbImpl.Stop(ctxStop))
}
