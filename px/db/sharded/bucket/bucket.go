// Package bucket provides functionality for working with database buckets.
package bucket

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
	"golang.org/x/sync/errgroup"
)

const (
	// BucketAlias bucket alias in SQL queries.
	BucketAlias = "__bucket__"
	// BucketPrefix prefix for bucket table names.
	BucketPrefix = "bucket_"
)

type bucketContextKeyType struct{}

var bucketContextKey bucketContextKeyType //nolint:gochecknoglobals // ok

// PrepareBucketSQL replaces bucket aliases in the query with their table names.
func PrepareBucketSQL(sql string, bucketID BucketID) string {
	return strings.ReplaceAll(sql, BucketAlias, BucketPrefix+bucketID.String())
}

// ToContext puts BucketID into context.
func ToContext(ctx context.Context, bucketID BucketID) context.Context {
	return context.WithValue(ctx, bucketContextKey, bucketID)
}

// FromContext extracts BucketID from context.
func FromContext(ctx context.Context) (BucketID, bool) {
	bucketID, ok := ctx.Value(bucketContextKey).(BucketID)
	return bucketID, ok
}

// BucketID bucket identifier.
type BucketID uint //nolint:revive // used as is

// String converts BucketID to string.
func (b BucketID) String() string {
	return strconv.Itoa(int(b)) //nolint:gosec // safe
}

// Schema returns the database schema name for the bucket.
func (b BucketID) Schema() string {
	return BucketPrefix + b.String()
}

// BucketRange range of buckets.
type BucketRange struct { //nolint:revive // used as is
	FromID BucketID
	ToID   BucketID
}

// Count number of buckets in the range.
func (b BucketRange) Count() int {
	return int(b.ToID - b.FromID + 1) //nolint:gosec // safe
}

// NewBucketRange creates a range of buckets.
func NewBucketRange(fromID, toID BucketID) *BucketRange {
	return &BucketRange{
		FromID: fromID,
		ToID:   toID,
	}
}

// Contains checks if the bucket id is within the range.
func (b BucketRange) Contains(bucketID BucketID) bool {
	return bucketID >= b.FromID && bucketID <= b.ToID
}

// ShardKeyToBucketIDFunc function to get bucket id by sharding key.
type ShardKeyToBucketIDFunc[T any] func(shardKey T) BucketID

// UniformBucketFn returns a function that uniformly distributes any keys across n buckets.
// uses string as a key.
func UniformBucketFn(n int) func(shardKey string) BucketID {
	return func(key string) BucketID {
		h := fnv.New32a()
		_, _ = io.WriteString(h, key)
		return BucketID(int(h.Sum32()) % n) //nolint:gosec // safe
	}
}

// BucketInfo information about a bucket.
type BucketInfo struct { //nolint:revive // used as is
	ShardID     shard.ShardID
	BucketRange *BucketRange
}

// BucketCount number of buckets.
func BucketCount(buckets []*BucketInfo) int { //nolint:revive // used as is
	count := 0
	for _, bucket := range buckets {
		count += bucket.BucketRange.Count()
	}
	return count
}

// DB is a wrapper around shard.DB, which manages the distribution of data between buckets,
// which in turn distribute data between shards.
// For working with a sharded database, you should use bucket.DB, not shard.DB.
type DB[T any] struct {
	shardDB                *shard.DB
	shardKeyToBucketIDFunc ShardKeyToBucketIDFunc[T]
	buckets                []*BucketInfo
	infoByShard            map[shard.ShardID]*BucketInfo
	shardByBucketID        map[BucketID]shard.ShardID
	name                   string
	runBucketFuncLimit     int
	logger                 ctxlog.ILogger

	afterStartFunc func(context.Context, *DB[T]) error
}

var _ bootstrap.IService = (*DB[any])(nil)

