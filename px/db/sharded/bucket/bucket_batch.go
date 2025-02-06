package bucket

import "github.com/jackc/pgx/v5"

// BucketBatch is an analog of pgx.Batch, but with bucket support.
// For correct work with buckets, you need to create a BucketBatch instance
// using the NewBucketBatch function, not pgx.BucketBatch.
// Then add queries to BucketBatch using the BucketBatch.Queue function,
// after which you can pass BucketBatch.PgxBatch() to SendBatch.
type BucketBatch struct {
	bucketID BucketID
	pgxBatch *pgx.Batch
}

// NewBucketBatch creates a new Batch.
func NewBucketBatch(bucketID BucketID) *BucketBatch {
	return &BucketBatch{
		bucketID: bucketID,
		pgxBatch: &pgx.Batch{},
	}
}

// BucketID returns the bucket identifier.
func (b *BucketBatch) BucketID() BucketID {
	return b.bucketID
}

// Queue adds a query to Batch, applying transformation of bucket aliases to table names.
func (b *BucketBatch) Queue(query string, args ...any) {
	b.pgxBatch.Queue(PrepareBucketSQL(query, b.bucketID), args...)
}

// QueueBucket adds a query to Batch. If bucketID doesn't match the current bucket, the operation is not performed.
func (b *BucketBatch) QueueBucket(bucketID BucketID, query string, args ...any) {
	if bucketID != b.bucketID {
		return
	}

	b.Queue(query, args...)
}

// Len returns the number of queries in Batch.
func (b *BucketBatch) Len() int {
	return b.pgxBatch.Len()
}

// PgxBatch returns the original pgx.Batch.
func (b *BucketBatch) PgxBatch() *pgx.Batch {
	return b.pgxBatch
}
