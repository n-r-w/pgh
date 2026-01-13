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

	// startup
	ctxStart, cancelStart := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancelStart)
	require.NoError(t, pgdbImpl.Start(ctxStart))

	// verify txmgr.ITransactionManager implementation

	// transaction not started
	require.False(t, pgdbImpl.InTransaction(ctx))

	// verify databaseWrapper
	iwrapper := pgdbImpl.Connection(ctx, conn.WithLogQueries())
	wrapper, _ := iwrapper.(*Wrapper)
	require.NotNil(t, wrapper)
	require.NotNil(t, wrapper.db)
	require.Nil(t, wrapper.tx)
	require.True(t, wrapper.logQueries)

	// begin transaction
	require.Error(t, pgdbImpl.Begin(ctx, func(ctxTr context.Context) error {
		require.True(t, pgdbImpl.InTransaction(ctxTr))

		// verify databaseWrapper
		iwrapperTran := pgdbImpl.Connection(ctxTr)
		require.NotNil(t, iwrapperTran)
		wrapperTran, _ := iwrapperTran.(*Wrapper)
		require.NotNil(t, wrapperTran.db)
		require.NotNil(t, wrapperTran.tx)
		require.True(t, wrapperTran.logQueries)

		// create table in DB
		_, errTran := wrapperTran.Exec(ctxTr, "CREATE TABLE test (id int)")
		require.NoError(t, errTran)

		// insert data into DB
		_, errTran = wrapperTran.Exec(ctxTr, "INSERT INTO test (id) VALUES (1), (2), (3)")
		require.NoError(t, errTran)

		// select data from DB via QueryRow
		var id int
		require.NoError(t, pgxscan.Get(ctxTr, wrapperTran, &id, "SELECT id FROM test WHERE id=1"))
		require.Equal(t, 1, id)

		// select all data from DB via Query
		var ids []int

		errTran = pgxscan.Select(ctxTr, wrapperTran, &ids, "SELECT id FROM test")
		require.NoError(t, errTran)
		require.Equal(t, []int{1, 2, 3}, ids)

		// verify SendBatch
		//nolint:exhaustruct // external type, QueuedQueries is managed by Queue method
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

		// verify CopyFrom
		_, errTran = wrapperTran.CopyFrom(ctxTr, pgx.Identifier{"test"}, []string{"id"}, pgx.CopyFromRows([][]any{{4}, {5}, {6}}))
		require.NoError(t, errTran)
		ids = []int{}
		require.NoError(t, pgxscan.Select(ctxTr, wrapperTran, &ids, "SELECT id FROM test"))
		require.Equal(t, []int{1, 2, 3, 4, 5, 6}, ids)

		// verify LargeObjects
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
	}, txmgr.Options{})) //nolint:exhaustruct // external type, zero values are acceptable defaults

	// transaction not started
	require.False(t, pgdbImpl.InTransaction(ctx))

	// data not inserted into DB (no test table)
	_, err := wrapper.Exec(ctx, "SELECT * FROM test")
	require.Error(t, err)

	// shutdown
	ctxStop, cancelStop := context.WithTimeout(ctx, 2*time.Second)
	t.Cleanup(cancelStop)
	require.NoError(t, pgdbImpl.Stop(ctxStop))
}