// New creates a sharded DB with bucket support.
func New[T any](shardDB *shard.DB, buckets []*BucketInfo,
	shardKeyToBucketIDFunc ShardKeyToBucketIDFunc[T], opts ...Option[T],
) *DB[T] {
	const defaultRunBucketFuncLimit = 10

	b := &DB[T]{
		name:                   "bucket_db",
		shardDB:                shardDB,
		shardKeyToBucketIDFunc: shardKeyToBucketIDFunc,
		afterStartFunc:         nil,
		buckets:                buckets,
		infoByShard:            make(map[shard.ShardID]*BucketInfo, len(buckets)),
		shardByBucketID:        make(map[BucketID]shard.ShardID),
		logger:                 ctxlog.NewStubWrapper(),
		runBucketFuncLimit:     defaultRunBucketFuncLimit,
	}

	for _, bucket := range buckets {
		b.infoByShard[bucket.ShardID] = bucket
		for bucketID := bucket.BucketRange.FromID; bucketID <= bucket.BucketRange.ToID; bucketID++ {
			b.shardByBucketID[bucketID] = bucket.ShardID
		}
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// GetFunc returns a function to get bucket id by sharding key.
func (b *DB[T]) GetFunc() ShardKeyToBucketIDFunc[T] {
	return b.shardKeyToBucketIDFunc
}

// GetShardID returns shardID for the specified bucketID.
func (b *DB[T]) GetShardID(bucketID BucketID) (shard.ShardID, error) {
	if shardID, ok := b.shardByBucketID[bucketID]; ok {
		return shardID, nil
	}

	return 0, fmt.Errorf("bucket %d not found", bucketID)
}

// GetBucketByKey returns ShardID, BucketID for the specified key.
func (b *DB[T]) GetBucketByKey(key T) (shard.ShardID, BucketID, error) {
	bucketID := b.shardKeyToBucketIDFunc(key)
	shardID, err := b.GetShardID(bucketID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get shard id for bucket %d: %w", bucketID, err)
	}

	return shardID, bucketID, nil
}

// Start launches the service.
func (b *DB[T]) Start(ctx context.Context) (err error) {
	defer func() {
		if err == nil && b.afterStartFunc != nil {
			err = b.afterStartFunc(ctx, b)
			if err != nil {
				err = fmt.Errorf("failed to run after start function: %w", err)
			}
		}
	}()

	return b.shardDB.Start(ctx)
}

// Stop stops the service.
func (b *DB[T]) Stop(ctx context.Context) error {
	return b.shardDB.Stop(ctx)
}

// Info returns service information.
func (b *DB[T]) Info() bootstrap.Info {
	return b.shardDB.Info()
}

// Connection returns IConnection interface implementation for the specified sharding key.
func (b *DB[T]) Connection(ctx context.Context, shardKey T, opt ...conn.ConnectionOption) conn.IConnection {
	var d conn.IConnection
	if shardID, bucketID, err := b.GetBucketByKey(shardKey); err != nil {
		d = conn.NewDatabaseErrorWrapper(err)
	} else {
		d = newBucketWrapper(b.ShardConnection(ctx, shardID, opt...), bucketID)
	}
	return d
}

// ShardConnection returns IConnection interface implementation for the specified shardID.
func (b *DB[T]) ShardConnection(ctx context.Context, shardID shard.ShardID,
	opt ...conn.ConnectionOption,
) conn.IConnection {
	return b.shardDB.Connection(ctx, shardID.String(), opt...)
}

// NewBatch creates a new Batch based on the key.
func (b *DB[T]) NewBatch(bucketID BucketID) *Batch {
	return NewBatch(bucketID)
}

// Exec executes a query without returning data.
func (b *DB[T]) Exec(ctx context.Context, shardKey T, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return b.Connection(ctx, shardKey).Exec(ctx, sql, arguments...)
}

// Query executes a query and returns the result.
func (b *DB[T]) Query(ctx context.Context, shardKey T, sql string, args ...any) (pgx.Rows, error) {
	return b.Connection(ctx, shardKey).Query(ctx, sql, args...)
}

// QueryRow gets a connection and executes a query that should return no more than one row.
// Errors are deferred until pgx.Row.Scan method is called. If the query doesn't select a row,
// pgx.Row.Scan will return pgx.ErrNoRows.
// Otherwise, pgx.Row.Scan scans the first selected row and discards the rest.
func (b *DB[T]) QueryRow(ctx context.Context, shardKey T, sql string, args ...any) pgx.Row {
	return b.Connection(ctx, shardKey).QueryRow(ctx, sql, args...)
}

// SendBatch sends a set of queries for execution, combining all queries into one package.
func (b *DB[T]) SendBatch(ctx context.Context, shardKey T, batch *pgx.Batch) pgx.BatchResults {
	return b.Connection(ctx, shardKey).SendBatch(ctx, batch)
}

// LargeObjects supports working with large objects and is only available within
// a transaction (this is a postgresql limitation).
// Outside of a transaction, it will panic.
func (b *DB[T]) LargeObjects(ctx context.Context, shardKey T) pgx.LargeObjects {
	return b.Connection(ctx, shardKey).LargeObjects()
}

// CopyFrom implements bulk data insertion into a table.
func (b *DB[T]) CopyFrom(ctx context.Context, shardKey T, tableName pgx.Identifier,
	columnNames []string, rowSrc pgx.CopyFromSource,
) (n int64, err error) {
	return b.Connection(ctx, shardKey).CopyFrom(ctx, tableName, columnNames, rowSrc)
}

// NewBucketCluster creates connections with shards immediately and wraps everything in bucket.DB.
// Helper to simplify bucket.DB creation.
func NewBucketCluster(shardInfo []*shard.ShardInfo, bucketInfo []*BucketInfo,
	bucketOpts ...Option[string],
) *DB[string] {
	shardDB := shard.New(shardInfo, shard.DefaultShardFunc)
	bucketDB := New(shardDB, bucketInfo, UniformBucketFn(BucketCount(bucketInfo)), bucketOpts...)

	return bucketDB
}

// NewBucketClusterFromDSN creates connections with shards immediately and wraps everything in bucket.DB.
// Connections to shards are created using DSN.
// Helper to simplify bucket.DB creation.
func NewBucketClusterFromDSN(dsn []shard.DSNInfo, bucketInfo []*BucketInfo,
	shardOpts []shard.Option,
	bucketOpts []Option[string],
) *DB[string] {
	tempDB := &DB[string]{
		shardDB:                nil,
		shardKeyToBucketIDFunc: nil,
		buckets:                nil,
		infoByShard:            nil,
		shardByBucketID:        nil,
		name:                   "",
		runBucketFuncLimit:     0,
		logger:                 nil,
		afterStartFunc:         nil,
	}
	for _, o := range bucketOpts {
		o(tempDB)
	}

	shardDB := shard.NewFromDSN(dsn, shard.DefaultShardFunc, shardOpts...)
	bucketDB := New(shardDB, bucketInfo, UniformBucketFn(BucketCount(bucketInfo)), bucketOpts...)

	return bucketDB
}

// RunBucketFunc executes a function for all buckets in the cluster.
// The order of buckets is not defined.
func (b *DB[T]) RunBucketFunc(ctx context.Context,
	f func(ctx context.Context, shardID shard.ShardID, bucketID BucketID, con conn.IConnection) error,
) error {
	errGroup, ctxGroup := errgroup.WithContext(ctx)
	errGroup.SetLimit(b.runBucketFuncLimit)

	_ = b.shardDB.RunFunc(ctxGroup,
		func(ctxFunc context.Context, shardID shard.ShardID, con conn.IConnection) error {
			for _, bucketInfo := range b.buckets {
				if shardID != bucketInfo.ShardID {
					continue
				}

				for bucketID := bucketInfo.BucketRange.FromID; bucketID <= bucketInfo.BucketRange.ToID; bucketID++ {
					bucketIDCopy := bucketID
					errGroup.Go(func() error {
						bucketCon := newBucketWrapper(con, bucketIDCopy)
						return f(ctxFunc, shardID, bucketIDCopy, bucketCon)
					})
				}
			}

			return nil
		},
		0) // no parallel execution in RunFunc because we have errGroup.Go in the loop above

	err := errGroup.Wait()
	if err != nil {
		return fmt.Errorf("failed to run function for cluster: %w", err)
	}

	return nil
}

// RunShardFunc executes a function for all shards in the cluster.
// The order of shards is not defined.
func (b *DB[T]) RunShardFunc(ctx context.Context, f func(ctx context.Context,
	shardID shard.ShardID, con conn.IConnection) error,
) error {
	return b.shardDB.RunFunc(ctx, f, len(b.infoByShard))
}

// GroupByShard groups objects by cluster shards based on a function that returns a key for each object.
func GroupByShard[KEYT any, OBJT any](
	db *DB[KEYT],
	objs []OBJT,
	f func(v OBJT) BucketID,
) (map[shard.ShardID][]OBJT, error) {
	res := make(map[shard.ShardID][]OBJT)

	for _, obj := range objs {
		bucketID := f(obj)

		shardID, err := db.GetShardID(bucketID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shard id for bucket %d: %w", bucketID, err)
		}

		res[shardID] = append(res[shardID], obj)
	}

	return res, nil
}

// GroupByShardFunc groups objects by cluster shards based on function
// fGroup, which returns a key for each object.
// Then function fCall is called for each group.
func GroupByShardFunc[KEYT any, OBJT any](
	ctx context.Context,
	db *DB[KEYT],
	objs []OBJT,
	fGroup func(v OBJT) BucketID,
	fCall func(ctx context.Context, shardID shard.ShardID, objs []OBJT) error,
) error {
	groups, err := GroupByShard(db, objs, fGroup)
	if err != nil {
		return fmt.Errorf("failed to group by shard: %w", err)
	}

	errGroup, ctxGroup := errgroup.WithContext(ctx)
	for shardID, objs := range groups {
		shardIDCopy := shardID
		objsCopy := objs
		errGroup.Go(func() error {
			if errCall := fCall(ctxGroup, shardIDCopy, objsCopy); errCall != nil {
				return fmt.Errorf("failed to fCall for shard %d: %w", shardID, errCall)
			}
			return nil
		})
	}

	return errGroup.Wait()
}
