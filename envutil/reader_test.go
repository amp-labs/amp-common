package envutil_test

import (
	"testing"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultValue = "default"

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderValueOrPanic(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		t.Setenv("TEST_PANIC", "value")

		reader := envutil.String("TEST_PANIC")

		assert.NotPanics(t, func() {
			value := reader.ValueOrPanic()
			assert.Equal(t, "value", value)
		})
	})

	t.Run("panics when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_PANIC_MISSING")

		assert.Panics(t, func() {
			reader.ValueOrPanic()
		})
	})

	t.Run("panics on error", func(t *testing.T) {
		t.Setenv("TEST_PANIC_ERROR", "not-a-number")

		reader := envutil.Int[int]("TEST_PANIC_ERROR")

		assert.Panics(t, func() {
			reader.ValueOrPanic()
		})
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderValueOrElse(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		t.Setenv("TEST_OR_ELSE", "value")

		reader := envutil.String("TEST_OR_ELSE")
		value := reader.ValueOrElse("default")
		assert.Equal(t, "value", value)
	})

	t.Run("returns default when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_OR_ELSE_MISSING")
		value := reader.ValueOrElse("default")
		assert.Equal(t, "default", value)
	})

	t.Run("returns default on error", func(t *testing.T) {
		t.Setenv("TEST_OR_ELSE_ERROR", "not-a-number")

		reader := envutil.Int[int]("TEST_OR_ELSE_ERROR")
		value := reader.ValueOrElse(42)
		assert.Equal(t, 42, value)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderValueOrElseFunc(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		t.Setenv("TEST_OR_ELSE_FUNC", "value")

		called := false
		reader := envutil.String("TEST_OR_ELSE_FUNC")
		value := reader.ValueOrElseFunc(func() string {
			called = true

			return defaultValue
		})
		assert.Equal(t, "value", value)
		assert.False(t, called, "function should not be called when value is present")
	})

	t.Run("calls function when missing", func(t *testing.T) {
		t.Parallel()

		called := false
		reader := envutil.String("TEST_OR_ELSE_FUNC_MISSING")
		value := reader.ValueOrElseFunc(func() string {
			called = true

			return defaultValue
		})
		assert.Equal(t, defaultValue, value)
		assert.True(t, called)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderValueOrElseFuncErr(t *testing.T) {
	t.Run("returns value when present", func(t *testing.T) {
		t.Setenv("TEST_OR_ELSE_FUNC_ERR", "value")

		called := false
		reader := envutil.String("TEST_OR_ELSE_FUNC_ERR")
		value, err := reader.ValueOrElseFuncErr(func() (string, error) {
			called = true

			return defaultValue, nil
		})
		require.NoError(t, err)
		assert.Equal(t, "value", value)
		assert.False(t, called)
	})

	t.Run("calls function when missing", func(t *testing.T) {
		t.Parallel()

		called := false
		reader := envutil.String("TEST_OR_ELSE_FUNC_ERR_MISSING")
		value, err := reader.ValueOrElseFuncErr(func() (string, error) {
			called = true

			return defaultValue, nil
		})
		require.NoError(t, err)
		assert.Equal(t, defaultValue, value)
		assert.True(t, called)
	})

	t.Run("returns error from function", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_OR_ELSE_FUNC_ERR_MISSING2")
		_, err := reader.ValueOrElseFuncErr(func() (string, error) {
			return "", assert.AnError
		})
		require.Error(t, err)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderDoWithValue(t *testing.T) {
	t.Run("calls function when present", func(t *testing.T) {
		t.Setenv("TEST_DO_WITH", "value")

		called := false

		var receivedValue string

		reader := envutil.String("TEST_DO_WITH")
		reader.DoWithValue(func(v string) {
			called = true
			receivedValue = v
		})
		assert.True(t, called)
		assert.Equal(t, "value", receivedValue)
	})

	t.Run("does not call function when missing", func(t *testing.T) {
		t.Parallel()

		called := false
		reader := envutil.String("TEST_DO_WITH_MISSING")
		reader.DoWithValue(func(v string) {
			called = true
		})
		assert.False(t, called)
	})

	t.Run("does not call function on error", func(t *testing.T) {
		t.Setenv("TEST_DO_WITH_ERROR", "not-a-number")

		called := false
		reader := envutil.Int[int]("TEST_DO_WITH_ERROR")
		reader.DoWithValue(func(v int) {
			called = true
		})
		assert.False(t, called)
	})
}

//nolint:dupl,tparallel // Similar test structure; Cannot use t.Parallel() with subtests using t.Setenv()
func TestReaderHasValue(t *testing.T) {
	t.Run("true when present", func(t *testing.T) {
		t.Setenv("TEST_HAS_VALUE", "value")

		reader := envutil.String("TEST_HAS_VALUE")
		assert.True(t, reader.HasValue())
	})

	t.Run("false when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_HAS_VALUE_MISSING")
		assert.False(t, reader.HasValue())
	})

	t.Run("false on error", func(t *testing.T) {
		t.Setenv("TEST_HAS_VALUE_ERROR", "not-a-number")

		reader := envutil.Int[int]("TEST_HAS_VALUE_ERROR")
		assert.False(t, reader.HasValue())
	})
}

//nolint:dupl,tparallel // Similar test structure; Cannot use t.Parallel() with subtests using t.Setenv()
func TestReaderHasError(t *testing.T) {
	t.Run("false when valid", func(t *testing.T) {
		t.Setenv("TEST_HAS_ERROR", "value")

		reader := envutil.String("TEST_HAS_ERROR")
		assert.False(t, reader.HasError())
	})

	t.Run("false when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_HAS_ERROR_MISSING")
		assert.False(t, reader.HasError())
	})

	t.Run("true on parse error", func(t *testing.T) {
		t.Setenv("TEST_HAS_ERROR_INVALID", "not-a-number")

		reader := envutil.Int[int]("TEST_HAS_ERROR_INVALID")
		assert.True(t, reader.HasError())
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderString(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		t.Setenv("TEST_STRING_METHOD", "value")

		reader := envutil.String("TEST_STRING_METHOD")
		str := reader.String()
		assert.Contains(t, str, "TEST_STRING_METHOD")
		assert.Contains(t, str, "value")
	})

	t.Run("without value", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_STRING_METHOD_MISSING")
		str := reader.String()
		assert.Contains(t, str, "TEST_STRING_METHOD_MISSING")
		assert.Contains(t, str, "not set")
	})

	t.Run("with error", func(t *testing.T) {
		t.Setenv("TEST_STRING_METHOD_ERROR", "not-a-number")

		reader := envutil.Int[int]("TEST_STRING_METHOD_ERROR")
		str := reader.String()
		assert.Contains(t, str, "TEST_STRING_METHOD_ERROR")
		assert.Contains(t, str, "error")
	})
}

func TestReaderKey(t *testing.T) {
	t.Parallel()

	reader := envutil.String("TEST_KEY_METHOD")
	assert.Equal(t, "TEST_KEY_METHOD", reader.Key())
}

func TestReaderError(t *testing.T) {
	t.Run("no error when valid", func(t *testing.T) {
		t.Setenv("TEST_ERROR_METHOD", "value")

		reader := envutil.String("TEST_ERROR_METHOD")
		assert.NoError(t, reader.Error())
	})

	t.Run("error when invalid", func(t *testing.T) {
		t.Setenv("TEST_ERROR_METHOD_INVALID", "not-a-number")

		reader := envutil.Int[int]("TEST_ERROR_METHOD_INVALID")
		assert.Error(t, reader.Error())
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderWithErrorIfMissing(t *testing.T) {
	t.Run("adds error when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_WITH_ERROR_MISSING")
		readerWithErr := reader.WithErrorIfMissing(assert.AnError)

		_, err := readerWithErr.Value()
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("preserves value when present", func(t *testing.T) {
		t.Setenv("TEST_WITH_ERROR_PRESENT", "value")

		reader := envutil.String("TEST_WITH_ERROR_PRESENT")
		readerWithErr := reader.WithErrorIfMissing(assert.AnError)

		value, err := readerWithErr.Value()
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})

	t.Run("preserves existing error", func(t *testing.T) {
		t.Setenv("TEST_WITH_ERROR_EXISTING", "not-a-number")

		reader := envutil.Int[int]("TEST_WITH_ERROR_EXISTING")
		readerWithErr := reader.WithErrorIfMissing(assert.AnError)

		_, err := readerWithErr.Value()
		require.Error(t, err)
		assert.NotErrorIs(t, err, assert.AnError)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderWithDefault(t *testing.T) {
	t.Run("uses default when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_WITH_DEFAULT_MISSING")
		readerWithDefault := reader.WithDefault("default")

		value, err := readerWithDefault.Value()
		require.NoError(t, err)
		assert.Equal(t, "default", value)
	})

	t.Run("preserves value when present", func(t *testing.T) {
		t.Setenv("TEST_WITH_DEFAULT_PRESENT", "value")

		reader := envutil.String("TEST_WITH_DEFAULT_PRESENT")
		readerWithDefault := reader.WithDefault("default")

		value, err := readerWithDefault.Value()
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})
}

func TestReaderWithFallback(t *testing.T) {
	t.Run("uses fallback when missing", func(t *testing.T) {
		t.Setenv("TEST_WITH_FALLBACK_B", "fallback")

		reader := envutil.String("TEST_WITH_FALLBACK_A")
		fallback := envutil.String("TEST_WITH_FALLBACK_B")
		readerWithFallback := reader.WithFallback(fallback)

		value, err := readerWithFallback.Value()
		require.NoError(t, err)
		assert.Equal(t, "fallback", value)
	})

	t.Run("preserves value when present", func(t *testing.T) {
		t.Setenv("TEST_WITH_FALLBACK_PRIMARY", "primary")
		t.Setenv("TEST_WITH_FALLBACK_SECONDARY", "fallback")

		reader := envutil.String("TEST_WITH_FALLBACK_PRIMARY")
		fallback := envutil.String("TEST_WITH_FALLBACK_SECONDARY")
		readerWithFallback := reader.WithFallback(fallback)

		value, err := readerWithFallback.Value()
		require.NoError(t, err)
		assert.Equal(t, "primary", value)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestReaderMap(t *testing.T) {
	t.Run("transforms value", func(t *testing.T) {
		t.Setenv("TEST_MAP", "hello")

		reader := envutil.String("TEST_MAP")
		mapped := reader.Map(func(s string) (string, error) {
			return s + " world", nil
		})

		value, err := mapped.Value()
		require.NoError(t, err)
		assert.Equal(t, "hello world", value)
	})

	t.Run("propagates error from map function", func(t *testing.T) {
		t.Setenv("TEST_MAP_ERROR", "value")

		reader := envutil.String("TEST_MAP_ERROR")
		mapped := reader.Map(func(s string) (string, error) {
			return "", assert.AnError
		})

		_, err := mapped.Value()
		require.Error(t, err)
	})

	t.Run("skips map when value missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_MAP_MISSING")
		mapped := reader.Map(func(s string) (string, error) {
			t.Fatal("map function should not be called")

			return "", nil
		})

		_, err := mapped.Value()
		require.Error(t, err)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestMapFunction(t *testing.T) {
	t.Run("transforms type A to B", func(t *testing.T) {
		t.Setenv("TEST_MAP_FUNC", "42")

		reader := envutil.String("TEST_MAP_FUNC")
		mapped := envutil.Map(reader, func(s string) (int, error) {
			if s == "42" {
				return 42, nil
			}

			return 0, assert.AnError
		})

		value, err := mapped.Value()
		require.NoError(t, err)
		assert.Equal(t, 42, value)
	})

	t.Run("preserves missing state", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String("TEST_MAP_FUNC_MISSING")
		mapped := envutil.Map(reader, func(s string) (int, error) {
			t.Fatal("map function should not be called")

			return 0, nil
		})

		_, err := mapped.Value()
		require.Error(t, err)
	})

	t.Run("preserves error state", func(t *testing.T) {
		t.Setenv("TEST_MAP_FUNC_ERROR", "not-a-number")

		reader := envutil.Int[int]("TEST_MAP_FUNC_ERROR")
		mapped := envutil.Map(reader, func(i int) (string, error) {
			t.Fatal("map function should not be called")

			return "", nil
		})

		_, err := mapped.Value()
		require.Error(t, err)
	})
}
