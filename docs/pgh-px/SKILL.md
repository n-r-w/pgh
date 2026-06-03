---
name: pgh-px
description: Guidelines for using `github.com/n-r-w/pgh/v2/px` package.
metadata:
  author: Roman Nikulenkov
---

<px>
  <core_model>
    1. `px.IQuerier` is the query interface. It contains `Query` and `Exec`.
    2. `px.IBatcher` is the batch interface. It contains `SendBatch`.
    3. `px.ITransactionBeginner` is the transaction-start interface. It contains `BeginTx`.
    4. `pgxpool.Pool`, `pgx.Conn`, and `pgx.Tx` can satisfy these narrow interfaces when their methods match the required operation.
    5. Prefer the narrow `px` interfaces in application code when a function only needs query, batch, or transaction-start behavior.
  </core_model>

  <query_helper_selection>
    1. Use Squirrel helpers when the query is built with `github.com/n-r-w/squirrel`:
      1) `px.Exec` for write queries.
      2) `px.Select` for many rows scanned into `*[]T`.
      3) `px.SelectOne` for one row scanned into `*T`.
      4) `px.SelectFunc` for row-by-row processing.
    2. Use plain SQL helpers when SQL text is already built:
      1) `px.ExecPlain` for write queries.
      2) `px.SelectPlain` for many rows scanned into `*[]T`.
      3) `px.SelectOnePlain` for one row scanned into `*T`.
      4) `px.SelectFuncPlain` for row-by-row processing.
    3. Use `pgh.Builder()` for new Squirrel queries passed to `px` helpers. It configures dollar placeholders.
    4. Pass query parameters as `pgh.Args` when calling plain helpers.
    5. Do not pass a slice destination to `SelectOne` or `SelectOnePlain`; pass a pointer to one value.
    6. Treat `pgx.ErrNoRows` from `SelectOne` and `SelectOnePlain` as a normal not-found signal when the caller expects optional data.
  </query_helper_selection>

  <transactions>
    1. Use `px.BeginFunc` when default `pgx.TxOptions{}` are enough.
    2. Use `px.BeginTxFunc` when transaction options are required.
    3. Put all transaction work inside the callback passed to `BeginFunc` or `BeginTxFunc`.
    4. Use the `ctx` and `tx` values passed into the callback. Do not ignore them and use an outer transaction or background context.
    5. Do not call `Commit` or `Rollback` inside the callback.
    6. Return an error from the callback when transaction work fails. `BeginFunc` and `BeginTxFunc` handle rollback.
    7. Do not add nested transaction helpers unless the target connection and business logic explicitly require nested transaction behavior.
  </transactions>

  <batch_and_split_helpers>
    1. Use `px.ExecBatch` for multiple Squirrel write queries.
    2. Use `px.SelectBatch` for multiple Squirrel select queries whose rows should be appended into one destination slice.
    3. Use `px.ExecSplit` or `px.ExecSplitPlain` when a list of independent queries must be executed in groups.
    4. For `px.ExecSplitPlain`, `queries` and `args` must have the same length. `args[i]` belongs to `queries[i]`.
    5. Use `px.InsertSplit` when inserting many rows with a Squirrel `InsertBuilder` and no returned rows are needed.
    6. Use `px.InsertSplitQuery` when inserting many rows with a Squirrel `InsertBuilder` and `RETURNING` rows must be scanned.
    7. Use `px.InsertSplitPlain` only when SQL matches its argument shape. It queues each group as one argument, `values[idx:idxTo]`, not expanded row placeholders.
    8. Use `px.InsertValuesPlain` when SQL has the form `INSERT INTO table (col1, col2)` without `VALUES`; the helper appends `VALUES` and placeholders.
    9. `splitSize` must be greater than zero for split helpers. Validate or choose it before calling the helper.
    10. Empty batches are no-ops in `SendBatch` and `SendBatchQuery`.
    11. Keep row value lengths consistent for `InsertValuesPlain`; mixed lengths return an error.
  </batch_and_split_helpers>

  <direct_data_access>
    Use `github.com/georgysavva/scany/v2/pgxscan` when code must work directly with query results instead of using `px.Select`, `px.SelectOne`, `px.SelectPlain`, or `px.SelectOnePlain`.
  </direct_data_access>

  <error_handling>
    1. Use `px.IsNoRows` to check not-found errors from `pgx.ErrNoRows` or PostgreSQL `NoDataFound`.
    2. Use `px.IsUniqueViolation` for PostgreSQL unique constraint violations.
    3. Use `px.IsForeignKeyViolation` for PostgreSQL foreign key constraint violations.
    4. Use `github.com/jackc/pgerrcode` directly when code needs PostgreSQL error classes or codes not wrapped by `px`.
    5. Preserve wrapped SQL errors with `%w` when adding context, so these helpers and `errors.Is` or `errors.As` keep working.
  </error_handling>
</px>
