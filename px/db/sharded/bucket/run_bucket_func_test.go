package bucket

import (
	"context"
	"testing"
	"time"

	"github.com/n-r-w/pgh/v2/px/db"
	"github.com/n-r-w/pgh/v2/px/db/conn"
	"github.com/n-r-w/pgh/v2/px/db/sharded/shard"
	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRunBucketFunc_Deadlock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockConnector1 := db.NewMockIStartStopConnector(ctrl)
	mockConnector2 := db.NewMockIStartStopConnector(ctrl)

	mockConnection := conn.NewMockIConnection(ctrl)
	mockConnector1.EXPECT().Connection(gomock.Any()).Return(mockConnection).AnyTimes()
	mockConnector2.EXPECT().Connection(gomock.Any()).Return(mockConnection).AnyTimes()

	shard1 := shard.ShardID(1)
	shard2 := shard.ShardID(2)

	shardDB := shard.New([]*shard.ShardInfo{
		{
			ShardID:    shard1,
			Connector:  mockConnector1,
			TxBeginner: txmgr.NewMockITransactionBeginner(ctrl),
			TxInformer: txmgr.NewMockITransactionInformer(ctrl),
		},
		{
			ShardID:    shard2,
			Connector:  mockConnector2,
			TxBeginner: txmgr.NewMockITransactionBeginner(ctrl),
			TxInformer: txmgr.NewMockITransactionInformer(ctrl),
		},
	}, shard.DefaultShardFunc)

	bucketDB := New(
		shardDB,
		[]*BucketInfo{
			{ShardID: shard1, BucketRange: NewBucketRange(0, 0)},
			{ShardID: shard2, BucketRange: NewBucketRange(1, 1)},
		},
		UniformBucketFn(2),
		WithRunLimit[string](2),
	)

	done := make(chan error, 1)
	go func() {
		done <- bucketDB.RunBucketFunc(context.Background(),
			func(context.Context, shard.ShardID, BucketID, conn.IConnection) error {
				return nil
			},
		)
	}()

	const timeout = 500 * time.Millisecond
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(timeout):
		t.Fatalf("RunBucketFunc deadlocked: did not complete within %v.", timeout)
	}
}
