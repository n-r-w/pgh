# Transaction Manager (txmgr)

A database-agnostic transaction management package that provides a clean and consistent way to handle database transactions in Go applications.

## Overview

The `txmgr` package implements a transaction management system that is independent of specific database implementations. It provides interfaces and types for managing transaction lifecycles, isolation levels, and operation modes while supporting nested transactions.

## Features

- Database-agnostic transaction management
- Support for different transaction isolation levels
- Configurable transaction modes (read-only/read-write)
- Advisory locking support
- Nested transaction handling
- Clean interface-based design

## Core Components

### Interfaces

- `ITransactionInformer`: Provides transaction state information (implemented for PostgreSQL in pgdb package)
- `ITransactionBeginner`: Handles transaction initiation (implemented for PostgreSQL in pgdb package)
- `ITransactionManager`: Main interface for transaction management

## Usage

### Creating a Transaction Manager

```go
tm := txmgr.New(beginner, informer)
```

### Starting a Transaction

```go
err := tm.Begin(ctx, func(ctxTx context.Context) error {
    // Your transactional code here
    return nil
}, txmgr.WithTransactionLevel(txmgr.TxReadCommitted),
   txmgr.WithTransactionMode(txmgr.TxReadWrite),
   txmgr.WithLock())
```

### Configuration Options

Transaction behavior can be customized using option functions:

```go
// Set isolation level
txmgr.WithTransactionLevel(txmgr.TxReadCommitted)

// Set transaction mode
txmgr.WithTransactionMode(txmgr.TxReadWrite)

// Enable advisory locking
txmgr.WithLock()
```

### Nested Transactions

The package handles nested transactions by maintaining consistent isolation levels and modes:

- If a transaction is already started, the package verifies that the requested isolation level and mode match the current transaction
- If they match, the function executes within the current transaction
- If they don't match, an error is returned

## Important Notes

1. Transaction options (isolation level and mode) cannot be changed once a transaction has started
2. The Lock option is advisory and its implementation depends on the underlying database driver
3. Default isolation level is `TxReadCommitted`
4. Default transaction mode is `TxReadWrite`
5. The concrete implementations of `ITransactionInformer` and `ITransactionBeginner` for PostgreSQL are provided in the pgdb package
