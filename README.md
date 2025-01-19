# PGH (PGX Helpers)

A Go package that provides helper functions and utilities for working with [pgx](https://github.com/jackc/pgx) and [pgxscan](https://github.com/jackc/pgxscan).

[![Go Reference](https://pkg.go.dev/badge/github.com/n-r-w/pgh.svg)](https://pkg.go.dev/github.com/n-r-w/pgh)
[![Go Report Card](https://goreportcard.com/badge/github.com/n-r-w/pgh)](https://goreportcard.com/report/github.com/n-r-w/pgh)

## Key Features

- Generic SQL query execution helpers with proper error handling and context support
- Batch operations with automatic splitting of large datasets
- SQL builder integration with [Squirrel](https://github.com/n-r-w/squirrel)
- Scanning query results into structs using [scany](https://github.com/georgysavva/scany)
- PostgreSQL-specific error handling utilities

## Installation

```bash
go get github.com/n-r-w/pgh
```

## Requirements

- Go 1.22 or higher
- PostgreSQL database

## Usage Examples

### Using with SQL Builder

```go
// Build and execute a query
import (
    "context"
    sq "github.com/n-r-w/squirrel"
    "github.com/n-r-w/pgh"
)

query := pgh.Builder().
    Select("id", "name", "email").
    From("users").
    Where(sq.Eq{"status": "active"})

var users []User
err := pgh.Select(ctx, db, query, &users)
```

### Query Execution with plain SQL

```go
import (
    "context"
    "github.com/n-r-w/pgh"
)

// Execute a simple query
tag, err := pgh.ExecPlain(ctx, db, "UPDATE users SET status = $1 WHERE id = $2",
    pgh.Args{"active", userID})

// Select multiple rows into a slice
var users []User
err := pgh.SelectPlain(ctx, db,
    "SELECT id, name, email FROM users WHERE status = $1",
    &users, pgh.Args{"active"})

// Select a single row
var user User
err := pgh.SelectOnePlain(ctx, db,
    "SELECT id, name, email FROM users WHERE id = $1",
    &user, pgh.Args{userID})
```

### Batch Operations

```go
import (
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/n-r-w/pgh"
)

// Insert multiple rows in batches
values := []pgh.Args{
    {1, "John", "john@example.com"},
    {2, "Jane", "jane@example.com"},
}
rowsAffected, err := pgh.InsertSplitPlain(ctx, db,
    "INSERT INTO users (id, name, email) VALUES", // note: VALUES keyword at the end
    values, 100) // Split into batches of 100

// Execute multiple different queries in a batch
batch := &pgx.Batch{}
batch.Queue("UPDATE users SET status = $1 WHERE id = $2", "active", 1)
batch.Queue("INSERT INTO audit_log (user_id, action) VALUES ($1, $2)", 1, "status_update")

// Send batch and get total rows affected
rowsAffected, err := pgh.SendBatch(ctx, db, batch)
```

### Error Handling

```go
import "github.com/n-r-w/pgh"

// Error handling with PostgreSQL-specific error codes
if err != nil {
    if pgh.IsNoRows(err) {
        // Handle no rows found
        // Handles both pgx.ErrNoRows and PostgreSQL 'no_data_found' error code
    }
    if pgh.IsUniqueViolation(err) {
        // Handle unique constraint violation
        // Maps to PostgreSQL error code '23505'
    }
    if pgh.IsForeignKeyViolation(err) {
        // Handle foreign key violation
        // Maps to PostgreSQL error code '23503'
    }
}

// For more PostgreSQL error codes, refer to:
// https://www.postgresql.org/docs/16/errcodes-appendix.html
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
