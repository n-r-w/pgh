//nolint:mnd,forbidigo,gocritic // ok
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/px/db/sharded/bucket"
	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
)

func main() {
	ctx := context.Background()

	// Setting up a PostgreSQL cluster with two shards
	var (
		shard1 = shard.ShardID(1)
		shard2 = shard.ShardID(2)
	)

	// Initialize bucket DB with two shards
	// In real applications, you would use actual DSN strings
	bucketDB := bucket.NewBucketClusterFromDSN(
		[]shard.DSNInfo{
			{
				ShardID: shard1,
				DSN:     "postgres://postgres:password1@localhost:5432/shard1", // Example DSN
			},
			{
				ShardID: shard2,
				DSN:     "postgres://postgres:password@localhost:5432/shard2", // Example DSN
			},
		},
		[]*bucket.BucketInfo{
			{
				ShardID:     shard1,
				BucketRange: bucket.NewBucketRange(0, 4), // Buckets 0-4 on shard1
			},
			{
				ShardID:     shard2,
				BucketRange: bucket.NewBucketRange(5, 9), // Buckets 5-9 on shard2
			},
		},
		[]shard.Option{},
		[]bucket.Option[string]{bucket.WithName[string]("example-cluster")}, // Optional: give the cluster a name
	)

	// Start the bucket DB
	if err := bucketDB.Start(ctx); err != nil {
		log.Fatalf("Failed to start bucket DB: %v", err)
	}
	defer func() {
		if err := bucketDB.Stop(ctx); err != nil {
			log.Fatalf("Failed to stop bucket DB: %v", err)
		}
	}()

	// Initialize the cluster by creating tables in each bucket
	// __bucket__ will be replaced with actual bucket names (bucket_0, bucket_1, etc.)
	if err := bucketDB.InitCluster(ctx, `
		CREATE TABLE IF NOT EXISTS __bucket__.users (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE
		)
	`); err != nil {
		log.Fatalf("Failed to initialize cluster: %v", err)
	}

	// Insert data into buckets using batch operations
	batch := bucket.NewShardBatch(bucketDB)

	users := []struct {
		Name  string
		Email string
	}{
		{"User 1", "user1@example.com"},
		{"User 2", "user2@example.com"},
		{"User 3", "user3@example.com"},
		{"User 4", "user4@example.com"},
	}

	// Queue insert operations
	// The email will be used as the shard key
	for _, user := range users {
		err := batch.Queue(
			user.Email, // Using email as shard key
			"INSERT INTO __bucket__.users (name, email) VALUES ($1, $2)",
			user.Name, user.Email,
		)
		if err != nil {
			log.Fatalf("Failed to queue insert: %v", err)
		}
	}

	// Execute all queued operations
	if err := batch.ExecAll(ctx); err != nil {
		log.Fatalf("Failed to execute batch: %v", err)
	}

	// Run a function across all buckets to count total users
	var totalUsers int64
	err := bucketDB.RunBucketFunc(ctx,
		func(ctx context.Context, shardID shard.ShardID, bucketID bucket.BucketID, con conn.IConnection) error {
			var count int
			err := pgxscan.Get(ctx, con,
				&count,
				"SELECT COUNT(*) FROM __bucket__.users",
			)
			if err != nil {
				return fmt.Errorf("failed to get count from bucket %d: %w", bucketID, err)
			}

			// Print count for this bucket
			fmt.Printf("Bucket %d on Shard %d has %d users\n", bucketID, shardID, count)
			totalUsers += int64(count)
			return nil
		})
	if err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}

	fmt.Printf("\nTotal users across all buckets: %d\n", totalUsers)

	// Example of reading user data using email as shard key
	email := "user1@example.com"
	var user struct {
		ID    int
		Name  string
		Email string
	}

	err = pgxscan.Get(ctx,
		bucketDB.Connection(ctx, email),
		&user,
		"SELECT id, name, email FROM __bucket__.users WHERE email = $1",
		email,
	)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("\nFound user: ID=%d, Name=%s, Email=%s\n", user.ID, user.Name, user.Email)
}
