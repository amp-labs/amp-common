package contexts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	momValue = "momValue"
	dadValue = "dadValue"
)

// TestNewBatmanContext tests the creation and basic behavior of batman context.
func TestNewBatmanContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with two non-nil parents", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		assert.NotNil(t, batman)
	})

	t.Run("handles nil mom context", func(t *testing.T) {
		t.Parallel()

		dad := context.Background()
		batman := NewBatmanContext(nil, dad) //nolint:staticcheck // Testing nil context behavior

		assert.NotNil(t, batman)
	})

	t.Run("handles nil dad context", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		batman := NewBatmanContext(mom, nil) //nolint:staticcheck // Testing nil context behavior

		assert.NotNil(t, batman)
	})

	t.Run("handles both nil parents", func(t *testing.T) {
		t.Parallel()

		batman := NewBatmanContext(nil, nil) //nolint:staticcheck // Testing nil context behavior

		assert.NotNil(t, batman)
	})

	t.Run("implements context.Context interface", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		// Verify batman can be used as a context
		assert.NotNil(t, batman.Done())
		assert.NoError(t, batman.Err())
	})
}

// TestBatmanContextDone tests the Done() behavior when both parents complete.
func TestBatmanContextDone(t *testing.T) {
	t.Parallel()

	t.Run("done channel closes when both parents are done", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		dadCtx, cancelDad := context.WithCancel(t.Context())

		batman := NewBatmanContext(momCtx, dadCtx)

		// Batman should not be done yet
		select {
		case <-batman.Done():
			t.Fatal("batman should not be done when both parents are alive")
		default:
		}

		// Cancel mom
		cancelMom()

		// Batman should still not be done (dad is alive)
		select {
		case <-batman.Done():
			t.Fatal("batman should not be done when dad is still alive")
		case <-time.After(10 * time.Millisecond):
		}

		// Cancel dad
		cancelDad()

		// Now batman should be done
		select {
		case <-batman.Done():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("batman should be done when both parents are done")
		}
	})

	t.Run("done when mom dies first", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())

		dadCtx, cancelDad := context.WithCancel(t.Context())
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelMom()
		cancelDad()

		select {
		case <-batman.Done():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("batman should be done")
		}
	})

	t.Run("done when dad dies first", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		defer cancelMom()

		dadCtx, cancelDad := context.WithCancel(t.Context())

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelDad()
		cancelMom()

		select {
		case <-batman.Done():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("batman should be done")
		}
	})

	t.Run("done when both parents timeout", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancelMom()

		dadCtx, cancelDad := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		select {
		case <-batman.Done():
			// Success - should be done after both timeouts
		case <-time.After(100 * time.Millisecond):
			t.Fatal("batman should be done after both parents timeout")
		}
	})

	t.Run("not done when only mom is canceled", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())

		dadCtx, cancelDad := context.WithCancel(t.Context())
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelMom()

		select {
		case <-batman.Done():
			t.Fatal("batman should not be done when only mom is canceled")
		case <-time.After(50 * time.Millisecond):
			// Success - batman is not done
		}
	})

	t.Run("not done when only dad is canceled", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		defer cancelMom()

		dadCtx, cancelDad := context.WithCancel(t.Context())

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelDad()

		select {
		case <-batman.Done():
			t.Fatal("batman should not be done when only dad is canceled")
		case <-time.After(50 * time.Millisecond):
			// Success - batman is not done
		}
	})

	t.Run("already done parents", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		dadCtx, cancelDad := context.WithCancel(t.Context())

		// Cancel both before creating batman
		cancelMom()
		cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		// Batman should be done immediately
		select {
		case <-batman.Done():
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("batman should be done immediately when both parents are already done")
		}
	})
}

