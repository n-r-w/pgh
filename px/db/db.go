package db

import (
	"context"
	"fmt"

	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/ctxlog"
	"github.com/n-r-w/pgh/v2/px/db/conn"

	"github.com/cenkalti/backoff/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx postgres driver
)

// PxDB service for working with PostgreSQL database. Implements IService interface.
type PxDB struct {
	name           string
	restartPolicy  []backoff.RetryOption
	dsn            string
	logQueries     bool
	afterStartFunc func(context.Context, *PxDB) error

	config *pgxpool.Config
	pool   *pgxpool.Pool

	logger ctxlog.ILogger
}

var _ bootstrap.IService = (*PxDB)(nil)

// New creates a new instance of PxDB.
func New(opt ...Option) *PxDB {
	p := &PxDB{
		name:   "pgdb",
		logger: ctxlog.NewStubWrapper(),
	}

	for _, o := range opt {
		o(p)
	}

	return p
}

// Start starts the service.
func (p *PxDB) Start(ctx context.Context) (err error) {
	p.logger.Debug(ctx, "starting pgdb", "database", p.name)

	defer func() {
		if err == nil && p.afterStartFunc != nil {
			err = p.afterStartFunc(ctx, p)
			if err != nil {
				err = fmt.Errorf("failed to run after start function: %w", err)
			}
		}
	}()

	var pool *pgxpool.Pool
	if p.config != nil {
		pool, err = pgxpool.NewWithConfig(ctx, p.config)
	} else {
		pool, err = pgxpool.New(ctx, p.dsn)
	}
	if err != nil {
		return fmt.Errorf("failed to create pgx pool for database %s: %w", p.name, err)
	}

	p.logger.Debug(ctx, "checking database connection", "database", p.name)

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to connect to database %s: %w", p.name, err)
	}

	p.pool = pool

	p.logger.Debug(ctx, "database connection established", "database", p.name)

	return nil
}

// Stop stops the service.
func (p *PxDB) Stop(_ context.Context) error {
	if p.pool != nil {
		p.pool.Close()
	}

	return nil
}

// Info returns service information.
func (p *PxDB) Info() bootstrap.Info {
	return bootstrap.Info{
		Name:          p.name,
		RestartPolicy: p.restartPolicy,
	}
}

// Connection extracts transaction/pool from context and returns database interface implementation.
// Use only at repository level. Returns IConnection interface implementation.
func (p *PxDB) Connection(ctx context.Context, opt ...conn.ConnectionOption) conn.IConnection {
	opts := &conn.ConnectionOptionData{}
	for _, o := range opt {
		o(opts)
	}

	it, ok := txFromContext(ctx)
	if !ok {
		return newDatabaseWrapperNoTran(p, opts.LogQueries || p.logQueries)
	}

	if p != it.db {
		panic("invalid DB") // this should never happen
	}

	return newDatabaseWrapperWithTran(p, it.tx, it.opts, opts.LogQueries || p.logQueries)
}
