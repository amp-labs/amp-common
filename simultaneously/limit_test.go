package simultaneously

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/should"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLimitJobFailed = errors.New("limited job failed")

func TestResolveExecutor_AmbientWithoutEffectiveCapIsUnwrapped(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(4)
	defer should.Close(exec, "closing executor")

	ctx := WithExecutor(t.Context(), exec)

	// No cap of its own, or a cap the call could never reach: nothing to enforce,
	// so the shared executor is handed back as-is.
	for _, maxConcurrent := range []int{-1, 0, 3, 10} {
		got, closeExec := resolveExecutor(ctx, maxConcurrent, 3)

		assert.Same(t, exec, got, "maxConcurrent=%d", maxConcurrent)
		require.NoError(t, closeExec())
	}
}

func TestResolveExecutor_AmbientWithEffectiveCapIsWrapped(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(4)
	defer should.Close(exec, "closing executor")

	got, closeExec := resolveExecutor(WithExecutor(t.Context(), exec), 2, 3)

	limited, ok := got.(*limitedExecutor)
	require.True(t, ok, "expected a per-call limiter, got %T", got)
	assert.Same(t, exec, limited.inner)
	assert.Equal(t, 2, cap(limited.sem))
	require.NoError(t, closeExec())
}

func TestDoCtx_AmbientExecutorHonorsSmallerMaxConcurrent(t *testing.T) {
	t.Parallel()

	// The pool has plenty of room; the call's own cap of 2 is what must hold.
	rec := newRecordingExecutor(10)
	defer should.Close(rec, "closing recording executor")

	tracker := &peakTracker{}

	err := DoCtx(WithExecutor(t.Context(), rec), 2,
		tracker.job(), tracker.job(), tracker.job(),
		tracker.job(), tracker.job(), tracker.job(),
	)

	require.NoError(t, err)
	assert.LessOrEqual(t, tracker.observed(), 2, "the call's own cap must hold on a shared executor")
	assert.Equal(t, int32(6), rec.goCalls.Load(), "every job still runs on the ambient executor")
	assert.Equal(t, int32(0), rec.closeCalls.Load(), "ownership stays with the caller that attached it")
}

func TestDoCtx_AmbientExecutorUncappedUsesWholePool(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	defer should.Close(exec, "closing executor")

	tracker := &peakTracker{}

	// maxConcurrent < 1 means no cap of its own, so the jobs may overlap up to the
	// pool's limit.
	err := DoCtx(WithExecutor(t.Context(), exec), 0, tracker.job(), tracker.job(), tracker.job())

	require.NoError(t, err)
	assert.Greater(t, tracker.observed(), 1)
	assert.LessOrEqual(t, tracker.observed(), 3)
}

func TestMapFamily_AmbientExecutorHonorsSmallerMaxConcurrent(t *testing.T) {
	t.Parallel()

	for _, tc := range mapFamilyCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Room for all three elements in the pool, but the call asks for one at
			// a time, so nothing may overlap.
			rec := newRecordingExecutor(3)
			defer should.Close(rec, "closing recording executor")

			tracker := &peakTracker{}

			err := tc.run(WithExecutor(t.Context(), rec), 1, func() {
				tracker.enter()
				defer tracker.exit()

				time.Sleep(20 * time.Millisecond)
			})

			require.NoError(t, err)
			assert.Equal(t, 1, tracker.observed(), "the call's own cap must hold on a shared executor")
			assert.Equal(t, int32(3), rec.goCalls.Load(), "each element dispatches on the ambient executor")
		})
	}
}

