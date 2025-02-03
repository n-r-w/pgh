# px: PostgreSQL Helper Functions Package

This package is a set of helper functions for working with PostgreSQL via [pgx/v5](https://github.com/jackc/pgx) in Go. It is part of the [github.com/n-r-w/pgh/v2](https://github.com/n-r-w/pgh) project and provides a collection of utilities that simplify database operations using the [Squirrel](https://github.com/n-r-w/squirrel) SQL builder.

## Function Groups

### 1. Query Execution Helpers

These functions execute SQL queries built with Squirrel. They include basic execution functions (`Exec`) and query functions for retrieving one or multiple rows (`SelectOne`, `Select`, and `SelectFunc`). They automatically convert Squirrel queries into executable SQL with appropriate arguments, bridging the gap between query construction and execution using pgx.

### 2. Batch Operations Helpers

Designed for handling bulk operations efficiently. This group includes:

- **Batch execution:** Functions like `ExecBatch` execute multiple queries together.
- **Splitting operations:** Functions such as `ExecSplit`, `InsertSplit`, and `InsertSplitQuery` divide large query sets into smaller batches, optimizing transaction management.
- **Batch selection:** `SelectBatch` facilitates executing multiple select queries concurrently.

### 3. Transaction Management Helpers

These functions encapsulate transaction management by automatically handling commit and rollback. The primary helpers (`BeginTxFunc` and `BeginFunc`) execute a callback within a transaction context, ensuring reliable error management and resource handling.

### 4. Error Handling Helpers

Utilities to interpret PostgreSQL error codes and provide coherent error checking. Functions like `IsNoRows`, `IsUniqueViolation`, and `IsForeignKeyViolation` detect common database errors, helping maintain consistent error handling across operations.

### 5. Squirrel Integration

Underlying all helper functions is seamless integration with Squirrel. This integration simplifies converting Squirrel queries to SQL and ensures that both simple and complex SQL operations are handled efficiently.
