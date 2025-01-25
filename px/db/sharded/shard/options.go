package shard

import (
	"github.com/cenkalti/backoff/v5"
	"github.com/n-r-w/ctxlog"
)

// Option option for DB.
type Option func(*DB)

// WithName sets the name of the sharded DB.
func WithName(name string) Option {
	return func(s *DB) {
		s.name = name
	}
}

// WithLogger sets the logger.
func WithLogger(logger ctxlog.ILogger) Option {
	return func(s *DB) {
		s.logger = logger
	}
}

// WithRestartPolicy sets the restart policy.
func WithRestartPolicy(restartPolicy []backoff.RetryOption) Option {
	return func(s *DB) {
		s.restartPolicy = restartPolicy
	}
}
