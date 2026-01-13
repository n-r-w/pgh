package shard

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/cenkalti/backoff/v5"
	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/px/db/telemetry"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
)

// ShardID shard identifier.
//
//nolint:revive // exported type ShardID is acceptable despite stuttering
type ShardID uint

// String converts ShardID to string.
func (s ShardID) String() string {
	return strconv.Itoa(int(s)) //nolint:gosec // safe
}

// ShardFunc function to get shard by key.
//
//nolint:revive // exported type ShardFunc is acceptable despite stuttering
type ShardFunc func(ctx context.Context, shardKey string) ShardID

// DefaultShardFunc default function for determining shard number by shardKey.
var DefaultShardFunc = func(_ context.Context, shardKey string) ShardID { //nolint:gochecknoglobals // ok
	shardID, err := strconv.Atoi(shardKey)
	if err != nil {
		// If for some reason there's no shard found for the bucket, return a non-existent shard.
		return math.MaxInt32
	}

	return ShardID(shardID)
}

// ShardInfo information about a shard.
//
//nolint:revive // exported type ShardInfo is acceptable despite stuttering
type ShardInfo struct {
	ShardID    ShardID
	Connector  db.IStartStopConnector
	TxBeginner txmgr.ITransactionBeginner
	TxInformer txmgr.ITransactionInformer

	txManager *txmgr.TransactionManager
}

// NewInfoPxDB helper function, that creates shard information based on db.PxDB.
// NewInfoPxDB creates a new ShardInfo from PxDB. telemetry is optional.
func NewInfoPxDB(
	shardID ShardID,
	pgdb *db.PxDB,
	t telemetry.ITelemetry,
) *ShardInfo {
	i := &ShardInfo{
		ShardID:    shardID,
		Connector:  pgdb,
		TxBeginner: pgdb,
		TxInformer: pgdb,
		txManager:  nil,
	}

	if t != nil {
		i.Connector = telemetry.New(pgdb, t)
	} else {
		i.Connector = pgdb
	}

	return i
}

// DB sharded database.
// Contains information about all shards. Each shard is a separate database.
// This is a service object that is used in bucket.DB for managing connections to shards.
type DB struct {
	shardFunc     ShardFunc
	shardInfo     []*ShardInfo // few shards, so map is not used
	name          string
	logger        ctxlog.ILogger
	restartPolicy []backoff.RetryOption
}

var _ bootstrap.IService = (*DB)(nil)

// New creates a sharded database.
func New(shardInfo []*ShardInfo, shardFunc ShardFunc, opts ...Option) *DB {
	s := &DB{
		shardFunc:     shardFunc,
		shardInfo:     shardInfo,
		name:          "",
		logger:        ctxlog.NewStubWrapper(),
		restartPolicy: nil,
	}

	for _, opt := range opts {
		opt(s)
	}

	for _, info := range s.shardInfo {
		info.txManager = txmgr.New(info.TxBeginner, info.TxInformer)
	}

	return s
}

// DSNInfo information about PxDB with DSN.
type DSNInfo struct {
	ShardID ShardID
	DSN     string
	Options []db.Option
}

// NewFromDSN creates a sharded database by creating PxDB based on DSN.
func NewFromDSN(dsn []DSNInfo, shardFunc ShardFunc, opts ...Option) *DB {
	s := &DB{
		shardFunc:     shardFunc,
		shardInfo:     nil,
		name:          "",
		logger:        ctxlog.NewStubWrapper(),
		restartPolicy: nil,
	}

	for _, opt := range opts {
		opt(s)
	}

	shardInfo := make([]*ShardInfo, 0, len(dsn))

	for _, dsn := range dsn {
		pgdbOpts := append([]db.Option{
			db.WithDSN(dsn.DSN),
			db.WithName(fmt.Sprintf("shard-%d", dsn.ShardID)),
			db.WithLogger(s.logger),
			db.WithRestartPolicy(s.restartPolicy...),
		}, dsn.Options...)

		pgdb := db.New(pgdbOpts...)
		shardInfo = append(shardInfo, &ShardInfo{
			ShardID:    dsn.ShardID,
			Connector:  pgdb,
			TxBeginner: pgdb,
			TxInformer: pgdb,
			txManager:  nil,
		})
	}

	return New(shardInfo, shardFunc, opts...)
}

