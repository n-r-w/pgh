# Bucket Sharding Example

This example demonstrates how to use the PostgreSQL client-side sharding functionality provided by the `buckets` package. It shows:

1. Setting up a sharded database cluster with multiple buckets
2. Creating tables across all buckets
3. Inserting data using batch operations
4. Reading data from specific buckets
5. Running operations across all buckets

## Prerequisites

- PostgreSQL servers running on ports 5432 and 5433
- Two databases created: 'shard1' and 'shard2'
- PostgreSQL user 'postgres' with password 'postgres' having access to both databases

## How to Run

1. Make sure your PostgreSQL servers are running
2. Create the required databases:

```sql
CREATE DATABASE shard1;
CREATE DATABASE shard2;
```

3. Run the example:

```bash
go run main.go
```

## What the Example Does

1. **Cluster Setup**: Creates a sharded cluster with:
   - Shard 1: Contains buckets 0-4
   - Shard 2: Contains buckets 5-9

2. **Table Creation**: Creates a `users` table in each bucket with the structure:

   ```sql
   CREATE TABLE users (
       id SERIAL PRIMARY KEY,
       name TEXT NOT NULL,
       email TEXT NOT NULL UNIQUE
   )
   ```

3. **Data Operations**:
   - Inserts test users across different buckets
   - Demonstrates batch operations for efficient data insertion
   - Shows how to read data using the shard key (email in this case)
   - Counts total users across all buckets

4. **Bucket Functions**: Shows how to run operations across all buckets in parallel

## Key Concepts

- **Bucket Aliases**: Tables are created using `__bucket__` alias which gets replaced with actual bucket schema names
- **Shard Keys**: Email is used as a shard key to determine which bucket stores the data
- **Batch Operations**: Multiple operations are batched together for better performance
- **Parallel Processing**: Operations across buckets can be executed in parallel
