# buckets - support for PostgreSQL client sharding

## Description

This module enables PostgreSQL client-side sharding to distribute data across different servers (shards).
Data is distributed across virtual buckets, where each bucket is a database schema. All tables belonging to one bucket are located in one database schema.
The main idea is that we initially set up a large number of buckets so that in the future there's no need to change the data distribution function
and the number of buckets. At the same time, we can move the buckets along with their tables between shards when necessary, without disrupting the overall data distribution across buckets.
When it's impossible to stop traffic, the process of moving buckets between shards is also not simple. But it's much easier than changing the data distribution function and the number of buckets.

## Usage

Package contains two main entities:

- [shard.DB](shard/shard.go) - sharded database. Contains information about all shards. Each shard is a separate database. This is a service object that is used in bucket.DB for managing connections to shards.
- [bucket.DB](bucket/bucket.go) - wrapper around shard.DB, which manages the distribution of data between buckets, which in turn distribute data between shards. For working with a sharded database, you should use bucket.DB, not shard.DB.

See the [example](/px/db/sharded/example/README.md)
