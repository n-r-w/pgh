package db

import (
	"context"
	"fmt"

	"github.com/n-r-w/bootstrap"
	"github.com/n-r-w/pgh/v2"
	"github.com/n-r-w/pgh/v2/px/db/shared"

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

	logger pgh.ILogger
}

var _ bootstrap.IService = (*PxDB)(nil)

// New creates a new instance of PxDB.
func New(opt ...Option) *PxDB {
	p := &PxDB{}

	for _, o := range opt {
		o(p)
	}

	if p.name == "" {
		p.name = "pxdb"
	}

	return p
}

// Start starts the service.
func (p *PxDB) Start(ctx context.Context) (err error) {
	if p.logger != nil {
		p.logger.Debugf(ctx, "starting pgdb for database %s", p.name)
	}

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

	if p.logger != nil {
		p.logger.Debugf(ctx, "checking connection to database %s", p.name)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to connect to database %s: %w", p.name, err)
	}

	p.pool = pool

	if p.logger != nil {
		p.logger.Debugf(ctx, "connected to database %s", p.name)
	}

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
func (p *PxDB) Connection(ctx context.Context, opt ...shared.ConnectionOption) shared.IConnection {
	opts := &shared.ConnectionOptionData{}
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
