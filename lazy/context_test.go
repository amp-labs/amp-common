package lazy

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testValue         = "test"
	originalValue     = "original"
	fromProviderValue = "from-provider"
)

func TestWithTestLocalCtx(t *testing.T) {
	t.Parallel()

	t.Run("creates independent test instance", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		globalLazy := NewCtx(func(ctx context.Context) string {
			callCount.Add(1)

			return "global-value"
		})

		// Enable testing mode to preserve create function
		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize global lazy
		result := globalLazy.Get(ctx)
		assert.Equal(t, "global-value", result)
		assert.Equal(t, int32(1), callCount.Load())

		// Create test-local instance
		key, getter := WithTestLocalCtx(globalLazy)
		assert.NotEmpty(t, key)

		// Test-local instance should have its own state
		localResult := getter(ctx)
		assert.Equal(t, "global-value", localResult)
		assert.Equal(t, int32(2), callCount.Load(), "test-local instance should initialize independently")

		// Call again - should not re-initialize
		localResult = getter(ctx)
		assert.Equal(t, "global-value", localResult)
		assert.Equal(t, int32(2), callCount.Load(), "test-local instance should be memoized")
	})

	t.Run("assigns name to lazy value if unnamed", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) int { return 42 })
		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize so create function is preserved
		lazy.Get(ctx)

		assert.Empty(t, lazy.name, "should start with no name")

		key, _ := WithTestLocalCtx(lazy)
		assert.NotEmpty(t, key, "should generate a key")
		assert.NotEmpty(t, lazy.name, "should assign name to lazy value")
		assert.Equal(t, string(lazy.name), key, "key should match assigned name")
	})

	t.Run("reuses existing name", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) int { return 42 }).WithName("my-lazy-value")
		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize so create function is preserved
		lazy.Get(ctx)

		key, _ := WithTestLocalCtx(lazy)
		assert.Equal(t, "my-lazy-value", key, "should use existing name")
	})

	t.Run("panics if create function is nil", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string { return testValue })

		// Initialize without testing mode - this clears the create function
		_ = lazy.Get(context.Background())

		assert.Panics(t, func() {
			WithTestLocalCtx(lazy)
		}, "should panic when create function is nil")
	})

	t.Run("test-local instance is truly independent", func(t *testing.T) {
		t.Parallel()

		globalCallCount := atomic.Int32{}
		globalLazy := NewCtx(func(ctx context.Context) *testStruct {
			globalCallCount.Add(1)

			return &testStruct{Value: 100}
		})

		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize global
		globalResult := globalLazy.Get(ctx)
		assert.Equal(t, 100, globalResult.Value)
		assert.Equal(t, int32(1), globalCallCount.Load())

		// Create test-local instance
		_, getter := WithTestLocalCtx(globalLazy)
		localResult := getter(ctx)

		// Modify local result
		localResult.Value = 200

		// Global should be unchanged
		assert.Equal(t, 100, globalLazy.Get(ctx).Value)
		assert.Equal(t, 200, localResult.Value)
		assert.Equal(t, int32(2), globalCallCount.Load())
	})
}

