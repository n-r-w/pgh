---
name: txmgr
description: Guidelines for using `github.com/n-r-w/pgh/v2/txmgr` package.
metadata:
  author: Roman Nikulenkov
---

<txmgr name="Transaction Manager Guidelines">
    This pattern isolates transaction management (application/usecase layer) from database operations (repository layer).

    <libraries>
        Related libraries:
            1. `github.com/jackc/pgx/v5`: PostgreSQL driver
            2. `github.com/n-r-w/pgh/v2`: Query builder wrapper
            3. `github.com/n-r-w/pgh/v2/px`: Query execution helpers
            4. `github.com/n-r-w/squirrel`: SQL query builder
            5. `github.com/n-r-w/pgh/v2/px/db`: Database connection management
            6. `github.com/n-r-w/pgh/v2/txmgr`: Transaction management
    </libraries>
    <architecture>
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
    </architecture>
    <use_cases_layer>
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
    </use_cases_layer>
    <repository_layer>
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
    </repository_layer>
    <transaction_options>
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
    </transaction_options>
    <transaction_hooks>
        Register hooks inside a txmgr-managed transaction when work must run after transaction finalization:

        ```go
        err := tm.Begin(ctx, func(ctxTx context.Context) error {
            if err := orderRepo.Create(ctxTx, order); err != nil {
                return err
            }

            if err := txmgr.AfterCommit(ctxTx, func(ctx context.Context) {
                // ctx has no active transaction.
                publisher.Publish(ctx, order.CreatedEvent())
            }); err != nil {
                return err
            }

            if err := txmgr.AfterRollback(ctxTx, func(ctx context.Context) {
                // ctx has no active transaction.
                metrics.TransactionRolledBack(ctx)
            }); err != nil {
                return err
            }

            return nil
        })
        ```

        Hook rules:
        1. `AfterCommit` runs only after a successful commit.
        2. `AfterRollback` runs only after a successful rollback.
        3. Hooks receive a context without an active transaction.
        4. Registering a non-nil hook outside a txmgr-managed transaction returns `txmgr.ErrNoTransactionHooks`.
        5. Nil hooks are ignored and return nil.
        6. Hook functions do not return errors. Handle hook failures inside the hook.
    </transaction_hooks>
    <manual_control>
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

        if err := txmgr.AfterCommit(ctxTx, func(ctx context.Context) {
            // ctx has no active transaction.
            publisher.Publish(ctx, order.CreatedEvent())
        }); err != nil {
            _ = finisher.Rollback(ctx)
            return err
        }

        // Explicit commit
        if err := finisher.Commit(ctxTx); err != nil {
            return err
        }
        ```
    </manual_control>
    <nested_transactions>
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
    </nested_transactions>
    <bypass_transaction>
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
    </bypass_transaction>
</txmgr>