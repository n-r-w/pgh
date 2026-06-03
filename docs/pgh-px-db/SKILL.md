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