package bucket

import (
	"context"
	"log/slog"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
	"github.com/n-r-w/pgh/v2/px/db/shared"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/require"
)

func TestBucketDB(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	slog.SetLogLoggerLevel(slog.LevelDebug)
	logger, err := pgh.NewSlogLogger(slog.Default(), "bucketDBtest")
	require.NoError(t, err)

	_, info1 := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)
	_, info2 := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)

	var (
		shard1 = shard.ShardID(1)
		shard2 = shard.ShardID(2)
	)

	bucketDB := NewBucketClusterFromDSN(
		[]shard.DSNInfo{
			{
				ShardID: shard1,
				DSN:     info1.DSN(),
				Options: []db.Option{
					db.WithLogPxDBQueries(),
				},
			},
			{
				ShardID: shard2,
				DSN:     info2.DSN(),
				Options: []db.Option{
					db.WithLogPxDBQueries(),
				},
			},
		},
		[]*BucketInfo{
			{
				ShardID:     shard1,
				BucketRange: NewBucketRange(0, 4),
			},
			{
				ShardID:     shard2,
				BucketRange: NewBucketRange(5, 9),
			},
		},
		[]shard.Option{
			shard.WithLogger(logger),
		},
		[]Option[string]{
			WithLogger[string](logger),
		},
	)

	require.NoError(t, bucketDB.Start(ctx))

	// create tables with buckets
	require.NoError(t, bucketDB.InitCluster(ctx, "CREATE TABLE __bucket__.test (id bigint PRIMARY KEY)"))

	// add data to buckets
	batch := NewShardBatch(bucketDB)
	for i := range 10 {
		shardKey := strconv.Itoa(i)
		require.NoError(t, batch.Queue(shardKey, "INSERT INTO __bucket__.test (id) VALUES ($1)", i))
	}
	require.NoError(t, batch.ExecAll(ctx))

	// check that the total amount of data in the buckets is correct
	var totalCount atomic.Int64
	require.NoError(t, bucketDB.RunBucketFunc(ctx,
		func(ctx context.Context, shardID shard.ShardID, bucketID BucketID, con shared.IConnection) error {
			var count int
			require.NoError(t, pgxscan.Get(ctx, con, &count, "SELECT COUNT(*) FROM __bucket__.test"))
			totalCount.Add(int64(count))
			return nil
		},
	))
	require.Equal(t, int64(10), totalCount.Load())
}