func TestWithTestLocalCtxErr(t *testing.T) {
	t.Parallel()

	t.Run("creates independent test instance", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		globalLazy := NewCtxErr(func(ctx context.Context) (string, error) {
			callCount.Add(1)

			return "global-value", nil
		})

		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize global lazy
		result, err := globalLazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "global-value", result)
		assert.Equal(t, int32(1), callCount.Load())

		// Create test-local instance AFTER initializing global
		// Now that the bug is fixed, testing mode properly preserves the create function
		key, getter := WithTestLocalCtxErr(globalLazy)
		assert.NotEmpty(t, key)

		// Test-local instance should have its own state
		localResult, err := getter(ctx)
		require.NoError(t, err)
		assert.Equal(t, "global-value", localResult)
		assert.Equal(t, int32(2), callCount.Load(), "test-local instance should initialize independently")

		// Call again - should not re-initialize
		localResult, err = getter(ctx)
		require.NoError(t, err)
		assert.Equal(t, "global-value", localResult)
		assert.Equal(t, int32(2), callCount.Load(), "test-local instance should be memoized")
	})

	t.Run("assigns name to lazy value if unnamed", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (int, error) { return 42, nil })

		// Don't call Get() yet - we want to test WithTestLocalCtxErr with uninitialized lazy
		// The create function should still be available from NewCtxErr
		assert.Empty(t, lazy.name, "should start with no name")

		key, _ := WithTestLocalCtxErr(lazy)
		assert.NotEmpty(t, key, "should generate a key")
		assert.NotEmpty(t, lazy.name, "should assign name to lazy value")
		assert.Equal(t, string(lazy.name), key, "key should match assigned name")
	})

	t.Run("reuses existing name", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (int, error) {
			return 42, nil
		}).WithName("my-lazy-err-value")

		// Don't call Get() yet - we want to test WithTestLocalCtxErr with uninitialized lazy
		// The create function should still be available from NewCtxErr

		key, _ := WithTestLocalCtxErr(lazy)
		assert.Equal(t, "my-lazy-err-value", key, "should use existing name")
	})

	t.Run("panics if create function is nil", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) { return testValue, nil })

		// Initialize without testing mode - this clears the create function
		_, _ = lazy.Get(context.Background())

		assert.Panics(t, func() {
			WithTestLocalCtxErr(lazy)
		}, "should panic when create function is nil")
	})

	t.Run("handles errors independently", func(t *testing.T) {
		t.Parallel()

		globalCallCount := atomic.Int32{}
		globalLazy := NewCtxErr(func(ctx context.Context) (string, error) {
			count := globalCallCount.Add(1)
			if count == 1 {
				return "", assert.AnError
			}

			return "success", nil
		})

		ctx := WithTestingEnabled(context.Background(), true)

		// Global lazy errors on first call
		_, err := globalLazy.Get(ctx)
		require.Error(t, err)
		assert.Equal(t, int32(1), globalCallCount.Load())

		// Create test-local instance - should start fresh
		_, getter := WithTestLocalCtxErr(globalLazy)

		// Test-local will succeed because it's the second call overall
		localResult, err := getter(ctx)
		require.NoError(t, err)
		assert.Equal(t, "success", localResult)
		assert.Equal(t, int32(2), globalCallCount.Load())

		// Global should retry and succeed now
		globalResult, err := globalLazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "success", globalResult)
		assert.Equal(t, int32(3), globalCallCount.Load())
	})
}

func TestWithTestingEnabled(t *testing.T) {
	t.Parallel()

	t.Run("enables testing mode", func(t *testing.T) {
		t.Parallel()

		ctx := WithTestingEnabled(context.Background(), true)
		assert.True(t, isTestingEnabled(ctx))
	})

	t.Run("disables testing mode", func(t *testing.T) {
		t.Parallel()

		ctx := WithTestingEnabled(context.Background(), false)
		assert.False(t, isTestingEnabled(ctx))
	})

	t.Run("default context has testing disabled", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		assert.False(t, isTestingEnabled(ctx))
	})

	t.Run("preserves create function when enabled", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string { return testValue })

		ctx := WithTestingEnabled(context.Background(), true)
		lazy.Get(ctx)

		// Create function should still be available
		assert.NotNil(t, lazy.create.Load(), "create function should be preserved in testing mode")
	})

	t.Run("clears create function when disabled", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string { return testValue })

		ctx := WithTestingEnabled(context.Background(), false)
		lazy.Get(ctx)

		// Create function should be cleared
		assert.Nil(t, lazy.create.Load(), "create function should be cleared in non-testing mode")
	})

	t.Run("works with OfCtxErr", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) { return testValue, nil })

		ctx := WithTestingEnabled(context.Background(), true)
		_, _ = lazy.Get(ctx)

		// Create function should be preserved in testing mode, just like OfCtx
		assert.NotNil(t, lazy.create.Load(), "create function should be preserved in testing mode")
	})

	t.Run("preserves create function on error", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) { return "", assert.AnError })

		ctx := WithTestingEnabled(context.Background(), true)
		_, err := lazy.Get(ctx)
		require.Error(t, err)

		// When Get() returns an error, create function is preserved (even without testing mode)
		// This allows retrying the initialization
		assert.NotNil(t, lazy.create.Load(), "create function should be preserved on error")
	})
}

