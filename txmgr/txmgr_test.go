package txmgr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestTransactionManager_Begin tests transaction start.
func TestTransactionManager_Begin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	// Transaction is not started, so Begin should be called.
	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().
		Begin(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, f func(context.Context) error, opts Options) error {
			require.Equal(t, TxReadCommitted, opts.Level)
			require.Equal(t, TxReadWrite, opts.Mode)
			require.True(t, opts.Lock)
			return f(ctx)
		}).
		Return(nil)

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false)

	tm := New(tmBeginner, tmInformer)

	require.NoError(t, tm.Begin(ctx, func(_ context.Context) error {
		return nil
	}, WithTransactionLevel(TxReadCommitted), WithTransactionMode(TxReadWrite), WithLock()))
}

// TestTransactionManager_Begin_InTransaction tests transaction start when a transaction is already in progress.
func TestTransactionManager_Begin_InTransaction(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	// Transaction is already started (InTransaction returns true), so Begin should not be called.
	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().Begin(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(true).Times(3)
	tmInformer.EXPECT().TransactionOptions(gomock.Any()).Return(Options{}).Times(3)

	tm := New(tmBeginner, tmInformer)

	// starting transaction
	require.NoError(t, tm.Begin(ctx, func(_ context.Context) error {
		return nil
	}))

	// error when changing isolation level
	require.Error(t, tm.Begin(ctx, func(_ context.Context) error {
		return nil
	}, WithTransactionLevel(TxReadUncommitted)))

	// error when changing transaction mode
	require.Error(t, tm.Begin(ctx, func(_ context.Context) error {
		return nil
	}, WithTransactionMode(TxReadOnly)))
}
