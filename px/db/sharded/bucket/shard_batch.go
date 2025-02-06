package bucket

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
)

type shardBatchInfo struct {
	pgxBatch  pgx.Batch
	res       pgx.BatchResults
	processed int
}

// ShardBatch analog of pgx.ShardBatch, with distribution of queries across shards.
type ShardBatch[TKEY any] struct {
	db        *DB[TKEY]
	batchInfo map[shard.ShardID]*shardBatchInfo
	closed    bool
}

// NewShardBatch creates a new ShardBatch.
func NewShardBatch[TKEY any](db *DB[TKEY]) *ShardBatch[TKEY] {
	return &ShardBatch[TKEY]{
		db:        db,
		batchInfo: make(map[shard.ShardID]*shardBatchInfo),
	}
}

// Queue adds a query to ShardBatch.
func (b *ShardBatch[TKEY]) Queue(key TKEY, sql string, args ...any) error {
	if b.closed {
		return errors.New("Batch.Queue: closed")
	}

	shardID, bucketID, err := b.db.GetBucketByKey(key)
	if err != nil {
		return fmt.Errorf("Batch.Queue: %w", err)
	}

	info, ok := b.batchInfo[shardID]
	if !ok {
		info = &shardBatchInfo{}
		b.batchInfo[shardID] = info
	}

	info.pgxBatch.Queue(PrepareBucketSQL(sql, bucketID), args...)

	return nil
}

// Len returns the number of queries in ShardBatch.
func (b *ShardBatch[TKEY]) Len() (n int) {
	for _, info := range b.batchInfo {
		n += info.pgxBatch.Len()
	}

	return n
}

// Send sends the batch for execution for each shard.
func (b *ShardBatch[TKEY]) Send(ctx context.Context) error {
	if b.closed {
		return errors.New("Batch.Send: closed")
	}

	wg := sync.WaitGroup{}
	wg.Add(len(b.batchInfo))

	for shardID, info := range b.batchInfo {
		go func(shardID shard.ShardID, info *shardBatchInfo) {
			defer wg.Done()
			info.res = b.db.ShardConnection(ctx, shardID).SendBatch(ctx, &info.pgxBatch)
		}(shardID, info)
	}

	wg.Wait()

	return nil
}

func (b *ShardBatch[TKEY]) nextResult() (pgx.BatchResults, error) {
	if b.closed {
		return nil, errors.New("Batch.nextResult: closed")
	}

	for _, info := range b.batchInfo {
		if info.processed == info.pgxBatch.Len() {
			continue
		}

		info.processed++

		if info.res == nil {
			return nil, errors.New("Batch.nextResult: batch was not sent")
		}

		return info.res, nil
	}

	return nil, errors.New("Batch.nextResult: no more results")
}

// Exec executes batch sequentially for each shard. The order of shard selection is not defined.
func (b *ShardBatch[TKEY]) Exec() (pgconn.CommandTag, error) {
	res, err := b.nextResult()
	if err != nil {
		return pgconn.CommandTag{}, err
	}

	return res.Exec()
}

// ExecAll executes all queries and then closes the batch.
func (b *ShardBatch[TKEY]) ExecAll(ctx context.Context) error {
	if err := b.Send(ctx); err != nil {
		return fmt.Errorf("ShardBatchExecAll: %w", err)
	}
	defer func() {
		err := b.Close()
		if err != nil {
			b.db.logger.Error(ctx, "failed to close batch", "error", err)
		}
	}()

	for range b.Len() {
		if _, err := b.Exec(); err != nil {
			return fmt.Errorf("ShardBatchExecAll: %w", err)
		}
	}

	return nil
}

// Query executes batch sequentially for each shard. The order of shard selection is not defined.
func (b *ShardBatch[TKEY]) Query() (pgx.Rows, error) {
	res, err := b.nextResult()
	if err != nil {
		return nil, err
	}

	return res.Query()
}

// QueryRow executes batch sequentially for each shard. The order of shard selection is not defined.
func (b *ShardBatch[TKEY]) QueryRow() pgx.Row {
	res, err := b.nextResult()
	if err != nil {
		return conn.NewErrRow(err)
	}

	return res.QueryRow()
}

// Close closes the Batch.
func (b *ShardBatch[TKEY]) Close() error {
	b.closed = true

	var errFound error
	for _, info := range b.batchInfo {
		if info.res == nil {
			continue
		}

		err := info.res.Close()
		if err == nil {
			continue
		}

		errFound = errors.Join(errFound, err)
	}

	return errFound
}

// ShardBatchQueryAllFunc executes queries, reads all rows from all
// query results and passes them to f, then closes the batch.
func ShardBatchQueryAllFunc[TKEY any, TRES any](ctx context.Context, batch *ShardBatch[TKEY],
	f func(ctx context.Context, rows []TRES) error,
) error {
	if err := batch.Send(ctx); err != nil {
		return fmt.Errorf("ShardBatchQueryAllFunc: %w", err)
	}
	defer func() {
		if err := batch.Close(); err != nil {
			batch.db.logger.Error(ctx, "failed to close batch", "error", err)
		}
	}()

	for range batch.Len() {
		var (
			rows pgx.Rows
			err  error
		)
		if rows, err = batch.Query(); err != nil {
			return err
		}

		var d []TRES
		if err = pgxscan.ScanAll(&d, rows); err != nil {
			return err
		}

		if err = f(ctx, d); err != nil {
			return err
		}
	}

	return nil
}

// ShardBatchQueryAll executes queries, reads all rows from all query results
// and passes them to dst, then closes the batch.
func ShardBatchQueryAll[TKEY any, TRES any](ctx context.Context, batch *ShardBatch[TKEY], dst *[]TRES) error {
	return ShardBatchQueryAllFunc(ctx, batch, func(_ context.Context, rows []TRES) error {
		*dst = append(*dst, rows...)
		return nil
	})
}