func TestDoCtx_CappedCallDoesNotHoldSharedSlotsWhileWaiting(t *testing.T) {
	t.Parallel()

	// Two shared slots. Call A is capped at 1 and has several jobs queued behind
	// its first one, which blocks until call B has run. If A's queued jobs held
	// shared slots while waiting on A's own cap, they would take the second slot,
	// B could never start, and A's first job would never be released.
	//
	// Both calls run under a deadline so that a regression fails with a context
	// error instead of hanging the suite: the deadline unblocks A's first job,
	// which drains everything and lets the executor close.
	exec := NewDefaultExecutor(2)
	defer should.Close(exec, "closing executor")

	deadline, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	ctx := WithExecutor(deadline, exec)

	bRan := make(chan struct{})
	aStarted := make(chan struct{})

	blockUntilB := func(ctx context.Context) error {
		close(aStarted)

		select {
		case <-bRan:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	noop := func(ctx context.Context) error { return nil }

	aDone := make(chan error, 1)

	go func() {
		aDone <- DoCtx(ctx, 1, blockUntilB, noop, noop, noop)
	}()

	// Let A's first job start and give A's launcher time to submit the rest, so
	// that if they were going to grab shared slots while waiting, they already
	// have by the time B asks for one.
	<-aStarted
	time.Sleep(50 * time.Millisecond)

	err := DoCtx(ctx, 0, func(ctx context.Context) error {
		close(bRan)

		return nil
	})
	require.NoError(t, err, "B must get a shared slot while A waits on its own cap")
	require.NoError(t, <-aDone)
}

func TestDoCtx_AmbientExecutorWithCapPropagatesErrorAndCancels(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(10)
	defer should.Close(exec, "closing executor")

	var ranAfterFailure atomic.Int32

	late := func(ctx context.Context) error {
		if ctx.Err() == nil {
			ranAfterFailure.Add(1)
		}

		return nil
	}

	// Capped at 1, so the failing first job finishes before any other starts, and
	// the rest must see a canceled context.
	err := DoCtx(WithExecutor(t.Context(), exec), 1,
		func(ctx context.Context) error { return errLimitJobFailed },
		late, late, late,
	)

	require.ErrorIs(t, err, errLimitJobFailed)
	assert.Equal(t, int32(0), ranAfterFailure.Load(), "jobs after the failure must see a canceled context")
}

func TestLimitedExecutor_CanceledWhileWaitingReportsContextError(t *testing.T) {
	t.Parallel()

	inner := NewDefaultExecutor(4)
	defer should.Close(inner, "closing executor")

	lim := newLimitedExecutor(inner, 1)

	release := make(chan struct{})
	firstDone := make(chan error, 1)

	// Occupy the only local slot.
	lim.GoContext(t.Context(), func(ctx context.Context) error {
		<-release

		return nil
	}, func(err error) { firstDone <- err })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	secondDone := make(chan error, 1)

	lim.GoContext(ctx, func(ctx context.Context) error {
		t.Error("callback must not run once its context is canceled")

		return nil
	}, func(err error) { secondDone <- err })

	require.ErrorIs(t, <-secondDone, context.Canceled)

	close(release)
	require.NoError(t, <-firstDone)
}

func TestLimitedExecutor_InnerPanicReleasesSlot(t *testing.T) {
	t.Parallel()

	for name, inner := range map[string]Executor{
		"panics without reporting": panickingExecutor{},
		"reports then panics":      lateResultExecutor{},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			lim := newLimitedExecutor(inner, 1)

			// With a single slot, a leaked token would block the second submission
			// forever.
			for range 2 {
				err := withinTimeout(t, 5*time.Second, func() (err error) {
					defer func() {
						if r := recover(); r == nil {
							err = errors.New("expected the inner panic to propagate") //nolint:err113
						}
					}()

					lim.GoContext(t.Context(), func(ctx context.Context) error { return nil }, func(error) {})

					return nil
				})
				require.NoError(t, err)
			}
		})
	}
}

func TestLimitedExecutor_CloseDoesNotCloseInner(t *testing.T) {
	t.Parallel()

	rec := newRecordingExecutor(1)
	defer should.Close(rec, "closing recording executor")

	require.NoError(t, newLimitedExecutor(rec, 1).Close())
	assert.Equal(t, int32(0), rec.closeCalls.Load())
}
