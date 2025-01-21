package px

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestBeginFunc(t *testing.T) {
	t.Parallel()

	t.Run("successful transaction", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := NewMockTx(ctrl)
		mockConn := NewMockITransactionBeginner(ctrl)

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(mockTx, nil)
		mockTx.EXPECT().Commit(ctx).Return(nil)
		// Account for deferred Rollback that happens even in successful case
		mockTx.EXPECT().Rollback(ctx).Return(pgx.ErrTxClosed)

		executed := false
		err := BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
			executed = true
			return nil
		})

		require.NoError(t, err)
		require.True(t, executed, "transaction function should have been executed")
	})

	t.Run("begin transaction error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockConn := NewMockITransactionBeginner(ctrl)
		expectedErr := errors.New("begin error")

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(nil, expectedErr)

		executed := false
		err := BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
			executed = true
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "begin transaction")
		require.False(t, executed, "transaction function should not have been executed")
	})

	t.Run("execution error with successful rollback", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := NewMockTx(ctrl)
		mockConn := NewMockITransactionBeginner(ctrl)
		expectedErr := errors.New("execution error")

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(mockTx, nil)
		// Expect two Rollback calls - one explicit and one from defer
		mockTx.EXPECT().Rollback(ctx).Return(nil)
		mockTx.EXPECT().Rollback(ctx).Return(pgx.ErrTxClosed)

		err := BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
			return expectedErr
		})

		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("execution error with rollback error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := NewMockTx(ctrl)
		mockConn := NewMockITransactionBeginner(ctrl)
		execErr := errors.New("execution error")
		rollbackErr := errors.New("rollback error")

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(mockTx, nil)
		// Expect two Rollback calls - one explicit and one from defer
		mockTx.EXPECT().Rollback(ctx).Return(rollbackErr)
		mockTx.EXPECT().Rollback(ctx).Return(rollbackErr)

		err := BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
			return execErr
		})

		require.Error(t, err)
		// The implementation just returns the rollback error directly without wrapping it
		require.Equal(t, "execution error (rollback error: rollback error)", err.Error())
	})

	t.Run("commit error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := NewMockTx(ctrl)
		mockConn := NewMockITransactionBeginner(ctrl)
		commitErr := errors.New("commit error")

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(mockTx, nil)
		mockTx.EXPECT().Commit(ctx).Return(commitErr)
		// Account for deferred Rollback after commit failure
		mockTx.EXPECT().Rollback(ctx).Return(nil)

		err := BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
			return nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "commit transaction")
	})

	t.Run("panic handling", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTx := NewMockTx(ctrl)
		mockConn := NewMockITransactionBeginner(ctrl)

		ctx := context.Background()
		mockConn.EXPECT().BeginTx(ctx, pgx.TxOptions{}).Return(mockTx, nil)
		// Expect two Rollback calls - one from panic recovery and one from defer
		mockTx.EXPECT().Rollback(ctx).Return(nil)
		mockTx.EXPECT().Rollback(ctx).Return(pgx.ErrTxClosed)

		require.Panics(t, func() {
			_ = BeginFunc(ctx, mockConn, func(ctx context.Context, tx pgx.Tx) error {
				panic("test panic")
			})
		})
	})
}
