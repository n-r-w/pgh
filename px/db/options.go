package db

import (
	"context"

	"github.com/cenkalti/backoff/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/n-r-w/ctxlog"
)

// Option option for PxDB.
type Option func(*PxDB)

// WithPool sets connection pool when creating a PxDB instance.
func WithPool(pool *pgxpool.Pool) Option {
	return func(p *PxDB) {
		p.pool = pool
	}
}

// WithName sets service name.
func WithName(name string) Option {
	return func(p *PxDB) {
		p.name = name
	}
}

// WithDSN sets DSN for database connection.
// If WithConfig is used, this option is ignored.
func WithDSN(dsn string) Option {
	return func(p *PxDB) {
		p.dsn = dsn
	}
}

// WithRestartPolicy sets service restart policy on error.
// Only works when using https://github.com/n-r-w/bootstrap
func WithRestartPolicy(policy ...backoff.RetryOption) Option {
	return func(p *PxDB) {
		p.restartPolicy = policy
	}
}

// WithConfig sets connection pool configuration.
func WithConfig(cfg *pgxpool.Config) Option {
	return func(p *PxDB) {
		p.config = cfg
	}
}

// WithLogPxDBQueries enables query logging at the PxDB service level.
func WithLogPxDBQueries() Option {
	return func(p *PxDB) {
		p.logQueries = true
	}
}

// WithAfterStartFunc sets a function that will be called after successful service start.
func WithAfterStartFunc(f func(context.Context, *PxDB) error) Option {
	return func(p *PxDB) {
		p.afterStartFunc = f
	}
}

// WithLogger sets the logger.
func WithLogger(logger ctxlog.ILogger) Option {
	return func(p *PxDB) {
		p.logger = logger
	}
}
