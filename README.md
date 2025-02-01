# PGH

A Go package that provides helper functions to combine the power of:

- [Squirrel](https://github.com/n-r-w/squirrel) - SQL builder
- [pgx](https://github.com/jackc/pgx) - PGX PostgreSQL driver
- [database/sql](https://pkg.go.dev/database/sql) - Golang SQL package
- [scany](https://github.com/georgysavva/scany) - Scanning query results into Golang structs

[![Go Reference](https://pkg.go.dev/badge/github.com/n-r-w/pgh.svg)](https://pkg.go.dev/github.com/n-r-w/pgh)
![CI Status](https://github.com/n-r-w/pgh/actions/workflows/go.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/n-r-w/pgh)](https://goreportcard.com/report/github.com/n-r-w/pgh)

## Purpose and Core Functionality

The PGH project is a Go package that provides helper functions to combine the power of several libraries:

- **Squirrel**: SQL builder
- **pgx**: PGX PostgreSQL driver
- **database/sql**: Golang SQL package
- **scany**: Scanning query results into Golang structs

## Requirements

- Go 1.23 or higher

## Key Features and Capabilities

- Helper functions for building SQL queries using Squirrel
- Functions for executing SQL queries and handling results using pgx
- Support for various query types, including modification queries, select queries, and batch queries
- Options for adding conditions, sorting, searching, and pagination to queries
- PostgreSQL error handling with specific error code support

## Additional Functionality

- [Transaction Manager (txmgr)](txmgr/README.md) - A database-agnostic transaction management system that provides clean and consistent handling of database transactions, isolation levels, and nested transactions
- [Transaction Manager implementation for PostgreSQL (pgdb)](px/db/README.md) - A PostgreSQL-specific implementation of the ITransactionInformer and ITransactionBeginner interfaces from the txmgr package
- [Client-side sharding (buckets)](px/db/buckets/README.md) - Support for distributing data across multiple database shards using virtual buckets (schemas) for PostgreSQL databases

## Getting Started

```bash
# general functionality
go get github.com/n-r-w/pgh/v2
# pgx + squirrel + scany
go get github.com/n-r-w/pgh/v2/px
# database/sql + squirrel + scany
go get github.com/n-r-w/pgh/v2/pq
```

## Usage Examples

- [Examples of using with github.com/jackc/pgx package](examples/pgx.go)
- [Examples of using with the database/sql package](examples/pq.go)