// TestBatmanContextDeadline tests the Deadline() method with various parent deadline combinations.
func TestBatmanContextDeadline(t *testing.T) {
	t.Parallel()

	t.Run("no deadline when neither parent has deadline", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)
		deadline, ok := batman.Deadline()

		assert.False(t, ok)
		assert.True(t, deadline.IsZero())
	})

	t.Run("returns mom deadline when only mom has deadline", func(t *testing.T) {
		t.Parallel()

		expectedDeadline := time.Now().Add(1 * time.Hour)

		momCtx, cancelMom := context.WithDeadline(t.Context(), expectedDeadline)
		defer cancelMom()

		dadCtx := context.Background()

		batman := NewBatmanContext(momCtx, dadCtx)
		deadline, ok := batman.Deadline()

		assert.True(t, ok)
		assert.Equal(t, expectedDeadline, deadline)
	})

	t.Run("returns dad deadline when only dad has deadline", func(t *testing.T) {
		t.Parallel()

		expectedDeadline := time.Now().Add(1 * time.Hour)
		momCtx := context.Background()

		dadCtx, cancelDad := context.WithDeadline(t.Context(), expectedDeadline)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)
		deadline, ok := batman.Deadline()

		assert.True(t, ok)
		assert.Equal(t, expectedDeadline, deadline)
	})

	t.Run("returns later deadline when both have deadlines", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		earlierDeadline := now.Add(1 * time.Hour)
		laterDeadline := now.Add(2 * time.Hour)

		momCtx, cancelMom := context.WithDeadline(t.Context(), earlierDeadline)
		defer cancelMom()

		dadCtx, cancelDad := context.WithDeadline(t.Context(), laterDeadline)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)
		deadline, ok := batman.Deadline()

		assert.True(t, ok)
		assert.Equal(t, laterDeadline, deadline)
	})

	t.Run("returns later deadline when dad has earlier deadline", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		earlierDeadline := now.Add(1 * time.Hour)
		laterDeadline := now.Add(2 * time.Hour)

		momCtx, cancelMom := context.WithDeadline(t.Context(), laterDeadline)
		defer cancelMom()

		dadCtx, cancelDad := context.WithDeadline(t.Context(), earlierDeadline)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)
		deadline, ok := batman.Deadline()

		assert.True(t, ok)
		assert.Equal(t, laterDeadline, deadline)
	})

	t.Run("returns same deadline when both have same deadline", func(t *testing.T) {
		t.Parallel()

		sameDeadline := time.Now().Add(1 * time.Hour)

		momCtx, cancelMom := context.WithDeadline(t.Context(), sameDeadline)
		defer cancelMom()

		dadCtx, cancelDad := context.WithDeadline(t.Context(), sameDeadline)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)
		deadline, ok := batman.Deadline()

		assert.True(t, ok)
		assert.Equal(t, sameDeadline, deadline)
	})
}

