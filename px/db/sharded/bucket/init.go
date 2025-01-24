package bucket

import (
	"context"
	"fmt"

	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
	"github.com/n-r-w/pgh/v2/px/db/shared"
)

// InitCluster initializes PostgreSQL cluster for working with buckets.
// Creates necessary schemas (buckets) and applies database query.
// bucket.DB must be successfully started using Start.
// sql contains SQL where table name is specified as __bucket__.tableName and
// will be replaced with bucket_<ID>.tableName. For example:
// CREATE TABLE IF NOT EXISTS __bucket__.tableName (id SERIAL PRIMARY KEY, name TEXT NOT NULL);.
func (b *DB[T]) InitCluster(ctx context.Context, sql string) (err error) {
	defer func() {
		if b.logger != nil {
			if err == nil {
				b.logger.Debugf(ctx, "cluster initialized")
			} else {
				b.logger.Errorf(ctx, "failed to init cluster: %v", err)
			}
		}
	}()

	// parallel execution of CREATE SCHEMA commands for the same database can lead to locks.
	// That's why we use RunShardFunc for sequential execution on each shard
	return b.RunShardFunc(ctx,
		func(ctx context.Context, shardID shard.ShardID, con shared.IConnection) error {
			return b.initClusterHelper(ctx, shardID, con, sql)
		},
	)
}

func (b *DB[T]) initClusterHelper(
	ctx context.Context, shardID shard.ShardID, con shared.IConnection, sql string,
) error {
	for _, bucket := range b.buckets {
		if bucket.ShardID != shardID {
			continue
		}

		for bucketID := bucket.BucketRange.FromID; bucketID <= bucket.BucketRange.ToID; bucketID++ {
			if err := b.shardDB.GetTxManager(shardID).Begin(ctx, func(ctxTr context.Context) error {
				// Create schema for the bucket
				_, errFunc := con.Exec(ctxTr, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", bucketID.Schema()))
				if errFunc != nil {
					return fmt.Errorf("failed to create schema for bucket %d, shard %d: %w", bucketID, shardID, errFunc)
				}

				// Execute database query
				preparedSQL := PrepareBucketSQL(sql, bucketID)
				_, errFunc = con.Exec(ctxTr, preparedSQL)
				if errFunc != nil {
					return fmt.Errorf("failed to init cluster for shard %d, bucket %d: %w", shardID, bucketID, errFunc)
				}

				if b.logger != nil {
					b.logger.Debugf(ctxTr, "bucket initialized: shard %d, bucket %d", shardID, bucketID)
				}

				return nil
			}); err != nil {
				return fmt.Errorf("failed to init cluster for shard %d, bucket %d: %w", shardID, bucketID, err)
			}
		}
	}
	return nil
}