func TestWithValueOverride(t *testing.T) {
	t.Parallel()

	t.Run("overrides lazy value", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtx(func(ctx context.Context) string {
			callCount.Add(1)

			return originalValue
		}).WithName("test-value")

		ctx := WithValueOverride(context.Background(), "test-value", "overridden")

		result := lazy.Get(ctx)
		assert.Equal(t, "overridden", result)
		assert.Equal(t, int32(0), callCount.Load(), "create function should not be called")
	})

	t.Run("does not affect unnamed lazy values", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string { return originalValue })

		ctx := WithValueOverride(context.Background(), "some-key", "overridden")

		result := lazy.Get(ctx)
		assert.Equal(t, originalValue, result, "unnamed lazy value should not be affected")
	})

	t.Run("requires exact key match", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string {
			return originalValue
		}).WithName("test-value")

		ctx := WithValueOverride(context.Background(), "wrong-key", "overridden")

		result := lazy.Get(ctx)
		assert.Equal(t, originalValue, result, "should not override with wrong key")
	})

	t.Run("supports multiple overrides", func(t *testing.T) {
		t.Parallel()

		lazy1 := NewCtx(func(ctx context.Context) string { return "original1" }).WithName("key1")
		lazy2 := NewCtx(func(ctx context.Context) string { return "original2" }).WithName("key2")

		ctx := context.Background()
		ctx = WithValueOverride(ctx, "key1", "override1")
		ctx = WithValueOverride(ctx, "key2", "override2")

		assert.Equal(t, "override1", lazy1.Get(ctx))
		assert.Equal(t, "override2", lazy2.Get(ctx))
	})

	t.Run("works with different types", func(t *testing.T) {
		t.Parallel()

		intLazy := NewCtx(func(ctx context.Context) int { return 0 }).WithName("int-key")
		stringLazy := NewCtx(func(ctx context.Context) string { return "" }).WithName("string-key")

		ctx := context.Background()
		ctx = WithValueOverride(ctx, "int-key", 42)
		ctx = WithValueOverride(ctx, "string-key", "hello")

		assert.Equal(t, 42, intLazy.Get(ctx))
		assert.Equal(t, "hello", stringLazy.Get(ctx))
	})

	t.Run("override takes precedence over initialization", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string {
			return "initialized"
		}).WithName("test-key")

		// Initialize first
		result := lazy.Get(context.Background())
		assert.Equal(t, "initialized", result)

		// Override should still work
		ctx := WithValueOverride(context.Background(), "test-key", "overridden")
		result = lazy.Get(ctx)
		assert.Equal(t, "overridden", result)
	})
}

