package envutil_test

import (
	"compress/gzip"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestString(t *testing.T) {
	t.Run("present value", func(t *testing.T) {
		t.Setenv("TEST_STRING", "hello")

		reader := envutil.String(t.Context(), "TEST_STRING")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "hello", value)
		assert.True(t, reader.HasValue())
	})

	t.Run("missing value", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String(t.Context(), "TEST_STRING_MISSING")
		_, err := reader.Value()
		require.Error(t, err)
		assert.False(t, reader.HasValue())
	})

	t.Run("with default", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String(t.Context(), "TEST_STRING_MISSING", envutil.Default("default"))
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "default", value)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestBytes(t *testing.T) {
	t.Run("present value", func(t *testing.T) {
		t.Setenv("TEST_BYTES", "hello")

		reader := envutil.Bytes(t.Context(), "TEST_BYTES")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), value)
	})

	t.Run("with default", func(t *testing.T) {
		t.Parallel()

		reader := envutil.Bytes(t.Context(), "TEST_BYTES_MISSING", envutil.Default([]byte("default")))
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, []byte("default"), value)
	})
}

func TestBool(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"true uppercase", "TRUE", true},
		{"1", "1", true},
		{"t", "t", true},
		{"false lowercase", "false", false},
		{"false uppercase", "FALSE", false},
		{"0", "0", false},
		{"f", "f", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL_" + tt.name
			t.Setenv(key, tt.value)

			reader := envutil.Bool(t.Context(), key)
			value, err := reader.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}

	t.Run("invalid value", func(t *testing.T) {
		t.Setenv("TEST_BOOL_INVALID", "invalid")

		reader := envutil.Bool(t.Context(), "TEST_BOOL_INVALID")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestInt(t *testing.T) {
	t.Run("valid int", func(t *testing.T) {
		t.Setenv("TEST_INT", "42")

		reader := envutil.Int[int](t.Context(), "TEST_INT")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, 42, value)
	})

	t.Run("negative int", func(t *testing.T) {
		t.Setenv("TEST_INT_NEG", "-100")

		reader := envutil.Int[int](t.Context(), "TEST_INT_NEG")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, -100, value)
	})

	t.Run("int64", func(t *testing.T) {
		t.Setenv("TEST_INT64", "9223372036854775807")

		reader := envutil.Int[int64](t.Context(), "TEST_INT64")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, int64(9223372036854775807), value)
	})

	t.Run("invalid int", func(t *testing.T) {
		t.Setenv("TEST_INT_INVALID", "not-a-number")

		reader := envutil.Int[int](t.Context(), "TEST_INT_INVALID")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestUint(t *testing.T) {
	t.Run("valid uint", func(t *testing.T) {
		t.Setenv("TEST_UINT", "42")

		reader := envutil.Uint[uint](t.Context(), "TEST_UINT")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, uint(42), value)
	})

	t.Run("uint16", func(t *testing.T) {
		t.Setenv("TEST_UINT16", "65535")

		reader := envutil.Uint[uint16](t.Context(), "TEST_UINT16")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, uint16(65535), value)
	})

	t.Run("negative value", func(t *testing.T) {
		t.Setenv("TEST_UINT_NEG", "-1")

		reader := envutil.Uint[uint](t.Context(), "TEST_UINT_NEG")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestFloat64(t *testing.T) {
	t.Run("valid float", func(t *testing.T) {
		t.Setenv("TEST_FLOAT64", "3.14159")

		reader := envutil.Float64(t.Context(), "TEST_FLOAT64")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.InDelta(t, 3.14159, value, 0.00001)
	})

	t.Run("scientific notation", func(t *testing.T) {
		t.Setenv("TEST_FLOAT64_SCI", "1.23e-4")

		reader := envutil.Float64(t.Context(), "TEST_FLOAT64_SCI")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.InDelta(t, 0.000123, value, 0.0000001)
	})
}

func TestFloat32(t *testing.T) {
	t.Run("valid float", func(t *testing.T) {
		t.Setenv("TEST_FLOAT32", "3.14")

		reader := envutil.Float32(t.Context(), "TEST_FLOAT32")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.InDelta(t, float32(3.14), value, 0.01)
	})
}

func TestDuration(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{"milliseconds", "300ms", 300 * time.Millisecond},
		{"seconds", "5s", 5 * time.Second},
		{"minutes", "10m", 10 * time.Minute},
		{"hours", "2h", 2 * time.Hour},
		{"combined", "1h30m", 90 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_DURATION_" + tt.name
			t.Setenv(key, tt.value)

			reader := envutil.Duration(t.Context(), key)
			value, err := reader.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestTime(t *testing.T) {
	t.Run("RFC3339 format", func(t *testing.T) {
		timeStr := "2024-01-15T10:30:00Z"
		t.Setenv("TEST_TIME", timeStr)

		reader := envutil.Time(t.Context(), "TEST_TIME", time.RFC3339)
		value, err := reader.Value()
		require.NoError(t, err)

		expected, _ := time.Parse(time.RFC3339, timeStr)
		assert.Equal(t, expected, value)
	})

	t.Run("custom format", func(t *testing.T) {
		t.Setenv("TEST_TIME_CUSTOM", "2024-01-15")

		reader := envutil.Time(t.Context(), "TEST_TIME_CUSTOM", "2006-01-02")
		value, err := reader.Value()
		require.NoError(t, err)

		expected, _ := time.Parse("2006-01-02", "2024-01-15")
		assert.Equal(t, expected, value)
	})
}

func TestPort(t *testing.T) {
	t.Run("valid port", func(t *testing.T) {
		t.Setenv("TEST_PORT", "8080")

		reader := envutil.Port(t.Context(), "TEST_PORT")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, uint16(8080), value)
	})

	t.Run("port 0", func(t *testing.T) {
		t.Setenv("TEST_PORT_ZERO", "0")

		reader := envutil.Port(t.Context(), "TEST_PORT_ZERO")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, uint16(0), value)
	})

	t.Run("port too large", func(t *testing.T) {
		t.Setenv("TEST_PORT_LARGE", "99999")

		reader := envutil.Port(t.Context(), "TEST_PORT_LARGE")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestHostAndPort(t *testing.T) {
	t.Run("valid host:port", func(t *testing.T) {
		t.Setenv("TEST_HOST_PORT", "localhost:8080")

		reader := envutil.HostAndPort(t.Context(), "TEST_HOST_PORT")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "localhost", value.Host)
		assert.Equal(t, uint16(8080), value.Port)
	})

	t.Run("domain with port", func(t *testing.T) {
		t.Setenv("TEST_HOST_PORT_DOMAIN", "example.com:443")

		reader := envutil.HostAndPort(t.Context(), "TEST_HOST_PORT_DOMAIN")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "example.com", value.Host)
		assert.Equal(t, uint16(443), value.Port)
	})
}

func TestURL(t *testing.T) {
	t.Run("valid URL", func(t *testing.T) {
		t.Setenv("TEST_URL", "https://example.com/path")

		reader := envutil.URL(t.Context(), "TEST_URL")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "https", value.Scheme)
		assert.Equal(t, "example.com", value.Host)
		assert.Equal(t, "/path", value.Path)
	})

	t.Run("URL with query params", func(t *testing.T) {
		t.Setenv("TEST_URL_QUERY", "https://example.com?foo=bar&baz=qux")

		reader := envutil.URL(t.Context(), "TEST_URL_QUERY")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "bar", value.Query().Get("foo"))
		assert.Equal(t, "qux", value.Query().Get("baz"))
	})

	t.Run("invalid URL", func(t *testing.T) {
		t.Setenv("TEST_URL_INVALID", "://invalid")

		reader := envutil.URL(t.Context(), "TEST_URL_INVALID")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestUUID(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		uuidStr := "550e8400-e29b-41d4-a716-446655440000"
		t.Setenv("TEST_UUID", uuidStr)

		reader := envutil.UUID(t.Context(), "TEST_UUID")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, uuidStr, value.String())
	})

	t.Run("invalid UUID", func(t *testing.T) {
		t.Setenv("TEST_UUID_INVALID", "not-a-uuid")

		reader := envutil.UUID(t.Context(), "TEST_UUID_INVALID")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected slog.Level
	}{
		{"debug lowercase", "debug", slog.LevelDebug},
		{"debug uppercase", "DEBUG", slog.LevelDebug},
		{"info", "info", slog.LevelInfo},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_SLOG_" + tt.name
			t.Setenv(key, tt.value)

			reader := envutil.SlogLevel(t.Context(), key)
			value, err := reader.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestGzipLevel(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected int
	}{
		{"default", "-1", gzip.DefaultCompression},
		{"no compression", "0", gzip.NoCompression},
		{"best speed", "1", gzip.BestSpeed},
		{"best compression", "9", gzip.BestCompression},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_GZIP_" + tt.name
			t.Setenv(key, tt.value)

			reader := envutil.GzipLevel(t.Context(), key)
			value, err := reader.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}

	t.Run("invalid level", func(t *testing.T) {
		t.Setenv("TEST_GZIP_INVALID", "100")

		reader := envutil.GzipLevel(t.Context(), "TEST_GZIP_INVALID")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestFileContents(t *testing.T) {
	t.Run("read existing file", func(t *testing.T) {
		// Create a temporary file
		tmpfile, err := os.CreateTemp(t.TempDir(), "test-*.txt")
		require.NoError(t, err)

		defer func() {
			_ = os.Remove(tmpfile.Name())
		}()

		content := []byte("test content")
		_, err = tmpfile.Write(content)
		require.NoError(t, err)

		_ = tmpfile.Close()

		t.Setenv("TEST_FILE_CONTENTS", tmpfile.Name())

		reader := envutil.FileContents(t.Context(), "TEST_FILE_CONTENTS")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, content, value)
	})

	t.Run("file not found", func(t *testing.T) {
		t.Setenv("TEST_FILE_MISSING", "/nonexistent/file.txt")

		reader := envutil.FileContents(t.Context(), "TEST_FILE_MISSING")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestMany(t *testing.T) {
	t.Run("multiple keys", func(t *testing.T) {
		t.Setenv("TEST_MANY_1", "value1")
		t.Setenv("TEST_MANY_2", "value2")

		readers := envutil.Many(t.Context(), "TEST_MANY_1", "TEST_MANY_2", "TEST_MANY_3")
		assert.Len(t, readers, 3)

		val1, err := readers["TEST_MANY_1"].Value()
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := readers["TEST_MANY_2"].Value()
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)

		_, err = readers["TEST_MANY_3"].Value()
		require.Error(t, err)
	})

	t.Run("empty keys", func(t *testing.T) {
		t.Parallel()

		readers := envutil.Many(t.Context())
		assert.Nil(t, readers)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestVarMap(t *testing.T) {
	t.Run("mixed present and missing", func(t *testing.T) {
		t.Setenv("TEST_VAR_MAP_1", "value1")
		t.Setenv("TEST_VAR_MAP_2", "value2")

		reader := envutil.VarMap(t.Context(), "TEST_VAR_MAP_1", "TEST_VAR_MAP_2", "TEST_VAR_MAP_3")
		value, err := reader.Value()
		require.NoError(t, err)

		assert.Len(t, value, 2)
		assert.Equal(t, "value1", value["TEST_VAR_MAP_1"])
		assert.Equal(t, "value2", value["TEST_VAR_MAP_2"])
		assert.NotContains(t, value, "TEST_VAR_MAP_3")
	})

	t.Run("empty keys", func(t *testing.T) {
		t.Parallel()

		reader := envutil.VarMap(t.Context())
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Empty(t, value)
	})
}

func TestNewReader(t *testing.T) {
	t.Parallel()

	t.Run("with value", func(t *testing.T) {
		t.Parallel()

		reader := envutil.NewReader("TEST_KEY", true, nil, "test-value")
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "test-value", value)
		assert.Equal(t, "TEST_KEY", reader.Key())
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		testErr := assert.AnError
		reader := envutil.NewReader("TEST_KEY", true, testErr, "")
		_, err := reader.Value()
		require.Error(t, err)
	})

	t.Run("not present", func(t *testing.T) {
		t.Parallel()

		reader := envutil.NewReader("TEST_KEY", false, nil, "")
		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestNoValue(t *testing.T) {
	t.Parallel()

	reader := envutil.NoValue[string]()
	assert.False(t, reader.HasValue())
	_, err := reader.Value()
	require.Error(t, err)
}

func TestValidate(t *testing.T) {
	t.Run("validation passes", func(t *testing.T) {
		t.Setenv("TEST_VALIDATE", "5")

		reader := envutil.Int[int](t.Context(), "TEST_VALIDATE", envutil.Validate(func(v int) error {
			if v > 0 && v < 10 {
				return nil
			}

			return assert.AnError
		}))

		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, 5, value)
	})

	t.Run("validation fails", func(t *testing.T) {
		t.Setenv("TEST_VALIDATE_FAIL", "15")

		reader := envutil.Int[int](t.Context(), "TEST_VALIDATE_FAIL", envutil.Validate(func(v int) error {
			if v > 0 && v < 10 {
				return nil
			}

			return assert.AnError
		}))

		_, err := reader.Value()
		require.Error(t, err)
	})
}

func TestFallback(t *testing.T) {
	t.Run("fallback used when missing", func(t *testing.T) {
		t.Setenv("TEST_FALLBACK_B", "fallback-value")

		fallbackReader := envutil.String(t.Context(), "TEST_FALLBACK_B")
		reader := envutil.String(t.Context(), "TEST_FALLBACK_A", envutil.Fallback(fallbackReader))

		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "fallback-value", value)
	})

	t.Run("primary used when present", func(t *testing.T) {
		t.Setenv("TEST_FALLBACK_PRIMARY", "primary-value")
		t.Setenv("TEST_FALLBACK_SECONDARY", "fallback-value")

		fallbackReader := envutil.String(t.Context(), "TEST_FALLBACK_SECONDARY")
		reader := envutil.String(t.Context(), "TEST_FALLBACK_PRIMARY", envutil.Fallback(fallbackReader))

		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "primary-value", value)
	})
}

//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestIfMissing(t *testing.T) {
	t.Run("custom error when missing", func(t *testing.T) {
		t.Parallel()

		reader := envutil.String(t.Context(), "TEST_IF_MISSING", envutil.IfMissing[string](assert.AnError))
		_, err := reader.Value()
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("no error when present", func(t *testing.T) {
		t.Setenv("TEST_IF_MISSING_PRESENT", "value")

		reader := envutil.String(t.Context(), "TEST_IF_MISSING_PRESENT", envutil.IfMissing[string](assert.AnError))
		value, err := reader.Value()
		require.NoError(t, err)
		assert.Equal(t, "value", value)
	})
}
