---
name: pgh-px-db
description: Guidelines for using `github.com/n-r-w/pgh/v2/px/db` package.
metadata:
  author: Roman Nikulenkov
---

<px_db name="PxDB Example">
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
    ```
</px_db>

<pinned_sessions>
    1. Use `pxDB.RunPinnedSession(ctx, callback, opts...)` when queries and short transactions need one PostgreSQL session.
    2. Pass the callback context to repositories. Obtain query wrappers with `pxDB.Connection(sessionCtx)` or `pxDB.Connection(txCtx)`. Do not reuse a wrapper created outside the session.
    3. Use `txmgr` for short transactions. Follow [transaction rules](../txmgr/SKILL.md).
    4. Run one operation at a time per session. Close rows and batch results before the next operation. Wait for all goroutines that use the session before the callback returns.
    5. Do not use session contexts or wrappers after the callback ends. After connection loss, return the error. Start any new attempt in a new session scope.
    6. Use `db.WithPinnedSessionCloseConnection()` if session state is unsafe for pool reuse. The application must manage advisory locks.
    7. With an application-owned pool, use `db.New(db.WithPool(pool))`. Do not call `PxDB.Start` or `PxDB.Stop`. The application manages the pool.
    8. Do not nest session scopes or open one from a transaction context.
    9. Use a direct PostgreSQL connection or session pooling. Do not use transaction pooling.
</pinned_sessions>
