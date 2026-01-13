# HOWTO: PostgreSQL Operations with pgh/v2

## Overview

This guide covers key usage patterns for:
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/n-r-w/pgh/v2` - Query builder wrapper
- `github.com/n-r-w/pgh/v2/px` - Query execution helpers
- `github.com/n-r-w/squirrel` - SQL query builder
- `github.com/n-r-w/pgh/v2/px/db` - Database connection management
- `github.com/n-r-w/pgh/v2/txmgr` - Transaction management

## 1. Basic Query Operations

### Query Builder

Use `pgh.Builder()` to create a squirrel query builder with PostgreSQL `$1, $2...` placeholders:

```go
import (
    "github.com/n-r-w/pgh/v2"
    sq "github.com/n-r-w/squirrel"
)

query := pgh.Builder().
    Select("id, name, email").
    From("users").
    Where(sq.Eq{"status": "active"})
```

### Select Operations

```go
import "github.com/n-r-w/pgh/v2/px"

// Select multiple rows
var users []User
if err := px.Select(ctx, db, query, &users); err != nil {
    return fmt.Errorf("get users: %w", err)
}

// Select single row
var user User
if err := px.SelectOne(ctx, db, query, &user); err != nil {
    return fmt.Errorf("get user: %w", err)
}

// Process rows one by one (for large datasets)
err := px.SelectFunc(ctx, db, query, func(row pgx.Row) error {
    var u User
    if err := row.Scan(&u.ID, &u.Name, &u.Email); err != nil {
        return err
    }
    // process u...
    return nil
})
```

### Exec Operations

```go
query := pgh.Builder().
    Update("users").
    SetMap(map[string]any{
        "name":  user.Name,
        "email": user.Email,
    }).
    Where(sq.Eq{"id": user.ID})

_, err := px.Exec(ctx, tx, query)
```

### Batch Operations

```go
// Execute multiple queries in one batch
queries := make([]sq.Sqlizer, 0, len(users))
for _, user := range users {
    q := pgh.Builder().
        Insert("users").
        SetMap(map[string]any{"name": user.Name, "email": user.Email})
    queries = append(queries, q)
}
_, err := px.ExecBatch(ctx, queries, db)

// Split large batches (e.g., 100 per batch)
rowsAffected, err := px.ExecSplit(ctx, db, queries, 100)

// Insert with batch splitting
baseQuery := pgh.Builder().
    Insert("users").
    Columns("name", "email")
values := make([]pgh.Args, 0, len(users))
for _, u := range users {
    values = append(values, pgh.Args{u.Name, u.Email})
}
rowsAffected, err = px.InsertSplit(ctx, db, baseQuery, values, 100)
```

### Error Handling

```go
if err != nil {
    switch {
    case px.IsNoRows(err):
        // No rows found
    case px.IsUniqueViolation(err):
        // Unique constraint violation (PostgreSQL code 23505)
    case px.IsForeignKeyViolation(err):
        // Foreign key violation (PostgreSQL code 23503)
    default:
        return err
    }
}
```

---

## 2. Transaction Helpers: BeginTxFunc and BeginFunc

The `px` package provides low-level transaction helpers for direct pgx usage.

### BeginTxFunc

Full control over transaction options:

```go
err := px.BeginTxFunc(ctx, pool, pgx.TxOptions{
    IsoLevel:   pgx.ReadCommitted,
    AccessMode: pgx.ReadWrite,
}, func(ctx context.Context, tx pgx.Tx) error {
    // Execute queries using tx directly
    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
    if err != nil {
        return err // Transaction will be rolled back
    }
    // Or use px helpers with tx
    query := pgh.Builder().Update("users").Set("active", true).Where("id = ?", 1)
    _, err = px.Exec(ctx, tx, query)
    if err != nil {
        return err // Transaction will be rolled back
    }
    return nil // Transaction will be committed
})
```

### BeginFunc

Simplified version with default transaction options:

```go
err := px.BeginFunc(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
    _, err := tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", 100, fromID)
    if err != nil {
        return err
    }
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 100, toID)
    return err
})
```

**Key behaviors:**
- Automatic commit on success
- Automatic rollback on error or panic
- Panic is re-thrown after rollback

---

## 3. Layer Separation: txmgr + PxDB Pattern

This pattern isolates transaction management (application/usecase layer) from database operations (repository layer).

### Architecture

```
┌─────────────────────────────────────────┐
│  Application/Usecase Layer              │
│  - Uses txmgr.ITransactionManager       │
│  - Starts/manages transactions          │
│  - Orchestrates repository calls        │
└────────────────────┬────────────────────┘
                     │ ctx with transaction
                     ▼