// GetShards returns information about shards.
func (s *DB) GetShards() []ShardID {
	return lo.Map(s.shardInfo, func(info *ShardInfo, _ int) ShardID {
		return info.ShardID
	})
}

// GetTxManager returns transaction manager for the shard.
func (s *DB) GetTxManager(shardID ShardID) txmgr.ITransactionManager {
	for _, info := range s.shardInfo {
		if info.ShardID == shardID {
			return info.txManager
		}
	}
	return nil
}

// GetFunc returns a function to get shard id by sharding key.
func (s *DB) GetFunc() ShardFunc {
	return s.shardFunc
}

// Start launches the service.
func (s *DB) Start(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	for _, info := range s.shardInfo {
		infoCopy := info
		errGroup.Go(func() error {
			if err := infoCopy.Connector.Start(ctx); err != nil {
				return fmt.Errorf("failed to start shard db %d: %w", infoCopy.ShardID, err)
			}
			return nil
		})
	}

	return errGroup.Wait()
}

// Stop stops the service.
func (s *DB) Stop(ctx context.Context) error {
	var (
		errTotal error
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(len(s.shardInfo))

	for _, info := range s.shardInfo {
		go func(info *ShardInfo) {
			defer wg.Done()
			if err := info.Connector.Stop(ctx); err != nil {
				mu.Lock()
				errTotal = errors.Join(errTotal, fmt.Errorf("failed to stop shard db %d: %w", info.ShardID, err))
				mu.Unlock()
			}
		}(info)
	}

	wg.Wait()

	return errTotal
}

// Info returns information about the shard.
func (s *DB) Info() bootstrap.Info {
	return bootstrap.Info{
		Name:          s.name,
		RestartPolicy: s.restartPolicy,
	}
}

func (s *DB) getShardInfo(ctx context.Context, shardKey string) *ShardInfo {
	shardID := s.shardFunc(ctx, shardKey)

	for _, info := range s.shardInfo {
		if info.ShardID == shardID {
			return info
		}
	}

	return nil
}

// Connection returns IConnection interface implementation for the specified shardKey.
func (s *DB) Connection(ctx context.Context, shardKey string, opt ...conn.ConnectionOption) conn.IConnection {
	info := s.getShardInfo(ctx, shardKey)
	if info == nil {
		return conn.NewDatabaseErrorWrapper(fmt.Errorf("shard %s not found", shardKey))
	}

	return info.Connector.Connection(ctx, opt...)
}

// Begin starts a function in a transaction for the specified shardKey.
func (s *DB) Begin(ctx context.Context, shardKey string,
	f func(context.Context) error, opts ...txmgr.Option,
) error {
	info := s.getShardInfo(ctx, shardKey)
	if info == nil {
		return fmt.Errorf("shard %s not found", shardKey)
	}

	return info.txManager.Begin(ctx, f, opts...)
}

// WithoutTransaction returns context without transaction for the specified shardKey.
func (s *DB) WithoutTransaction(ctx context.Context, shardKey string) context.Context {
	info := s.getShardInfo(ctx, shardKey)
	if info == nil {
		s.logger.Error(ctx, "without transaction failed", "reason", "shard not found", "shardKey", shardKey)
		return ctx
	}

	return info.txManager.WithoutTransaction(ctx)
}

// RunFunc executes a function for all shards.
// The order of shards is not defined.
// runParallel specifies the number of goroutines to use for parallel execution.
// If runParallel is 0, the function will be executed in the sequential way.
func (s *DB) RunFunc(ctx context.Context,
	f func(ctx context.Context, shardID ShardID, con conn.IConnection) error,
	runParallel int,
) error {
	var eg *errgroup.Group

	if runParallel > 0 {
		eg, ctx = errgroup.WithContext(ctx)
		eg.SetLimit(runParallel)
	}

	for _, info := range s.shardInfo {
		if eg != nil {
			eg.Go(func() error {
				con := info.Connector.Connection(ctx)
				if err := f(ctx, info.ShardID, con); err != nil {
					return fmt.Errorf("failed to run function for shard %d: %w", info.ShardID, err)
				}

				return nil
			})
		} else {
			con := info.Connector.Connection(ctx)
			if err := f(ctx, info.ShardID, con); err != nil {
				return fmt.Errorf("failed to run function for shard %d: %w", info.ShardID, err)
			}
		}
	}

	if eg != nil {
		return eg.Wait()
	}

	return nil
}