func TestWithValueOverrideProvider(t *testing.T) {
	t.Parallel()

	t.Run("overrides with provider function", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtx(func(ctx context.Context) string {
			callCount.Add(1)

			return originalValue
		}).WithName("test-value")

		providerCallCount := atomic.Int32{}
		provider := func(ctx context.Context) string {
			providerCallCount.Add(1)

			return fromProviderValue
		}

		ctx := WithValueOverrideProvider(context.Background(), "test-value", provider)

		result := lazy.Get(ctx)
		assert.Equal(t, fromProviderValue, result)
		assert.Equal(t, int32(0), callCount.Load(), "create function should not be called")
		assert.Equal(t, int32(1), providerCallCount.Load(), "provider should be called")
	})

	t.Run("provider is called on each Get", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) int {
			return 0
		}).WithName("counter")

		providerCallCount := atomic.Int32{}
		provider := func(ctx context.Context) int {
			return int(providerCallCount.Add(1))
		}

		ctx := WithValueOverrideProvider(context.Background(), "counter", provider)

		assert.Equal(t, 1, lazy.Get(ctx))
		assert.Equal(t, 2, lazy.Get(ctx))
		assert.Equal(t, 3, lazy.Get(ctx))
	})

	t.Run("does not affect unnamed lazy values", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string { return originalValue })

		provider := func(ctx context.Context) string { return fromProviderValue }
		ctx := WithValueOverrideProvider(context.Background(), "some-key", provider)

		result := lazy.Get(ctx)
		assert.Equal(t, originalValue, result, "unnamed lazy value should not be affected")
	})

	t.Run("provider receives context", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string {
			return originalValue
		}).WithName("test-value")

		type contextKey string

		var receivedValue string

		provider := func(ctx context.Context) string {
			if val, ok := ctx.Value(contextKey("custom-key")).(string); ok {
				receivedValue = val
			}

			return fromProviderValue
		}

		ctx := context.WithValue(context.Background(), contextKey("custom-key"), "custom-value")
		ctx = WithValueOverrideProvider(ctx, "test-value", provider)

		lazy.Get(ctx)
		assert.Equal(t, "custom-value", receivedValue, "provider should receive context")
	})

	t.Run("nil provider is ignored", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string {
			return originalValue
		}).WithName("test-value")

		ctx := WithValueOverrideProvider[string](context.Background(), "test-value", nil)

		result := lazy.Get(ctx)
		assert.Equal(t, originalValue, result, "nil provider should be ignored")
	})
}

func TestWithValueOverrideErrorProvider(t *testing.T) {
	t.Parallel()

	t.Run("overrides with error provider function", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			callCount.Add(1)

			return originalValue, nil
		}).WithName("test-value")

		providerCallCount := atomic.Int32{}
		provider := func(ctx context.Context) (string, error) {
			providerCallCount.Add(1)

			return fromProviderValue, nil
		}

		ctx := WithValueOverrideErrorProvider(context.Background(), "test-value", provider)

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, fromProviderValue, result)
		assert.Equal(t, int32(0), callCount.Load(), "create function should not be called")
		assert.Equal(t, int32(1), providerCallCount.Load(), "provider should be called")
	})

	t.Run("provider errors are returned", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		}).WithName("test-value")

		provider := func(ctx context.Context) (string, error) {
			return "", assert.AnError
		}

		ctx := WithValueOverrideErrorProvider(context.Background(), "test-value", provider)

		result, err := lazy.Get(ctx)
		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		assert.Empty(t, result)
	})

	t.Run("provider is called on each Get", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (int, error) {
			return 0, nil
		}).WithName("counter")

		providerCallCount := atomic.Int32{}
		provider := func(ctx context.Context) (int, error) {
			return int(providerCallCount.Add(1)), nil
		}

		ctx := WithValueOverrideErrorProvider(context.Background(), "counter", provider)

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, result)

		result, err = lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, result)
	})

	t.Run("does not affect unnamed lazy values", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		})

		provider := func(ctx context.Context) (string, error) {
			return fromProviderValue, nil
		}
		ctx := WithValueOverrideErrorProvider(context.Background(), "some-key", provider)

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, originalValue, result, "unnamed lazy value should not be affected")
	})

	t.Run("nil provider is ignored", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		}).WithName("test-value")

		ctx := WithValueOverrideErrorProvider[string](context.Background(), "test-value", nil)

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, originalValue, result, "nil provider should be ignored")
	})

	t.Run("works with OfCtx via regular provider", func(t *testing.T) {
		t.Parallel()

		// OfCtx can use error provider through type compatibility
		lazy := NewCtx(func(ctx context.Context) string {
			return originalValue
		}).WithName("test-value")

		// This won't work because OfCtx.Get doesn't check for error providers
		// This test documents the expected behavior
		provider := func(ctx context.Context) (string, error) {
			return fromProviderValue, nil
		}

		ctx := WithValueOverrideErrorProvider(context.Background(), "test-value", provider)

		result := lazy.Get(ctx)
		// OfCtx ignores error providers
		assert.Equal(t, originalValue, result, "OfCtx should ignore error providers")
	})
}

