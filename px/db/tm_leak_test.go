package db

import (
	"context"
	"testing"
	"time"

	"github.com/n-r-w/pgh/v2/txmgr"
	"github.com/n-r-w/testdock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBeginTxHelper_ConnectionLeak_Simulation verifies that beginTxHelper
// properly releases acquired connections when BeginTx fails.
func TestBeginTxHelper_ConnectionLeak_Simulation(t *testing.T) {
	t.Parallel()

	_, informer := testdock.GetPgxPool(t, testdock.DefaultPostgresDSN)

	pxDB := New(
		WithName("leak-simulation"),
		WithDSN(informer.DSN()),
	)

	ctx := context.Background()

	ctxStart, cancelStart := context.WithTimeout(ctx, 5*time.Second)
	t.Cleanup(cancelStart)
	require.NoError(t, pxDB.Start(ctxStart))
	t.Cleanup(func() {
		ctxStop, cancelStop := context.WithTimeout(ctx, 2*time.Second)
		defer cancelStop()
		_ = pxDB.Stop(ctxStop)
	})

	internalPool := pxDB.pool
	require.NotNil(t, internalPool)

	// Baseline
	statsBefore := internalPool.Stat()
	acquiredBefore := statsBefore.AcquiredConns()
	t.Logf("Connections before: acquired=%d, total=%d", acquiredBefore, statsBefore.TotalConns())

	// Create a context that will be canceled by the hook AFTER Acquire succeeds.
	// This ensures Acquire returns a valid connection, but BeginTx fails.
	ctxBeginTx, cancelBeginTx := context.WithCancel(ctx)
	pxDB.testHookAfterAcquire = func() {
		cancelBeginTx()
	}
	t.Cleanup(func() {
		pxDB.testHookAfterAcquire = nil
	})

	_, _, errBegin := pxDB.beginTxHelper(ctxBeginTx, txmgr.Options{}) //nolint:exhaustruct // testing zero-value options
	require.Error(t, errBegin, "beginTxHelper should fail when BeginTx fails")
	require.ErrorIs(t, errBegin, context.Canceled, "error should wrap context.Canceled")
	t.Logf("beginTxHelper failed as expected: %v", errBegin)

	// Connection should be released back to pool (not leaked).
	statsAfter := internalPool.Stat()
	acquiredAfter := statsAfter.AcquiredConns()
	t.Logf("Connections after beginTxHelper failure: acquired=%d", acquiredAfter)

	assert.Equal(t, acquiredBefore, acquiredAfter,
		"Connection leak: acquired connection count should remain unchanged after BeginTx failure")
}
