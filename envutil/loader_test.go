package envutil

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	t.Parallel()

	t.Run("creates empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()

		require.NotNil(t, loader)
		assert.Empty(t, loader.environment)
		assert.Empty(t, loader.Keys())
	})

	t.Run("multiple loaders are independent", func(t *testing.T) {
		t.Parallel()

		loader1 := NewLoader()
		loader2 := NewLoader()

		loader1.Set("KEY1", "value1")
		loader2.Set("KEY2", "value2")

		assert.True(t, loader1.Contains("KEY1"))
		assert.False(t, loader1.Contains("KEY2"))
		assert.False(t, loader2.Contains("KEY1"))
		assert.True(t, loader2.Contains("KEY2"))
	})
}

// nolint:tparallel // Cannot use t.Parallel() on parent test because some subtests use t.Setenv()
func TestLoader_LoadEnv(t *testing.T) {
	t.Run("loads current process environment", func(t *testing.T) {
		// Set a unique environment variable for this test
		testKey := "TEST_LOAD_ENV_UNIQUE_KEY"
		testValue := "test_value_12345"
		t.Setenv(testKey, testValue)

		loader := NewLoader()
		loader.LoadEnv()

		value, found := loader.Get(testKey)
		assert.True(t, found)
		assert.Equal(t, testValue, value)
	})

	t.Run("loads into empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		assert.Empty(t, loader.Keys())

		loader.LoadEnv()

		// Should have loaded at least some environment variables
		assert.NotEmpty(t, loader.Keys())
	})

	t.Run("merges with existing variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("CUSTOM_KEY", "custom_value")

		loader.LoadEnv()

		// Custom key should still exist
		value, found := loader.Get("CUSTOM_KEY")
		assert.True(t, found)
		assert.Equal(t, "custom_value", value)
	})

	t.Run("overrides existing variables", func(t *testing.T) {
		testKey := "TEST_LOAD_ENV_OVERRIDE"
		testValue := "env_value"
		t.Setenv(testKey, testValue)

		loader := NewLoader()
		loader.Set(testKey, "old_value")

		loader.LoadEnv()

		value, found := loader.Get(testKey)
		assert.True(t, found)
		assert.Equal(t, testValue, value) // Should be overridden by env value
	})
}

func TestLoader_LoadFile(t *testing.T) {
	t.Parallel()

	t.Run("loads .env file", func(t *testing.T) {
		t.Parallel()

		// Create a temporary .env file
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, "test.env")
		content := `KEY1=value1
KEY2=value2
# Comment
KEY3=value3`
		err := os.WriteFile(envFile, []byte(content), 0o600)
		require.NoError(t, err)

		loader := NewLoader()
		count, err := loader.LoadFile(envFile)

		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
		assert.Equal(t, "value1", mustGet(t, loader, "KEY1"))
		assert.Equal(t, "value2", mustGet(t, loader, "KEY2"))
		assert.Equal(t, "value3", mustGet(t, loader, "KEY3"))
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		_, err := loader.LoadFile("/non/existent/file.env")

		assert.Error(t, err)
	})

	t.Run("merges with existing variables", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, "test.env")
		err := os.WriteFile(envFile, []byte("FILE_KEY=file_value"), 0o600)
		require.NoError(t, err)

		loader := NewLoader()
		loader.Set("EXISTING_KEY", "existing_value")

		_, err = loader.LoadFile(envFile)
		require.NoError(t, err)

		assert.Equal(t, "existing_value", mustGet(t, loader, "EXISTING_KEY"))
		assert.Equal(t, "file_value", mustGet(t, loader, "FILE_KEY"))
	})

	t.Run("overrides existing variables", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, "test.env")
		err := os.WriteFile(envFile, []byte("SHARED_KEY=new_value"), 0o600)
		require.NoError(t, err)

		loader := NewLoader()
		loader.Set("SHARED_KEY", "old_value")

		_, err = loader.LoadFile(envFile)
		require.NoError(t, err)

		assert.Equal(t, "new_value", mustGet(t, loader, "SHARED_KEY"))
	})
}