┌─────────────────────────────────────────┐
│  Repository Layer                       │
│  - Uses pxDB.Connection(ctx)            │
│  - Unaware of transaction boundaries    │
│  - Executes queries via conn.IConnection│
└─────────────────────────────────────────┘
```

### Setup

```go
// Create PxDB instance
pxDB := db.New(
    db.WithDSN("postgres://user:password@localhost:5432/dbname"),
)

// Start the service
if err := pxDB.Start(ctx); err != nil {
    log.Fatal(err)
}
defer pxDB.Stop(ctx)

// Create transaction manager
// PxDB implements both ITransactionBeginner and ITransactionInformer
tm := txmgr.New(pxDB, pxDB)
```

### Usecase Layer

```go
type OrderUsecase struct {
    tm       txmgr.ITransactionManager
    orderRepo OrderRepository
    stockRepo StockRepository
}

func (u *OrderUsecase) CreateOrder(ctx context.Context, order Order) error {
    return u.tm.Begin(ctx, func(ctxTx context.Context) error {
        // All repository calls share the same transaction
        if err := u.stockRepo.Reserve(ctxTx, order.Items); err != nil {
            return err // Transaction rolled back
        }
        if err := u.orderRepo.Create(ctxTx, order); err != nil {
            return err // Transaction rolled back
        }
        return nil // Transaction committed
    },
        txmgr.WithTransactionLevel(txmgr.TxReadCommitted),
        txmgr.WithTransactionMode(txmgr.TxReadWrite),
    )
}
```

### Repository Layer

```go
type OrderRepositoryImpl struct {
    db *db.PxDB
}

func (r *OrderRepositoryImpl) Create(ctx context.Context, order Order) error {
    // Connection() extracts transaction from context (if present)
    // or returns pool connection (if no transaction)
    connection := r.db.Connection(ctx)
    
    query := pgh.Builder().
        Insert("orders").
        Columns("user_id", "total").
        Values(order.UserID, order.Total).
        Suffix("RETURNING id")
    
    return px.SelectOne(ctx, connection, query, &order.ID)
}

func (r *OrderRepositoryImpl) GetByID(ctx context.Context, id int64) (*Order, error) {
    connection := r.db.Connection(ctx)
    
    query := pgh.Builder().
        Select("*").
        From("orders").
        Where("id = ?", id)
    
    var order Order
    if err := px.SelectOne(ctx, connection, query, &order); err != nil {
        return nil, err
    }
    return &order, nil
}
```

### Transaction Options

```go
// Isolation levels
txmgr.WithTransactionLevel(txmgr.TxReadCommitted)   // Default
txmgr.WithTransactionLevel(txmgr.TxRepeatableRead)
txmgr.WithTransactionLevel(txmgr.TxSerializable)
txmgr.WithTransactionLevel(txmgr.TxReadUncommitted)

// Access modes
txmgr.WithTransactionMode(txmgr.TxReadWrite) // Default
txmgr.WithTransactionMode(txmgr.TxReadOnly)

// Advisory locking hint
txmgr.WithLock()
```

### Manual Transaction Control (BeginTx)

For cases requiring explicit commit/rollback control:

```go
ctxTx, finisher, err := tm.BeginTx(ctx,
    txmgr.WithTransactionLevel(txmgr.TxSerializable),
)
if err != nil {
    return err
}

// Execute operations
if err := r.orderRepo.Create(ctxTx, order); err != nil {
    _ = finisher.Rollback(ctx)
    return err
}

// Explicit commit
if err := finisher.Commit(ctx); err != nil {
    return err
}
```

### Nested Transactions

The `txmgr` handles nested calls automatically:
- If transaction exists with matching options → reuses it
- If transaction exists with different options → returns error
- No transaction → starts new one

```go
func (u *Usecase) OuterOperation(ctx context.Context) error {
    return u.tm.Begin(ctx, func(ctxTx context.Context) error {
        // This works - same transaction is reused
        return u.tm.Begin(ctxTx, func(ctxInner context.Context) error {
            // Still in the same transaction
            return u.repo.DoSomething(ctxInner)
        }) // No separate commit - outer transaction controls it
    }, txmgr.WithTransactionLevel(txmgr.TxReadCommitted))
}
```

### Bypass Transaction

Execute query outside current transaction:

```go
func (r *RepoImpl) LogOperation(ctx context.Context, msg string) error {
    // Remove transaction from context
    ctxNoTx := r.db.WithoutTransaction(ctx)
    connection := r.db.Connection(ctxNoTx)
    
    // This runs outside any transaction
    query := pgh.Builder().
        Insert("audit_log").
        SetMap(map[string]any{"message": msg})
    _, err := px.Exec(ctxNoTx, connection, query)
    return err
}
```
