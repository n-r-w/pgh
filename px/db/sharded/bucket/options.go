package bucket

import (
	"context"

	"github.com/n-r-w/ctxlog"
)

// Option option for bucket.DB.
type Option[T any] func(*DB[T])

// WithName sets the name of the sharded DB.
func WithName[T any](name string) Option[T] {
	return func(b *DB[T]) {
		b.name = name
	}
}

// WithRunLimit sets the limit of parallel executions in RunBucketFunc method.
func WithRunLimit[T any](limit int) Option[T] {
	return func(b *DB[T]) {
		b.runBucketFuncLimit = limit
	}
}

// WithAfterStartFunc sets a function that will be called after successful service startup.
func WithAfterStartFunc[T any](f func(context.Context, *DB[T]) error) Option[T] {
	return func(b *DB[T]) {
		b.afterStartFunc = f
	}
}

// WithLogger sets the logger.
func WithLogger[T any](logger ctxlog.ILogger) Option[T] {
	return func(b *DB[T]) {
		b.logger = logger
	}
}
