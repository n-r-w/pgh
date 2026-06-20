package txmgr

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type transactionMarkerKey struct{}

// TestAfterCommit_OutsideTransaction tests that commit hook registration fails without a txmgr-managed transaction.
func TestAfterCommit_OutsideTransaction(t *testing.T) {
	t.Parallel()

	called := false

	err := AfterCommit(t.Context(), func(context.Context) {
		called = true
	})

	require.ErrorIs(t, err, ErrNoTransactionHooks)
	require.False(t, called)
}

// TestAfterRollback_OutsideTransaction tests that rollback hook registration fails without a txmgr-managed transaction.
func TestAfterRollback_OutsideTransaction(t *testing.T) {
	t.Parallel()

	called := false

	err := AfterRollback(t.Context(), func(context.Context) {
		called = true
	})

	require.ErrorIs(t, err, ErrNoTransactionHooks)
	require.False(t, called)
}

// TestAfterHooks_Nil tests that nil hooks are ignored.
func TestAfterHooks_Nil(t *testing.T) {
	t.Parallel()

	require.NoError(t, AfterCommit(t.Context(), nil))
	require.NoError(t, AfterRollback(t.Context(), nil))
}

// TestTransactionManager_Begin_CommitsAndRunsAfterCommit tests commit hook execution in callback mode.
func TestTransactionManager_Begin_CommitsAndRunsAfterCommit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	hookCtx := context.WithValue(txCtx, transactionMarkerKey{}, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	finisher := NewMockITransactionFinisher(mc)
	finisher.EXPECT().Commit(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return nil
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().
		BeginTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, opts Options) (context.Context, ITransactionFinisher, error) {
			require.Equal(t, TxReadCommitted, opts.Level)
			require.Equal(t, TxReadWrite, opts.Mode)
			require.True(t, opts.Lock)
			return txCtx, finisher, nil
		})
	tmBeginner.EXPECT().WithoutTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) context.Context {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return hookCtx
	})

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false).Times(2)

	tm := New(tmBeginner, tmInformer)
	order := make([]string, 0, 2)

	err := tm.Begin(ctx, func(ctxTx context.Context) error {
		order = append(order, "callback")
		require.NoError(t, AfterCommit(ctxTx, func(ctx context.Context) {
			require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
			require.ErrorIs(t, AfterCommit(ctx, func(context.Context) {}), ErrNoTransactionHooks)
			order = append(order, "after_commit")
		}))
		return nil
	}, WithTransactionLevel(TxReadCommitted), WithTransactionMode(TxReadWrite), WithLock())

	require.NoError(t, err)
	require.Equal(t, []string{"callback", "after_commit"}, order)
}

// TestTransactionManager_Begin_RollsBackAndRunsAfterRollback tests rollback hook execution in callback mode.
func TestTransactionManager_Begin_RollsBackAndRunsAfterRollback(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	hookCtx := context.WithValue(txCtx, transactionMarkerKey{}, false)
	callbackErr := errors.New("callback failed")

	mc := gomock.NewController(t)
	defer mc.Finish()

	finisher := NewMockITransactionFinisher(mc)
	finisher.EXPECT().Rollback(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return nil
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(txCtx, finisher, nil)
	tmBeginner.EXPECT().WithoutTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) context.Context {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return hookCtx
	})

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false).Times(2)

	tm := New(tmBeginner, tmInformer)
	order := make([]string, 0, 2)
	commitCalled := false

	err := tm.Begin(ctx, func(ctxTx context.Context) error {
		order = append(order, "callback")
		require.NoError(t, AfterCommit(ctxTx, func(context.Context) {
			commitCalled = true
		}))
		require.NoError(t, AfterRollback(ctxTx, func(ctx context.Context) {
			require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
			order = append(order, "after_rollback")
		}))
		return callbackErr
	})

	require.ErrorIs(t, err, callbackErr)
	require.False(t, commitCalled)
	require.Equal(t, []string{"callback", "after_rollback"}, order)
}

// TestTransactionManager_BeginTx_CommitRunsAfterCommit tests commit hook execution in manual mode.
func TestTransactionManager_BeginTx_CommitRunsAfterCommit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	hookCtx := context.WithValue(txCtx, transactionMarkerKey{}, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	baseFinisher := NewMockITransactionFinisher(mc)
	baseFinisher.EXPECT().Commit(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return nil
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(txCtx, baseFinisher, nil)
	tmBeginner.EXPECT().WithoutTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) context.Context {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return hookCtx
	})

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false).Times(2)

	tm := New(tmBeginner, tmInformer)
	ctxTx, finisher, err := tm.BeginTx(ctx)
	require.NoError(t, err)

	called := false
	require.NoError(t, AfterCommit(ctxTx, func(ctx context.Context) {
		require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
		called = true
	}))

	require.NoError(t, finisher.Commit(ctxTx))
	require.True(t, called)
}

