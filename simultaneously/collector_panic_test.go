package simultaneously

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errLateResult = errors.New("late result")

// panickingExecutor panics out of GoContext instead of dispatching, standing in
// for a buggy custom Executor. Before launch() recovered, this stranded the
// collector's wait group and hung the call forever.
type panickingExecutor struct{}

func (panickingExecutor) GoContext(ctx context.Context, fn func(context.Context) error, done func(error)) {
	panic("executor exploded")
}

func (p panickingExecutor) Go(fn func(context.Context) error, done func(error)) {
	p.GoContext(context.Background(), fn, done)
}

func (panickingExecutor) Close() error { return nil }

// lateResultExecutor reports a result and then panics, exercising the
// exactly-once guard: the first result must win and the panic must not add a
// second one.
type lateResultExecutor struct{}

func (lateResultExecutor) GoContext(ctx context.Context, fn func(context.Context) error, done func(error)) {
	done(errLateResult)

	panic("executor exploded after reporting")
}

func (l lateResultExecutor) Go(fn func(context.Context) error, done func(error)) {
	l.GoContext(context.Background(), fn, done)
}

func (lateResultExecutor) Close() error { return nil }

// withinTimeout runs fn and fails the test if it does not finish in time, so a
// regression shows up as a clear failure rather than a hung suite.
func withinTimeout(t *testing.T, limit time.Duration, fn func() error) error {
	t.Helper()

	result := make(chan error, 1)

	go func() {
		result <- fn()
	}()

	select {
	case err := <-result:
		return err
	case <-time.After(limit):
		t.Fatal("call never returned; the collector stranded its wait group")

		return nil
	}
}

func TestDoCtxWithExecutor_ExecutorPanicDoesNotHang(t *testing.T) {
	t.Parallel()

	err := withinTimeout(t, 10*time.Second, func() error {
		return DoCtxWithExecutor(t.Context(), panickingExecutor{},
			func(ctx context.Context) error { return nil },
			func(ctx context.Context) error { return nil },
		)
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "executor exploded")
}

func TestDoCtxWithExecutor_ExecutorPanicAfterResultCountsOnce(t *testing.T) {
	t.Parallel()

	err := withinTimeout(t, 10*time.Second, func() error {
		return DoCtxWithExecutor(t.Context(), lateResultExecutor{},
			func(ctx context.Context) error { return nil },
		)
	})

	// The reported result wins; the trailing panic is dropped rather than
	// becoming a second result for the same callback.
	require.ErrorIs(t, err, errLateResult)
	assert.NotContains(t, err.Error(), "recovered from panic")
}

func TestDoCtxWithExecutor_NilExecutorDoesNotHang(t *testing.T) {
	t.Parallel()

	// A typed nil is filtered out of the context by GetExecutor, but nothing
	// stops a caller passing one here directly. It must fail, not hang.
	var typedNil *defaultExecutor

	err := withinTimeout(t, 10*time.Second, func() error {
		return DoCtxWithExecutor(t.Context(), typedNil,
			func(ctx context.Context) error { return nil },
		)
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
}