// TestBatmanContextErr tests the Err() method with various parent error states.
func TestBatmanContextErr(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when both parents have no error", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		assert.NoError(t, batman.Err())
	})

	t.Run("returns mom error when only mom is canceled", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		dadCtx := context.Background()

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelMom()

		// Wait for cancellation to propagate
		time.Sleep(10 * time.Millisecond)

		err := batman.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns dad error when only dad is canceled", func(t *testing.T) {
		t.Parallel()

		momCtx := context.Background()
		dadCtx, cancelDad := context.WithCancel(t.Context())

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelDad()

		// Wait for cancellation to propagate
		time.Sleep(10 * time.Millisecond)

		err := batman.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns joined error when both parents are canceled", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())
		dadCtx, cancelDad := context.WithCancel(t.Context())

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelMom()
		cancelDad()

		// Wait for cancellations to propagate
		time.Sleep(10 * time.Millisecond)

		err := batman.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("returns deadline exceeded when both parents timeout", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancelMom()

		dadCtx, cancelDad := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		// Wait for both to timeout
		time.Sleep(50 * time.Millisecond)

		err := batman.Err()
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("returns mixed errors when one is canceled and one times out", func(t *testing.T) {
		t.Parallel()

		momCtx, cancelMom := context.WithCancel(t.Context())

		dadCtx, cancelDad := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancelDad()

		batman := NewBatmanContext(momCtx, dadCtx)

		cancelMom()

		// Wait for dad to timeout
		time.Sleep(50 * time.Millisecond)

		err := batman.Err()
		require.Error(t, err)
		// Should contain both canceled and deadline exceeded
		require.ErrorIs(t, err, context.Canceled)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

// TestBatmanContextValue tests the Value() method with various parent value configurations.
func TestBatmanContextValue(t *testing.T) {
	t.Parallel()

	type contextKey string

	t.Run("returns nil when neither parent has value", func(t *testing.T) {
		t.Parallel()

		mom := context.Background()
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		assert.Nil(t, batman.Value(contextKey("key")))
	})

	t.Run("returns mom value when only mom has value", func(t *testing.T) {
		t.Parallel()

		key := contextKey("key")
		expectedValue := momValue

		mom := context.WithValue(t.Context(), key, expectedValue)
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		assert.Equal(t, expectedValue, batman.Value(key))
	})

	t.Run("returns dad value when only dad has value", func(t *testing.T) {
		t.Parallel()

		key := contextKey("key")
		expectedValue := dadValue

		mom := context.Background()
		dad := context.WithValue(t.Context(), key, expectedValue)

		batman := NewBatmanContext(mom, dad)

		assert.Equal(t, expectedValue, batman.Value(key))
	})

	t.Run("returns mom value when both have same key (mom has priority)", func(t *testing.T) {
		t.Parallel()

		key := contextKey("key")

		mom := context.WithValue(t.Context(), key, momValue)
		dad := context.WithValue(t.Context(), key, dadValue)

		batman := NewBatmanContext(mom, dad)

		// Mom has priority
		assert.Equal(t, momValue, batman.Value(key))
	})

	t.Run("returns different values for different keys", func(t *testing.T) {
		t.Parallel()

		key1 := contextKey("key1")
		key2 := contextKey("key2")
		momValue := "momValue"
		dadValue := "dadValue"

		mom := context.WithValue(t.Context(), key1, momValue)
		dad := context.WithValue(t.Context(), key2, dadValue)

		batman := NewBatmanContext(mom, dad)

		assert.Equal(t, momValue, batman.Value(key1))
		assert.Equal(t, dadValue, batman.Value(key2))
	})

	t.Run("handles various value types", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name string
		}

		intKey := contextKey("int")
		stringKey := contextKey("string")
		structKey := contextKey("struct")
		ptrKey := contextKey("ptr")

		intValue := 42
		stringValue := "test"
		structValue := testStruct{Name: "test"}
		ptrValue := &testStruct{Name: "pointer"}

		mom := context.WithValue(t.Context(), intKey, intValue)
		mom = context.WithValue(mom, stringKey, stringValue)
		dad := context.WithValue(t.Context(), structKey, structValue)
		dad = context.WithValue(dad, ptrKey, ptrValue)

		batman := NewBatmanContext(mom, dad)

		assert.Equal(t, intValue, batman.Value(intKey))
		assert.Equal(t, stringValue, batman.Value(stringKey))
		assert.Equal(t, structValue, batman.Value(structKey))
		assert.Equal(t, ptrValue, batman.Value(ptrKey))
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		t.Parallel()

		key := contextKey("key")
		value := "value"

		mom := context.WithValue(t.Context(), key, value)
		dad := context.Background()

		batman := NewBatmanContext(mom, dad)

		assert.Nil(t, batman.Value(contextKey("nonexistent")))
	})
}

// TestBatmanContextIntegration tests real-world scenarios with batman context.
func TestBatmanContextIntegration(t *testing.T) {
	t.Parallel()

	t.Run("coordinated shutdown scenario", func(t *testing.T) {
		t.Parallel()

		httpCtx, stopHTTP := context.WithCancel(t.Context())
		grpcCtx, stopGRPC := context.WithCancel(t.Context())

		batman := NewBatmanContext(httpCtx, grpcCtx)

		// Simulate HTTP server shutting down first
		stopHTTP()

		// Batman should not be done yet
		select {
		case <-batman.Done():
			t.Fatal("should not be done until both servers stop")
		case <-time.After(10 * time.Millisecond):
		}

		// Now gRPC server shuts down
		stopGRPC()

		// Now batman should be done
		select {
		case <-batman.Done():
			// Success - both servers stopped
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be done when both servers stop")
		}
	})

	t.Run("fan-in operation completion", func(t *testing.T) {
		t.Parallel()

		op1Ctx, finishOp1 := context.WithCancel(t.Context())
		op2Ctx, finishOp2 := context.WithCancel(t.Context())

		batman := NewBatmanContext(op1Ctx, op2Ctx)

		// Simulate operations completing in any order
		done := make(chan struct{})

		go func() {
			<-batman.Done()
			close(done)
		}()

		// Complete operations
		finishOp1()
		finishOp2()

		// Wait for batman to be done
		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("should be done when both operations complete")
		}
	})

	t.Run("preserves values from both parents", func(t *testing.T) {
		t.Parallel()

		type contextKey string

		userKey := contextKey("user")
		sessionKey := contextKey("session")

		mom := context.WithValue(t.Context(), userKey, "alice")
		dad := context.WithValue(t.Context(), sessionKey, "session123")

		batman := NewBatmanContext(mom, dad)

		assert.Equal(t, "alice", batman.Value(userKey))
		assert.Equal(t, "session123", batman.Value(sessionKey))
	})

	t.Run("deadline reflects both parent timeouts", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		shortTimeout := now.Add(100 * time.Millisecond)
		longTimeout := now.Add(200 * time.Millisecond)

		shortCtx, cancelShort := context.WithDeadline(t.Context(), shortTimeout)
		defer cancelShort()

		longCtx, cancelLong := context.WithDeadline(t.Context(), longTimeout)
		defer cancelLong()

		batman := NewBatmanContext(shortCtx, longCtx)

		deadline, ok := batman.Deadline()
		require.True(t, ok)
		// Deadline should be the later one (longTimeout)
		assert.Equal(t, longTimeout, deadline)
	})
}