// TestTransactionManager_BeginTx_RollbackRunsAfterRollback tests rollback hook execution in manual mode.
func TestTransactionManager_BeginTx_RollbackRunsAfterRollback(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	hookCtx := context.WithValue(txCtx, transactionMarkerKey{}, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	baseFinisher := NewMockITransactionFinisher(mc)
	baseFinisher.EXPECT().Rollback(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return nil
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(txCtx, baseFinisher, nil)
	tmBeginner.EXPECT().WithoutTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) context.Context {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return hookCtx
	})

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false).Times(2)

	tm := New(tmBeginner, tmInformer)
	ctxTx, finisher, err := tm.BeginTx(ctx)
	require.NoError(t, err)

	called := false
	require.NoError(t, AfterRollback(ctxTx, func(ctx context.Context) {
		require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
		called = true
	}))

	require.NoError(t, finisher.Rollback(ctxTx))
	require.True(t, called)
}

// TestTransactionManager_Begin_NestedHooksRunAfterOuterCommit tests that nested hooks wait for the outer transaction.
func TestTransactionManager_Begin_NestedHooksRunAfterOuterCommit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	hookCtx := context.WithValue(txCtx, transactionMarkerKey{}, false)

	mc := gomock.NewController(t)
	defer mc.Finish()

	baseFinisher := NewMockITransactionFinisher(mc)
	baseFinisher.EXPECT().Commit(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return nil
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(txCtx, baseFinisher, nil).Times(1)
	tmBeginner.EXPECT().WithoutTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) context.Context {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return hookCtx
	}).Times(1)

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).DoAndReturn(func(ctx context.Context) bool {
		inTransaction, _ := ctx.Value(transactionMarkerKey{}).(bool)
		return inTransaction
	}).AnyTimes()
	tmInformer.EXPECT().TransactionOptions(gomock.Any()).Return(Options{}).AnyTimes()

	tm := New(tmBeginner, tmInformer)
	order := make([]string, 0, 2)

	err := tm.Begin(ctx, func(ctxOuter context.Context) error {
		require.NoError(t, AfterCommit(ctxOuter, func(ctx context.Context) {
			require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
			order = append(order, "outer")
		}))

		err := tm.Begin(ctxOuter, func(ctxInner context.Context) error {
			require.Equal(t, ctxOuter, ctxInner)
			require.NoError(t, AfterCommit(ctxInner, func(ctx context.Context) {
				require.False(t, ctx.Value(transactionMarkerKey{}).(bool))
				order = append(order, "inner")
			}))
			return nil
		})
		require.NoError(t, err)
		require.Empty(t, order)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, []string{"outer", "inner"}, order)
}

// TestTransactionManager_Begin_InTransactionOptions tests option checks for an existing transaction.
func TestTransactionManager_Begin_InTransactionOptions(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(t.Context(), transactionMarkerKey{}, true)

	mc := gomock.NewController(t)
	defer mc.Finish()

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().Begin(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(0)

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(true).AnyTimes()
	tmInformer.EXPECT().TransactionOptions(gomock.Any()).Return(Options{}).AnyTimes()

	tm := New(tmBeginner, tmInformer)
	called := false

	require.NoError(t, tm.Begin(ctx, func(ctxTx context.Context) error {
		require.Equal(t, ctx, ctxTx)
		called = true
		return nil
	}))
	require.True(t, called)

	require.Error(t, tm.Begin(ctx, func(context.Context) error {
		return nil
	}, WithTransactionLevel(TxReadUncommitted)))

	_, _, err := tm.BeginTx(ctx, WithTransactionMode(TxReadOnly))
	require.Error(t, err)
}

// TestTransactionManager_Begin_CommitErrorSkipsHooks tests that commit hooks require a successful commit.
func TestTransactionManager_Begin_CommitErrorSkipsHooks(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	txCtx := context.WithValue(ctx, transactionMarkerKey{}, true)
	commitErr := errors.New("commit failed")

	mc := gomock.NewController(t)
	defer mc.Finish()

	finisher := NewMockITransactionFinisher(mc)
	finisher.EXPECT().Commit(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
		require.True(t, ctx.Value(transactionMarkerKey{}).(bool))
		return commitErr
	})

	tmBeginner := NewMockITransactionBeginner(mc)
	tmBeginner.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(txCtx, finisher, nil)

	tmInformer := NewMockITransactionInformer(mc)
	tmInformer.EXPECT().InTransaction(gomock.Any()).Return(false).Times(2)

	tm := New(tmBeginner, tmInformer)
	called := false

	err := tm.Begin(ctx, func(ctxTx context.Context) error {
		require.NoError(t, AfterCommit(ctxTx, func(context.Context) {
			called = true
		}))
		return nil
	})

	require.ErrorIs(t, err, commitErr)
	require.False(t, called)
}
