# DB Package

Package `db` provides PostgreSQL database functionality using the `pgx` driver with built-in connection pooling, transaction management, and query logging capabilities.

## Features

- Transaction management with different isolation levels and access modes
- Connection pooling via `pgxpool`
- Query logging support
- Tracing and metrics support
- Large objects support (within transactions)
- Batch operations for bulk data processing
- Service-based architecture with start/stop lifecycle management
- Implementation of the ITransactionInformer and ITransactionBeginner interfaces from the [txmgr](../../txmgr/README.md) package

## Usage

```go
import (   
    "github.com/n-r-w/pgh/v2/px/db"
)
```

### Creating a DB Instance

```go
db := db.New(
    db.WithName("mydb"),
    db.WithDSN("postgres://user:password@localhost:5432/dbname"),
    db.WithLogQueries(), // Enable query logging
)

// Start the service
if err := db.Start(context.Background()); err != nil {
    log.Fatal(err)
}
defer db.Stop(context.Background())
```

### Options

The `db.New()` function accepts various options to configure the database connection:

- `WithName(name string)` - Sets service name for logging
- `WithDSN(dsn string)` - Sets connection string
- `WithPool(pool *pgxpool.Pool)` - Sets existing connection pool
- `WithConfig(cfg *pgxpool.Config)` - Sets pool configuration
- `WithLogQueries()` - Enables query logging
- `WithRestartPolicy(policy github.com/cenkalti/backoff/v5)` - Sets restart policy on errors. Only works when using <https://github.com/n-r-w/bootstrap>
- `WithAfterStartFunc(f func(context.Context, *PxDB) error)` - Sets function to run after successful start
- `WithLogger(logger ctxlog.ILogger)` - Sets custom logger implementation

### Transaction Management

```go
err := db.Begin(ctx, func(ctxTr context.Context) error {
        // Use the database within transaction
        connection := db.Connection(ctxTr)
        
        _, err := connection.Exec(ctxTr, "INSERT INTO users (name) VALUES ($1)", "John")
        if err != nil {
            return err // Transaction will be rolled back
        }
        
        return nil // Transaction will be committed
    }, 
    txmgr.Options{}, // default options
)
```

### Query Execution

Implements the `IConnection` interface from the [conn](../../conn/README.md) package, which allows executing SQL queries, batch operations, large objects, CopyFrom, and other operations.

### Pinned Sessions

Use `RunPinnedSession` when ordinary queries and several short transactions must use one PostgreSQL server session. Pass the callback context to repositories and transaction-manager calls. `Connection` selects the pinned connection or its active transaction from that context.

```go
tm := txmgr.New(database, database)

err := database.RunPinnedSession(ctx, func(sessionCtx context.Context) error {
    if _, err := database.Connection(sessionCtx).Exec(
        sessionCtx,
        "select pg_advisory_lock($1)",
        lockID,
    ); err != nil {
        return err
    }

    if err := tm.Begin(sessionCtx, func(txCtx context.Context) error {
        return repository.Update(txCtx)
    }); err != nil {
        return err
    }

    return repository.Read(sessionCtx)
}, db.WithPinnedSessionCloseConnection())
```

The callback scope acquires one pool connection. After success, error, cancellation, or panic, it returns the connection to the pool by default or closes it when configured to do so. A committed or rolled-back transaction does not dispose of that connection. Query, batch, and copy operations through saved session contexts or connections return `ErrPinnedSessionClosed` after the callback ends. Large-object operations remain transaction-scoped and return the ended transaction error, such as `pgx.ErrTxClosed`. A lost pinned connection returns its connection error and is not replaced from the pool.

Use `WithPinnedSessionCloseConnection` when session-level state, such as an advisory lock, makes the connection unsafe to return to the pool:

```go
err := database.RunPinnedSession(
    ctx,
    runWork,
    db.WithPinnedSessionCloseConnection(),
)
```

The close option closes only the acquired connection. It does not close the supplied pool. Independent pinned-session callbacks can use different connections from the same pool.

Do not use one pinned session concurrently. Serialize its queries and transactions, close rows and batch results before the next operation, and stop session-using goroutines before the callback returns. During an active transaction, a query through the session context or through `WithoutTransaction` returns `ErrPinnedSessionTransactionActive`; use the transaction context instead. After the transaction finishes, transaction hooks and later session queries use the pinned connection outside an SQL transaction.

## Telemetry Package

See the [telemetry](./telemetry/README.md) package for more information.

## Support for PostgreSQL client sharding

See the [sharded](./sharded/README.md) module for more information.