func TestLoader_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns value when key exists", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		value, found := loader.Get("KEY")

		assert.True(t, found)
		assert.Equal(t, "value", value)
	})

	t.Run("returns empty string and false when key missing", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()

		value, found := loader.Get("MISSING_KEY")

		assert.False(t, found)
		assert.Empty(t, value)
	})

	t.Run("returns empty string value when set", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EMPTY_KEY", "")

		value, found := loader.Get("EMPTY_KEY")

		assert.True(t, found)
		assert.Empty(t, value)
	})
}

func TestLoader_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets new variable", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("NEW_KEY", "new_value")

		assert.Equal(t, "new_value", mustGet(t, loader, "NEW_KEY"))
	})

	t.Run("updates existing variable", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "old_value")
		loader.Set("KEY", "new_value")

		assert.Equal(t, "new_value", mustGet(t, loader, "KEY"))
	})

	t.Run("sets empty string value", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EMPTY", "")

		value, found := loader.Get("EMPTY")
		assert.True(t, found)
		assert.Empty(t, value)
	})
}

func TestLoader_SetAll(t *testing.T) {
	t.Parallel()

	t.Run("sets multiple variables at once", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		vars := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
			"KEY3": "value3",
		}

		loader.SetAll(vars)

		assert.Equal(t, "value1", mustGet(t, loader, "KEY1"))
		assert.Equal(t, "value2", mustGet(t, loader, "KEY2"))
		assert.Equal(t, "value3", mustGet(t, loader, "KEY3"))
	})

	t.Run("merges with existing variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EXISTING", "existing_value")

		loader.SetAll(map[string]string{
			"NEW": "new_value",
		})

		assert.Equal(t, "existing_value", mustGet(t, loader, "EXISTING"))
		assert.Equal(t, "new_value", mustGet(t, loader, "NEW"))
	})

	t.Run("overrides existing variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "old_value")

		loader.SetAll(map[string]string{
			"KEY": "new_value",
		})

		assert.Equal(t, "new_value", mustGet(t, loader, "KEY"))
	})

	t.Run("handles empty map", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EXISTING", "value")

		loader.SetAll(map[string]string{})

		assert.Equal(t, "value", mustGet(t, loader, "EXISTING"))
		assert.Len(t, loader.Keys(), 1)
	})
}

func TestLoader_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing variable", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		loader.Delete("KEY")

		_, found := loader.Get("KEY")
		assert.False(t, found)
	})

	t.Run("is no-op for missing variable", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		loader.Delete("MISSING_KEY")

		assert.Equal(t, "value", mustGet(t, loader, "KEY"))
	})
}

func TestLoader_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")
		loader.Set("KEY3", "value3")

		loader.Clear()

		assert.Empty(t, loader.Keys())
		assert.False(t, loader.Contains("KEY1"))
		assert.False(t, loader.Contains("KEY2"))
		assert.False(t, loader.Contains("KEY3"))
	})

	t.Run("is no-op on empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Clear()

		assert.Empty(t, loader.Keys())
	})

	t.Run("can add variables after clear", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")
		loader.Clear()
		loader.Set("NEW_KEY", "new_value")

		assert.Len(t, loader.Keys(), 1)
		assert.Equal(t, "new_value", mustGet(t, loader, "NEW_KEY"))
	})
}

func TestLoader_Filter(t *testing.T) {
	t.Parallel()

	t.Run("keeps only matching variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("APP_KEY1", "value1")
		loader.Set("APP_KEY2", "value2")
		loader.Set("OTHER_KEY", "value3")

		loader.Filter(func(key, _ string) bool {
			return strings.HasPrefix(key, "APP_")
		})

		assert.Len(t, loader.Keys(), 2)
		assert.True(t, loader.Contains("APP_KEY1"))
		assert.True(t, loader.Contains("APP_KEY2"))
		assert.False(t, loader.Contains("OTHER_KEY"))
	})

	t.Run("filters based on value", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "keep")
		loader.Set("KEY2", "")
		loader.Set("KEY3", "keep")

		loader.Filter(func(_, value string) bool {
			return value != ""
		})

		assert.Len(t, loader.Keys(), 2)
		assert.True(t, loader.Contains("KEY1"))
		assert.True(t, loader.Contains("KEY3"))
		assert.False(t, loader.Contains("KEY2"))
	})

	t.Run("removes all when predicate always returns false", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")

		loader.Filter(func(_, _ string) bool {
			return false
		})

		assert.Empty(t, loader.Keys())
	})

	t.Run("keeps all when predicate always returns true", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")

		loader.Filter(func(_, _ string) bool {
			return true
		})

		assert.Len(t, loader.Keys(), 2)
	})

	t.Run("is no-op on empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Filter(func(_, _ string) bool {
			return true
		})

		assert.Empty(t, loader.Keys())
	})
}