func TestOverridePrecedence(t *testing.T) {
	t.Parallel()

	t.Run("direct value takes precedence over provider", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) string {
			return originalValue
		}).WithName("test-value")

		provider := func(ctx context.Context) string {
			t.Fatal("provider should not be called")

			return fromProviderValue
		}

		ctx := context.Background()
		ctx = WithValueOverrideProvider(ctx, "test-value", provider)
		ctx = WithValueOverride(ctx, "test-value", "direct-value")

		result := lazy.Get(ctx)
		assert.Equal(t, "direct-value", result)
	})

	t.Run("error provider takes precedence over regular provider for OfCtxErr", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		}).WithName("test-value")

		regularProvider := func(ctx context.Context) string {
			t.Fatal("regular provider should not be called")

			return "from-regular-provider"
		}

		errorProvider := func(ctx context.Context) (string, error) {
			return "from-error-provider", nil
		}

		ctx := context.Background()
		ctx = WithValueOverrideProvider(ctx, "test-value", regularProvider)
		ctx = WithValueOverrideErrorProvider(ctx, "test-value", errorProvider)

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "from-error-provider", result)
	})

	t.Run("direct value takes precedence over error provider", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		}).WithName("test-value")

		errorProvider := func(ctx context.Context) (string, error) {
			t.Fatal("error provider should not be called")

			return "from-error-provider", nil
		}

		ctx := context.Background()
		ctx = WithValueOverrideErrorProvider(ctx, "test-value", errorProvider)
		ctx = WithValueOverride(ctx, "test-value", "direct-value")

		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "direct-value", result)
	})
}

func TestContextIntegrationWithOfCtx(t *testing.T) {
	t.Parallel()

	t.Run("override works across multiple calls", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtx(func(ctx context.Context) int {
			callCount.Add(1)

			return 42
		}).WithName("test-int")

		// First call without override
		result := lazy.Get(context.Background())
		assert.Equal(t, 42, result)
		assert.Equal(t, int32(1), callCount.Load())

		// Second call with override
		ctx := WithValueOverride(context.Background(), "test-int", 100)
		result = lazy.Get(ctx)
		assert.Equal(t, 100, result)
		assert.Equal(t, int32(1), callCount.Load(), "should not initialize again")

		// Third call without override - should return memoized originalValue
		result = lazy.Get(context.Background())
		assert.Equal(t, 42, result)
		assert.Equal(t, int32(1), callCount.Load())
	})

	t.Run("testing mode with test-local instance", func(t *testing.T) {
		t.Parallel()

		globalCallCount := atomic.Int32{}
		globalLazy := NewCtx(func(ctx context.Context) string {
			count := globalCallCount.Add(1)

			return "value-" + string('0'+count)
		}).WithName("global")

		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize global
		globalResult := globalLazy.Get(ctx)
		assert.Equal(t, "value-1", globalResult)

		// Create test-local instance
		key, getter := WithTestLocalCtx(globalLazy)

		// The test-local instance is completely independent and doesn't have a name,
		// so context overrides won't affect it. The key returned is for the global lazy.
		testResult := getter(ctx)
		assert.Equal(t, "value-2", testResult, "test-local instance initializes independently")

		// But we can override the global instance using the key
		overrideCtx := WithValueOverride(ctx, key, "overridden-global")
		assert.Equal(t, "overridden-global", globalLazy.Get(overrideCtx))

		// Test-local still returns its own initialized value
		assert.Equal(t, "value-2", getter(ctx))
	})
}

