package shard

import (
	"context"
	"testing"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/require"
)

func TestShardDB(t *testing.T) {
	t.Parallel()

	var (
		ctx, cancel = context.WithTimeout(context.Background(), time.Minute)

		shard1 = ShardID(1)
		shard2 = ShardID(2)

		shardKey1 = "shard1"
		shardKey2 = "shard2"
	)
	t.Cleanup(cancel)

	_, info1 := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	_, info2 := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)

	shardFunc := func(_ context.Context, shardKey string) ShardID {
		if shardKey == shardKey1 {
			return shard1
		}
		if shardKey == shardKey2 {
			return shard2
		}

		t.Fatalf("invalid shard key: %s", shardKey)
		return 0
	}

	shardDB := NewFromDSN([]DSNInfo{
		{
			ShardID: shard1,
			DSN:     info1.DSN(),
		},
		{
			ShardID: shard2,
			DSN:     info2.DSN(),
		},
	}, shardFunc)

	require.Len(t, shardDB.GetShards(), 2)

	require.NoError(t, shardDB.Start(ctx))
	defer func() {
		require.NoError(t, shardDB.Stop(ctx))
	}()

	// create table in shard1
	_, err := shardDB.Connection(ctx, shardKey1).Exec(ctx, "CREATE TABLE test1 (id int)")
	require.NoError(t, err)

	// create table in shard2
	_, err = shardDB.Connection(ctx, shardKey2).Exec(ctx, "CREATE TABLE test2 (id int)")
	require.NoError(t, err)

	// write to shard1
	con1 := shardDB.Connection(ctx, shardKey1)
	_, err = con1.Exec(ctx, "INSERT INTO test1 (id) VALUES (1)")
	require.NoError(t, err)

	// write to shard2
	con2 := shardDB.Connection(ctx, shardKey2)
	_, err = con2.Exec(ctx, "INSERT INTO test2 (id) VALUES (2)")
	require.NoError(t, err)

	// read from shard1
	var id int
	err = pgxscan.Get(ctx, con1, &id, "SELECT id FROM test1 WHERE id=1")
	require.NoError(t, err)
	require.Equal(t, 1, id)

	// read from shard2
	err = pgxscan.Get(ctx, con2, &id, "SELECT id FROM test2 WHERE id=2")
	require.NoError(t, err)
	require.Equal(t, 2, id)
}