func TestLoader_Contains(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing key", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		assert.True(t, loader.Contains("KEY"))
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()

		assert.False(t, loader.Contains("MISSING"))
	})

	t.Run("returns true for empty value", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EMPTY", "")

		assert.True(t, loader.Contains("EMPTY"))
	})
}

func TestLoader_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns all keys", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")
		loader.Set("KEY3", "value3")

		keys := loader.Keys()

		assert.Len(t, keys, 3)
		assert.True(t, slices.Contains(keys, "KEY1"))
		assert.True(t, slices.Contains(keys, "KEY2"))
		assert.True(t, slices.Contains(keys, "KEY3"))
	})

	t.Run("returns empty slice for empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		keys := loader.Keys()

		assert.NotNil(t, keys)
		assert.Empty(t, keys)
	})

	t.Run("returns copy of keys", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		keys := loader.Keys()
		_ = append(keys, "MODIFIED") // Append to copy to verify it doesn't affect loader

		// Original loader should be unaffected
		assert.Len(t, loader.Keys(), 1)
	})
}

func TestLoader_AsMap(t *testing.T) {
	t.Parallel()

	t.Run("returns copy of environment map", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")

		m := loader.AsMap()

		assert.Len(t, m, 2)
		assert.Equal(t, "value1", m["KEY1"])
		assert.Equal(t, "value2", m["KEY2"])
	})

	t.Run("returns empty map for empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		m := loader.AsMap()

		assert.NotNil(t, m)
		assert.Empty(t, m)
	})

	t.Run("modifying returned map does not affect loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "original")

		m := loader.AsMap()
		m["KEY"] = "modified"
		m["NEW_KEY"] = "new_value"

		// Original loader should be unaffected
		assert.Equal(t, "original", mustGet(t, loader, "KEY"))
		assert.False(t, loader.Contains("NEW_KEY"))
	})
}

func TestLoader_AsSlice(t *testing.T) {
	t.Parallel()

	t.Run("returns slice of KEY=VALUE strings", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")

		slice := loader.AsSlice()

		assert.Len(t, slice, 2)
		assert.True(t, slices.Contains(slice, "KEY1=value1"))
		assert.True(t, slices.Contains(slice, "KEY2=value2"))
	})

	t.Run("returns empty slice for empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		slice := loader.AsSlice()

		assert.NotNil(t, slice)
		assert.Empty(t, slice)
	})

	t.Run("handles empty values", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("EMPTY", "")

		slice := loader.AsSlice()

		assert.Len(t, slice, 1)
		assert.Equal(t, "EMPTY=", slice[0])
	})

	t.Run("handles values with equals signs", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value=with=equals")

		slice := loader.AsSlice()

		assert.Len(t, slice, 1)
		assert.Equal(t, "KEY=value=with=equals", slice[0])
	})
}

func TestLoader_EnhanceContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with environment overrides", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("TEST_KEY", "test_value")

		ctx := loader.EnhanceContext(context.Background())

		// Verify the value is in the context by using envutil readers
		value, found := getEnvOverride(ctx, "TEST_KEY")
		assert.True(t, found)
		assert.Equal(t, "test_value", value)
	})

	t.Run("does not modify original context", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		originalCtx := context.Background()
		enhancedCtx := loader.EnhanceContext(originalCtx)

		// Original context should not have the override
		_, found := getEnvOverride(originalCtx, "KEY")
		assert.False(t, found)

		// Enhanced context should have the override
		_, found = getEnvOverride(enhancedCtx, "KEY")
		assert.True(t, found)
	})

	t.Run("includes all loader variables", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")
		loader.Set("KEY3", "value3")

		ctx := loader.EnhanceContext(context.Background())

		for _, key := range []string{"KEY1", "KEY2", "KEY3"} {
			_, found := getEnvOverride(ctx, key)
			assert.True(t, found, "key %s should be in context", key)
		}
	})
}

