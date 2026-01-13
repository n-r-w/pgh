package bucket

import "github.com/jackc/pgx/v5"

// Batch is an analog of pgx.Batch, but with bucket support.
// For correct work with buckets, you need to create a Batch instance
// using the NewBatch function, not pgx.Batch.
// Then add queries to Batch using the Batch.Queue function,
// after which you can pass Batch.PgxBatch() to SendBatch.
type Batch struct {
	bucketID BucketID
	pgxBatch *pgx.Batch
}

// NewBatch creates a new Batch.
func NewBatch(bucketID BucketID) *Batch {
	return &Batch{
		bucketID: bucketID,
		pgxBatch: &pgx.Batch{
			QueuedQueries: nil,
		},
	}
}

// BucketID returns the bucket identifier.
func (b *Batch) BucketID() BucketID {
	return b.bucketID
}

// Queue adds a query to Batch, applying transformation of bucket aliases to table names.
func (b *Batch) Queue(query string, args ...any) {
	b.pgxBatch.Queue(PrepareBucketSQL(query, b.bucketID), args...)
}

// QueueBucket adds a query to Batch. If bucketID doesn't match the current bucket, the operation is not performed.
func (b *Batch) QueueBucket(bucketID BucketID, query string, args ...any) {
	if bucketID != b.bucketID {
		return
	}

	b.Queue(query, args...)
}

// Len returns the number of queries in Batch.
func (b *Batch) Len() int {
	return b.pgxBatch.Len()
}

// PgxBatch returns the original pgx.Batch.
func (b *Batch) PgxBatch() *pgx.Batch {
	return b.pgxBatch
}
