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
    db.WithLogPxDBQueries(), // Enable query logging
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
- `WithLogPxDBQueries()` - Enables query logging
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

## Telemetry Package

See the [telemetry](./telemetry/README.md) package for more information.

## Support for PostgreSQL client sharding

See the [sharded](./sharded/README.md) module for more information.