func TestContextIntegrationWithOfCtxErr(t *testing.T) {
	t.Parallel()

	t.Run("override works with error handling", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			count := callCount.Add(1)
			if count < 2 {
				return "", assert.AnError
			}

			return "success", nil
		}).WithName("test-value")

		// First call errors
		_, err := lazy.Get(context.Background())
		require.Error(t, err)
		assert.Equal(t, int32(1), callCount.Load())

		// Override to skip error
		ctx := WithValueOverride(context.Background(), "test-value", "overridden")
		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "overridden", result)
		assert.Equal(t, int32(1), callCount.Load(), "should not call create function")

		// Without override, it should retry and succeed
		result, err = lazy.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "success", result)
		assert.Equal(t, int32(2), callCount.Load())
	})

	t.Run("error provider can return errors", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtxErr(func(ctx context.Context) (string, error) {
			return originalValue, nil
		}).WithName("test-value")

		attempts := atomic.Int32{}
		errorProvider := func(ctx context.Context) (string, error) {
			count := attempts.Add(1)
			if count < 3 {
				return "", assert.AnError
			}

			return "success-from-provider", nil
		}

		ctx := WithValueOverrideErrorProvider(context.Background(), "test-value", errorProvider)

		// First two attempts error
		_, err := lazy.Get(ctx)
		require.Error(t, err)

		_, err = lazy.Get(ctx)
		require.Error(t, err)

		// Third attempt succeeds
		result, err := lazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, "success-from-provider", result)
	})

	t.Run("testing mode with test-local error instance", func(t *testing.T) {
		t.Parallel()

		globalCallCount := atomic.Int32{}
		globalLazy := NewCtxErr(func(ctx context.Context) (int, error) {
			count := globalCallCount.Add(1)
			if count == 1 {
				return 0, assert.AnError
			}

			return int(count), nil
		}).WithName("global-err")

		ctx := WithTestingEnabled(context.Background(), true)

		// Initialize global - errors first time
		_, err := globalLazy.Get(ctx)
		require.Error(t, err)

		// Create test-local instance BEFORE global succeeds
		// (must do this before global Get() succeeds or create function gets cleared)
		key, getter := WithTestLocalCtxErr(globalLazy)

		// Test-local starts fresh and shares the create function
		// It will succeed because it's the second call (count=2)
		testResult, err := getter(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, testResult, "test-local initializes independently")

		// We can override the global instance using the key
		overrideCtx := WithValueOverride(ctx, key, 999)
		globalOverridden, err := globalLazy.Get(overrideCtx)
		require.NoError(t, err)
		assert.Equal(t, 999, globalOverridden, "global can be overridden")

		// Global should succeed on retry without override
		globalResult, err := globalLazy.Get(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, globalResult, "global retries and succeeds")
	})
}

func TestConcurrentContextOverrides(t *testing.T) {
	t.Parallel()

	t.Run("concurrent overrides don't interfere", func(t *testing.T) {
		t.Parallel()

		callCount := atomic.Int32{}
		lazy := NewCtx(func(ctx context.Context) int {
			callCount.Add(1)

			return 0
		}).WithName("concurrent-test")

		const goroutines = 100

		done := make(chan int, goroutines)

		// Launch goroutines with different overrides
		for i := range goroutines {
			go func() {
				ctx := WithValueOverride(context.Background(), "concurrent-test", i)

				result := lazy.Get(ctx)
				done <- result
			}()
		}

		// Collect results - each should get its own override
		results := make(map[int]bool)

		for range goroutines {
			result := <-done
			results[result] = true
		}

		// Should have gotten all different values
		assert.Len(t, results, goroutines)
		assert.Equal(t, int32(0), callCount.Load(), "create function should never be called")
	})

	t.Run("concurrent provider calls", func(t *testing.T) {
		t.Parallel()

		lazy := NewCtx(func(ctx context.Context) int {
			return 0
		}).WithName("provider-test")

		providerCallCount := atomic.Int32{}
		provider := func(ctx context.Context) int {
			return int(providerCallCount.Add(1))
		}

		ctx := WithValueOverrideProvider(context.Background(), "provider-test", provider)

		const goroutines = 100

		done := make(chan int, goroutines)

		// Launch concurrent Gets
		for range goroutines {
			go func() {
				result := lazy.Get(ctx)
				done <- result
			}()
		}

		// Collect results
		results := make(map[int]bool)

		for range goroutines {
			result := <-done
			results[result] = true
		}

		// Each call should get a unique value from provider
		assert.Len(t, results, goroutines)
		assert.Equal(t, int32(goroutines), providerCallCount.Load())
	})
}

// Helper type for testing.
type testStruct struct {
	Value int
}
