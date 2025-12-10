package xform_test

import (
	"compress/gzip"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/envtypes"
	"github.com/amp-labs/amp-common/xform"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrimString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no whitespace", "hello", "hello"},
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"both sides", "  hello  ", "hello"},
		{"tabs", "\t\thello\t\t", "hello"},
		{"newlines", "\n\nhello\n\n", "hello"},
		{"empty", "", ""},
		{"only whitespace", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.TrimString(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sep      string
		input    string
		expected []string
	}{
		{"comma separator", ",", "a,b,c", []string{"a", "b", "c"}},
		{"space separator", " ", "a b c", []string{"a", "b", "c"}},
		{"multi-char separator", "::", "a::b::c", []string{"a", "b", "c"}},
		{"no separator found", ",", "abc", []string{"abc"}},
		{"empty string", ",", "", []string{""}},
		{"trailing separator", ",", "a,b,", []string{"a", "b", ""}},
		{"leading separator", ",", ",a,b", []string{"", "a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			splitter := xform.SplitString(tt.sep)
			result, err := splitter(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeyify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []string
		expected map[string]struct{}
	}{
		{
			name:  "basic list",
			input: []string{"a", "b", "c"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
			},
		},
		{
			name:  "duplicates",
			input: []string{"a", "b", "a"},
			expected: map[string]struct{}{
				"a": {},
				"b": {},
			},
		},
		{
			name:     "empty list",
			input:    []string{},
			expected: map[string]struct{}{},
		},
		{
			name:     "nil list",
			input:    nil,
			expected: map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Keyify(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"basic string", []byte("hello"), "hello"},
		{"empty bytes", []byte{}, ""},
		{"nil bytes", nil, ""},
		//nolint:gosmopolitan // Intentional test data for unicode handling
		{"unicode", []byte("hello 世界"), "hello 世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.String(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{"basic string", "hello", []byte("hello")},
		{"empty string", "", []byte{}},
		//nolint:gosmopolitan // Intentional test data for unicode handling
		{"unicode", "hello 世界", []byte("hello 世界")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Bytes(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "HELLO", "hello"},
		{"mixed case", "HeLLo", "hello"},
		{"lowercase", "hello", "hello"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.ToLower(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToUpper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "hello", "HELLO"},
		{"mixed case", "HeLLo", "HELLO"},
		{"uppercase", "HELLO", "HELLO"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.ToUpper(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		oldStr   string
		newStr   string
		input    string
		expected string
	}{
		{"basic replacement", "foo", "bar", "foo baz foo", "bar baz bar"},
		{"no match", "foo", "bar", "baz qux", "baz qux"},
		{"empty old string", "", "x", "abc", "xaxbxcx"},
		{"empty new string", "foo", "", "foo bar foo", " bar "},
		{"multi-char", "hello", "goodbye", "hello world hello", "goodbye world goodbye"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			replacer := xform.ReplaceAll(tt.oldStr, tt.newStr)
			result, err := replacer(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOneOf(t *testing.T) {
	t.Parallel()

	t.Run("string valid choice", func(t *testing.T) {
		t.Parallel()

		validator := xform.OneOf("red", "green", "blue")
		result, err := validator("green")
		require.NoError(t, err)
		assert.Equal(t, "green", result)
	})

	t.Run("string invalid choice", func(t *testing.T) {
		t.Parallel()

		validator := xform.OneOf("red", "green", "blue")
		_, err := validator("yellow")
		assert.ErrorIs(t, err, xform.ErrInvalidChoice)
	})

	t.Run("int valid choice", func(t *testing.T) {
		t.Parallel()

		validator := xform.OneOf(1, 2, 3)
		result, err := validator(2)
		require.NoError(t, err)
		assert.Equal(t, 2, result)
	})

	t.Run("int invalid choice", func(t *testing.T) {
		t.Parallel()

		validator := xform.OneOf(1, 2, 3)
		_, err := validator(4)
		assert.ErrorIs(t, err, xform.ErrInvalidChoice)
	})

	t.Run("empty choices", func(t *testing.T) {
		t.Parallel()

		validator := xform.OneOf[string]()
		_, err := validator("anything")
		assert.ErrorIs(t, err, xform.ErrInvalidChoice)
	})
}

func TestBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  bool
		wantError bool
	}{
		{"true lowercase", "true", true, false},
		{"true uppercase", "TRUE", true, false},
		{"true mixed", "True", true, false},
		{"t", "t", true, false},
		{"T", "T", true, false},
		{"1", "1", true, false},
		{"false lowercase", "false", false, false},
		{"false uppercase", "FALSE", false, false},
		{"false mixed", "False", false, false},
		{"f", "f", false, false},
		{"F", "F", false, false},
		{"0", "0", false, false},
		{"invalid", "invalid", false, true},
		{"empty", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Bool(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  int64
		wantError bool
	}{
		{"zero", "0", 0, false},
		{"positive", "123", 123, false},
		{"negative", "-456", -456, false},
		{"max int64", "9223372036854775807", 9223372036854775807, false},
		{"min int64", "-9223372036854775808", -9223372036854775808, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
		{"float", "3.14", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Int64(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUint64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  uint64
		wantError bool
	}{
		{"zero", "0", 0, false},
		{"positive", "123", 123, false},
		{"max uint64", "18446744073709551615", 18446744073709551615, false},
		{"negative", "-1", 0, true},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Uint64(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  float64
		wantError bool
	}{
		{"zero", "0", 0, false},
		{"positive int", "123", 123, false},
		{"negative int", "-456", -456, false},
		{"positive float", "3.14", 3.14, false},
		{"negative float", "-2.71", -2.71, false},
		{"scientific notation", "1.23e5", 123000, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Float64(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.0001)
			}
		})
	}
}

func TestFloat32(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    float64
		expected float32
	}{
		{"zero", 0, 0},
		{"positive", 3.14, 3.14},
		{"negative", -2.71, -2.71},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Float32(tt.input)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

func TestPositive(t *testing.T) {
	t.Parallel()

	t.Run("int positive", func(t *testing.T) {
		t.Parallel()

		result, err := xform.Positive(5)
		require.NoError(t, err)
		assert.Equal(t, 5, result)
	})

	t.Run("int zero", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(0)
		assert.ErrorIs(t, err, xform.ErrNonPositive)
	})

	t.Run("int negative", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(-5)
		assert.ErrorIs(t, err, xform.ErrNonPositive)
	})

	t.Run("float positive", func(t *testing.T) {
		t.Parallel()

		result, err := xform.Positive(3.14)
		require.NoError(t, err)
		assert.InDelta(t, 3.14, result, 0.0001)
	})

	t.Run("float zero", func(t *testing.T) {
		t.Parallel()

		_, err := xform.Positive(0.0)
		assert.ErrorIs(t, err, xform.ErrNonPositive)
	})

	t.Run("duration positive", func(t *testing.T) {
		t.Parallel()

		result, err := xform.Positive(time.Second)
		require.NoError(t, err)
		assert.Equal(t, time.Second, result)
	})
}

func TestGzipLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  int
		wantError bool
	}{
		{"default", "default", gzip.DefaultCompression, false},
		{"best-speed hyphen", "best-speed", gzip.BestSpeed, false},
		{"best_speed underscore", "best_speed", gzip.BestSpeed, false},
		{"best-compression hyphen", "best-compression", gzip.BestCompression, false},
		{"best_compression underscore", "best_compression", gzip.BestCompression, false},
		{"no-compression hyphen", "no-compression", gzip.NoCompression, false},
		{"no_compression underscore", "no_compression", gzip.NoCompression, false},
		{"none", "none", gzip.NoCompression, false},
		{"huffman-only hyphen", "huffman-only", gzip.HuffmanOnly, false},
		{"huffman_only underscore", "huffman_only", gzip.HuffmanOnly, false},
		{"number -1", "-1", gzip.DefaultCompression, false},
		{"number -2", "-2", gzip.HuffmanOnly, false},
		{"number 0", "0", gzip.NoCompression, false},
		{"number 1", "1", gzip.BestSpeed, false},
		{"number 9", "9", gzip.BestCompression, false},
		{"uppercase", "DEFAULT", gzip.DefaultCompression, false},
		{"whitespace", "  default  ", gzip.DefaultCompression, false},
		{"invalid string", "invalid", 0, true},
		{"invalid number", "10", 0, true},
		{"invalid number -3", "-3", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.GzipLevel(tt.input)
			if tt.wantError {
				assert.ErrorIs(t, err, xform.ErrInvalidGzipLevel)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  uint16
		wantError bool
	}{
		{"zero", "0", 0, false},
		{"http", "80", 80, false},
		{"https", "443", 443, false},
		{"max port", "65535", 65535, false},
		{"negative", "-1", 0, true},
		{"too large", "65536", 0, true},
		{"way too large", "100000", 0, true},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Port(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestHostAndPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  envtypes.HostPort
		wantError bool
	}{
		{
			name:  "localhost with port",
			input: "localhost:8080",
			expected: envtypes.HostPort{
				Host: "localhost",
				Port: 8080,
			},
			wantError: false,
		},
		{
			name:  "IP with port",
			input: "127.0.0.1:443",
			expected: envtypes.HostPort{
				Host: "127.0.0.1",
				Port: 443,
			},
			wantError: false,
		},
		{
			name:  "hostname with port",
			input: "example.com:9000",
			expected: envtypes.HostPort{
				Host: "example.com",
				Port: 9000,
			},
			wantError: false,
		},
		{
			name:      "no port",
			input:     "localhost",
			wantError: true,
		},
		{
			name:      "invalid port",
			input:     "localhost:abc",
			wantError: true,
		},
		{
			name:      "port too large",
			input:     "localhost:70000",
			wantError: true,
		},
		{
			name:      "empty",
			input:     "",
			wantError: true,
		},
		{
			name:      "only colon",
			input:     ":",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.HostAndPort(tt.input)
			if tt.wantError {
				require.Error(t, err)
				assert.ErrorIs(t, err, xform.ErrBadHostAndPort)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestURL(t *testing.T) {
	t.Parallel()

	t.Run("parses valid URLs", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			input string
		}{
			{"http URL", "http://example.com"},
			{"https URL", "https://example.com"},
			{"URL with path", "https://example.com/path/to/resource"},
			{"URL with query", "https://example.com?foo=bar"},
			{"relative URL", "/path/to/resource"},
			{"empty", ""}, // url.Parse allows empty strings
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := xform.URL(tt.input)
				require.NoError(t, err)
				assert.NotNil(t, result)
			})
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		t.Parallel()

		_, err := xform.URL("://invalid")
		assert.Error(t, err)
	})

	t.Run("parses https URL correctly", func(t *testing.T) {
		t.Parallel()

		result, err := xform.URL("https://example.com")
		require.NoError(t, err)
		assert.Equal(t, "example.com", result.Host)
		assert.Equal(t, "https", result.Scheme)
	})
}

func TestUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  uuid.UUID
		wantError bool
	}{
		{
			name:      "valid UUID with hyphens",
			input:     "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			expected:  uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			wantError: false,
		},
		{
			name:      "valid UUID without hyphens",
			input:     "6ba7b8109dad11d180b400c04fd430c8",
			expected:  uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
			wantError: false,
		},
		{
			name:      "nil UUID",
			input:     "00000000-0000-0000-0000-000000000000",
			expected:  uuid.Nil,
			wantError: false,
		},
		{"invalid format", "invalid", uuid.Nil, true},
		{"empty", "", uuid.Nil, true},
		{"too short", "6ba7b810", uuid.Nil, true}, // typos:disable-line (partial UUID hex string)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.UUID(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestPath(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		// Create a temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		result, err := xform.Path(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, tmpFile, result.Path)
		assert.NotNil(t, result.Info)
		assert.False(t, result.Info.IsDir())
	})

	t.Run("existing directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		result, err := xform.Path(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, result.Path)
		assert.NotNil(t, result.Info)
		assert.True(t, result.Info.IsDir())
	})

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist")

		result, err := xform.Path(nonExistent)
		require.NoError(t, err)
		assert.Equal(t, nonExistent, result.Path)
		assert.Nil(t, result.Info)
	})
}

func TestPathExists(t *testing.T) {
	t.Parallel()

	t.Run("existing path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		result, err := xform.PathExists(path)
		require.NoError(t, err)
		assert.Equal(t, tmpFile, result.Path)
	})

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.PathExists(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestPathNotExists(t *testing.T) {
	t.Parallel()

	t.Run("non-existent path", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		result, err := xform.PathNotExists(path)
		require.NoError(t, err)
		assert.Equal(t, "/does-not-exist", result.Path)
	})

	t.Run("existing path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		_, err = xform.PathNotExists(path)
		assert.ErrorIs(t, err, os.ErrExist)
	})
}

func TestPathIsFile(t *testing.T) {
	t.Parallel()

	t.Run("is a file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		result, err := xform.PathIsFile(path)
		require.NoError(t, err)
		assert.Equal(t, tmpFile, result.Path)
	})

	t.Run("is a directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		path, err := xform.Path(tmpDir)
		require.NoError(t, err)

		_, err = xform.PathIsFile(path)
		assert.ErrorIs(t, err, xform.ErrNotAFile)
	})

	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.PathIsFile(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestPathIsNonEmptyFile(t *testing.T) {
	t.Parallel()

	t.Run("non-empty file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		result, err := xform.PathIsNonEmptyFile(path)
		require.NoError(t, err)
		assert.Equal(t, tmpFile, result.Path)
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(tmpFile, []byte{}, 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		_, err = xform.PathIsNonEmptyFile(path)
		assert.ErrorIs(t, err, xform.ErrEmptyFile)
	})

	t.Run("is a directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		path, err := xform.Path(tmpDir)
		require.NoError(t, err)

		_, err = xform.PathIsNonEmptyFile(path)
		assert.ErrorIs(t, err, xform.ErrNotAFile)
	})

	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.PathIsNonEmptyFile(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestPathIsDir(t *testing.T) {
	t.Parallel()

	t.Run("is a directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		path, err := xform.Path(tmpDir)
		require.NoError(t, err)

		result, err := xform.PathIsDir(path)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, result.Path)
	})

	t.Run("is a file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(tmpFile, []byte("test"), 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		_, err = xform.PathIsDir(path)
		assert.ErrorIs(t, err, xform.ErrNotADir)
	})

	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.PathIsDir(path)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestOpenFile(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		err := os.WriteFile(tmpFile, content, 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		file, err := xform.OpenFile(path)
		require.NoError(t, err)

		defer func() { _ = file.Close() }()

		data, err := os.ReadFile(file.Name())
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.OpenFile(path)
		assert.Error(t, err)
	})
}

func TestReadFile(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		err := os.WriteFile(tmpFile, content, 0o600)
		require.NoError(t, err)

		path, err := xform.Path(tmpFile)
		require.NoError(t, err)

		data, err := xform.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		path := envtypes.LocalPath{
			Path: "/does-not-exist",
			Info: nil,
		}

		_, err := xform.ReadFile(path)
		assert.Error(t, err)
	})
}

func TestDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		wantError bool
	}{
		{"seconds", "5s", 5 * time.Second, false},
		{"minutes", "10m", 10 * time.Minute, false},
		{"hours", "2h", 2 * time.Hour, false},
		{"milliseconds", "100ms", 100 * time.Millisecond, false},
		{"microseconds", "50us", 50 * time.Microsecond, false},
		{"nanoseconds", "1000ns", 1000 * time.Nanosecond, false},
		{"combined", "1h30m", 90 * time.Minute, false},
		{"zero", "0s", 0, false},
		{"invalid", "invalid", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Duration(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		layout    string
		input     string
		wantError bool
	}{
		{"RFC3339", time.RFC3339, "2023-01-15T10:30:00Z", false},
		{"RFC3339 invalid", time.RFC3339, "invalid", true},
		{"custom format", "2006-01-02", "2023-01-15", false},
		{"custom format invalid", "2006-01-02", "15-01-2023", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.Time(tt.layout)(tt.input)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, result)
			}
		})
	}
}

func TestCastNumeric(t *testing.T) {
	t.Parallel()

	t.Run("int64 to int32", func(t *testing.T) {
		t.Parallel()

		result, err := xform.CastNumeric[int64, int32](42)
		require.NoError(t, err)
		assert.Equal(t, int32(42), result)
	})

	t.Run("int to uint", func(t *testing.T) {
		t.Parallel()

		result, err := xform.CastNumeric[int, uint](42)
		require.NoError(t, err)
		assert.Equal(t, uint(42), result)
	})

	t.Run("float64 to float32", func(t *testing.T) {
		t.Parallel()

		result, err := xform.CastNumeric[float64, float32](3.14)
		require.NoError(t, err)
		assert.InDelta(t, float32(3.14), result, 0.0001)
	})

	t.Run("int to float64", func(t *testing.T) {
		t.Parallel()

		result, err := xform.CastNumeric[int, float64](42)
		require.NoError(t, err)
		assert.InDelta(t, float64(42), result, 0.0001)
	})
}

func TestSlogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  slog.Level
		wantError bool
	}{
		{"debug", "debug", slog.LevelDebug, false},
		{"info", "info", slog.LevelInfo, false},
		{"warn", "warn", slog.LevelWarn, false},
		{"error", "error", slog.LevelError, false},
		{"invalid", "invalid", 0, true},
		{"uppercase", "DEBUG", 0, true}, // case-sensitive
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := xform.SlogLevel(tt.input)
			if tt.wantError {
				assert.ErrorIs(t, err, xform.ErrInvalidLogLevel)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
