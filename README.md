# PGH

A Go package that provides helper functions to combine the power of:

- [Squirrel](https://github.com/n-r-w/squirrel) - SQL builder
- [pgx](https://github.com/jackc/pgx) - PGX PostgreSQL driver
- [database/sql](https://pkg.go.dev/database/sql) - Golang SQL package
- [scany](https://github.com/georgysavva/scany) - Scanning query results into Golang structs

[![Go Reference](https://pkg.go.dev/badge/github.com/n-r-w/pgh.svg)](https://pkg.go.dev/github.com/n-r-w/pgh/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/n-r-w/pgh)](https://goreportcard.com/report/github.com/n-r-w/pgh/v2)

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
