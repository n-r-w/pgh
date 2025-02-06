# Package pq

This package is a set of helper functions for working with PostgreSQL via database/sql in Go. It is part of the [github.com/n-r-w/pgh/v2](https://github.com/n-r-w/pgh) project and provides a collection of utilities that simplify database operations using the [Squirrel](https://github.com/n-r-w/squirrel) SQL builder.

## Overview

The `pq` package provides a streamlined interface for interacting with PostgreSQL databases. It offers helper functions to manage connections, execute SQL queries, handle transactions, and generate queries using the squirrel query builder. The package is designed following clean architecture and modern Go practices, emphasizing scalability, performance, and security.

## Function Groups

### Core PostgreSQL Operations

Functions in this group encapsulate basic operations such as establishing connections, executing SQL queries, and processing query results. They provide a lightweight abstraction over the standard database/sql package to simplify common tasks when interacting with PostgreSQL.

### Query Builder Integration (Squirrel)

This group includes functions that facilitate building SQL queries using the squirrel library. They assist in dynamically constructing complex SQL queries with proper parameter binding, enhancing code clarity and reducing the risk of SQL injection vulnerabilities.

### Transaction Management

Functions in this group are focused on managing database transactions. They provide mechanisms for safely executing multiple SQL statements within a transaction, including automatic rollback and commit operations in response to errors, thereby ensuring data consistency.