func TestLoader_SetContext(t *testing.T) {
	t.Parallel()

	t.Run("calls setter for each variable", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY1", "value1")
		loader.Set("KEY2", "value2")

		called := make(map[string]string)
		setter := func(key, value any) {
			// The key is envContextKey type, need to convert to string
			keyVal, ok := key.(envContextKey)
			require.True(t, ok, "key should be envContextKey type")
			valueStr, ok := value.(string)
			require.True(t, ok, "value should be string type")

			called[string(keyVal)] = valueStr
		}

		loader.SetContext(setter)

		assert.Len(t, called, 2)
		assert.Equal(t, "value1", called["KEY1"])
		assert.Equal(t, "value2", called["KEY2"])
	})

	t.Run("handles nil setter", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("KEY", "value")

		// Should not panic
		assert.NotPanics(t, func() {
			loader.SetContext(nil)
		})
	})

	t.Run("calls setter with empty loader", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()

		called := false
		setter := func(_, _ any) {
			called = true
		}

		loader.SetContext(setter)

		assert.False(t, called)
	})
}

// Integration tests

func TestLoader_Integration_ConfigurationLayering(t *testing.T) {
	t.Parallel()

	t.Run("layers configuration from multiple sources", func(t *testing.T) {
		t.Parallel()

		// Create test files
		tmpDir := t.TempDir()

		baseEnv := filepath.Join(tmpDir, "base.env")
		err := os.WriteFile(baseEnv, []byte(`
DATABASE_URL=postgres://localhost/base
PORT=8080
LOG_LEVEL=info
`), 0o600)
		require.NoError(t, err)

		prodEnv := filepath.Join(tmpDir, "production.env")
		err = os.WriteFile(prodEnv, []byte(`
DATABASE_URL=postgres://prod-server/myapp
LOG_LEVEL=error
`), 0o600)
		require.NoError(t, err)

		// Layer configurations
		loader := NewLoader()
		_, err = loader.LoadFile(baseEnv)
		require.NoError(t, err)
		_, err = loader.LoadFile(prodEnv)
		require.NoError(t, err)
		loader.Set("ENVIRONMENT", "production")

		// Verify final configuration
		assert.Equal(t, "postgres://prod-server/myapp", mustGet(t, loader, "DATABASE_URL"))
		assert.Equal(t, "8080", mustGet(t, loader, "PORT")) // From base
		assert.Equal(t, "error", mustGet(t, loader, "LOG_LEVEL"))
		assert.Equal(t, "production", mustGet(t, loader, "ENVIRONMENT"))
	})
}

func TestLoader_Integration_SecurityFiltering(t *testing.T) {
	t.Parallel()

	t.Run("filters sensitive variables before subprocess", func(t *testing.T) {
		t.Parallel()

		loader := NewLoader()
		loader.Set("DATABASE_URL", "postgres://localhost/myapp")
		loader.Set("API_SECRET", "secret123")
		loader.Set("JWT_PRIVATE_KEY", "private_key_data")
		loader.Set("PORT", "8080")
		loader.Set("LOG_LEVEL", "info")

		// Remove sensitive variables
		loader.Filter(func(key, _ string) bool {
			sensitive := []string{"SECRET", "KEY", "PASSWORD", "TOKEN"}
			for _, s := range sensitive {
				if strings.Contains(key, s) {
					return false
				}
			}

			return true
		})

		// Verify only non-sensitive variables remain
		assert.Len(t, loader.Keys(), 3)
		assert.True(t, loader.Contains("DATABASE_URL"))
		assert.True(t, loader.Contains("PORT"))
		assert.True(t, loader.Contains("LOG_LEVEL"))
		assert.False(t, loader.Contains("API_SECRET"))
		assert.False(t, loader.Contains("JWT_PRIVATE_KEY"))
	})
}

// Helper functions

func mustGet(t *testing.T, loader *Loader, key string) string {
	t.Helper()

	value, found := loader.Get(key)
	require.True(t, found, "key %s should exist", key)

	return value
}
