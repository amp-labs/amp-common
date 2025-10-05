// Package envutil provides type-safe environment variable parsing with a fluent API.
// It offers built-in support for strings, integers, booleans, durations, URLs, UUIDs,
// file paths, and more, with optional defaults, validation, and error handling.
package envutil

import (
	"compress/gzip"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/amp-labs/amp-common/envtypes"
	"github.com/amp-labs/amp-common/tuple"
	"github.com/amp-labs/amp-common/xform"
	"github.com/google/uuid"
)

// get returns a Reader for the given environment variable key.
func get(key string) Reader[string] {
	val, ok := os.LookupEnv(key)

	return Reader[string]{
		key:     key,
		present: ok,
		value:   val,
	}
}

// NewReader creates a Reader from raw values instead of environment variables.
// Useful when you want Reader's fluent API and error handling but with
// data from a different source.
func NewReader[T any](key string, present bool, err error, value T) Reader[T] {
	return Reader[T]{
		key:     key,
		present: present,
		value:   value,
		err:     err,
	}
}

// NoValue returns an empty Reader with no value. Useful as a placeholder
// or when constructing Readers programmatically.
func NoValue[T any]() Reader[T] {
	return Reader[T]{
		key:     "",
		present: false,
	}
}

// String returns a Reader for the given environment variable key.
func String(key string, opts ...Option[string]) Reader[string] {
	rdr := get(key)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Bytes returns a Reader for a byte slice environment variable.
func Bytes(key string, opts ...Option[[]byte]) Reader[[]byte] {
	rdr := Map(get(key), xform.Bytes)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Bool returns a Reader for a boolean environment variable.
// Accepts: "true", "false", "1", "0", "yes", "no" (case-insensitive).
func Bool(key string, opts ...Option[bool]) Reader[bool] {
	rdr := Map(get(key), xform.Bool)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Int returns a Reader for an integer environment variable.
// Supports all signed integer types: int, int8, int16, int32, int64.
func Int[I xform.Intish](key string, opts ...Option[I]) Reader[I] {
	rdr := Map(Map(get(key), xform.Int64), xform.CastNumeric[int64, I])
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Uint returns a Reader for an unsigned integer environment variable.
// Supports all unsigned integer types: uint, uint8, uint16, uint32, uint64.
func Uint[U xform.Uintish](key string, opts ...Option[U]) Reader[U] {
	rdr := Map(Map(get(key), xform.Uint64), xform.CastNumeric[uint64, U])
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Float64 returns a Reader for a float64 environment variable.
func Float64(key string, opts ...Option[float64]) Reader[float64] {
	rdr := Map(get(key), xform.Float64)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Float32 returns a Reader for a float32 environment variable.
func Float32(key string, opts ...Option[float32]) Reader[float32] {
	rdr := Map(Map(get(key), xform.Float64), xform.Float32)

	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Duration returns a Reader for a time.Duration environment variable.
// Accepts formats like "300ms", "1.5h", "2h45m".
func Duration(key string, opts ...Option[time.Duration]) Reader[time.Duration] {
	rdr := Map(get(key), xform.Duration)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Time returns a Reader for a time.Time environment variable.
// The format parameter specifies the expected time format (e.g., time.RFC3339).
func Time(key string, format string, opts ...Option[time.Time]) Reader[time.Time] {
	rdr := Map(get(key), xform.Time(format))
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// Port returns a Reader for a network port environment variable.
// Valid range: 1-65535.
func Port(key string, opts ...Option[uint16]) Reader[uint16] {
	rdr := Map(get(key), xform.Port)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// HostAndPort returns a Reader for a host:port environment variable.
// Expected format: "hostname:port" (e.g., "localhost:8080", "db.example.com:5432").
func HostAndPort(key string, opts ...Option[envtypes.HostPort]) Reader[envtypes.HostPort] {
	rdr := Map(get(key), xform.HostAndPort)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// URL returns a Reader for a URL environment variable.
// Parses the value using url.Parse.
func URL(key string, opts ...Option[*url.URL]) Reader[*url.URL] {
	rdr := Map(get(key), xform.URL)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// UUID returns a Reader for a UUID environment variable.
// Accepts standard UUID format: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx".
func UUID(key string, opts ...Option[uuid.UUID]) Reader[uuid.UUID] {
	rdr := Map(get(key), xform.UUID)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// SlogLevel returns a Reader for a slog.Level environment variable.
// Accepts: "debug", "info", "warn", "error" (case-insensitive).
func SlogLevel(key string, opts ...Option[slog.Level]) Reader[slog.Level] {
	rdr := Map(Map(Map(get(key), xform.TrimString), xform.ToLower), xform.SlogLevel)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// FilePath returns a Reader for a file path environment variable.
// Validates that the path points to an existing file (not a directory).
func FilePath(key string, opts ...Option[envtypes.LocalPath]) Reader[envtypes.LocalPath] {
	rdr := Map(Map(get(key), xform.Path), xform.PathIsFile)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// FileContents returns a Reader that reads file contents from a path.
// The environment variable value is treated as a file path, which is read into memory.
// Note: Default() provides default file contents, not a default file path.
func FileContents(key string, opts ...Option[[]byte]) Reader[[]byte] {
	rdr := Map(Map(Map(get(key), xform.Path), xform.PathExists), xform.ReadFile)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// GzipLevel returns a Reader for a gzip compression level environment variable.
// Valid values: gzip.DefaultCompression, gzip.BestSpeed, gzip.BestCompression,
// gzip.NoCompression, gzip.HuffmanOnly.
func GzipLevel(key string, opts ...Option[int]) Reader[int] {
	rdr := Map(get(key), xform.GzipLevel)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	// Add a sanity check since the caller might pass in an invalid default
	return rdr.Map(func(i int) (int, error) {
		switch i {
		case gzip.DefaultCompression, gzip.BestSpeed,
			gzip.BestCompression, gzip.NoCompression, gzip.HuffmanOnly:
			return i, nil
		default:
			return 0, fmt.Errorf("%w: %d", xform.ErrInvalidGzipLevel, i)
		}
	})
}

// Many returns a map of Readers for multiple environment variable keys.
// Useful when you need to process several related variables.
func Many(keys ...string) map[string]Reader[string] {
	if len(keys) == 0 {
		return nil
	}

	out := make(map[string]Reader[string], len(keys))

	for _, key := range keys {
		out[key] = get(key)
	}

	return out
}

// VarMap returns a Reader containing a map of environment variable values.
// Only variables that are present in the environment are included in the map.
// All variables are treated as optional; missing variables are simply omitted.
func VarMap(keys ...string) Reader[map[string]string] {
	if len(keys) == 0 {
		return NewReader[map[string]string]("", true, nil, make(map[string]string))
	}

	out := make(map[string]string, len(keys))

	for _, rdr := range Many(keys...) {
		if rdr.HasValue() {
			out[rdr.Key()] = rdr.value
		}
	}

	return NewReader("", true, nil, out)
}

// String2 returns a Reader containing a tuple of 2 environment variables.
// All-or-nothing: if any variable is missing, the entire Reader is missing.
// For more flexibility, use individual Readers or VarMap.
func String2(
	key1 string,
	key2 string,
	opts ...Option[string],
) Reader[tuple.Tuple2[string, string]] {
	return Combine2(
		String(key1, opts...),
		String(key2, opts...))
}

// String3 returns a Reader containing a tuple of 3 environment variables.
// All-or-nothing: if any variable is missing, the entire Reader is missing.
// For more flexibility, use individual Readers or VarMap.
func String3(
	key1 string,
	key2 string,
	key3 string,
	opts ...Option[string],
) Reader[tuple.Tuple3[string, string, string]] {
	return Combine3(
		String(key1, opts...),
		String(key2, opts...),
		String(key3, opts...))
}

// String4 returns a Reader containing a tuple of 4 environment variables.
// All-or-nothing: if any variable is missing, the entire Reader is missing.
// For more flexibility, use individual Readers or VarMap.
func String4(
	key1 string,
	key2 string,
	key3 string,
	key4 string,
	opts ...Option[string],
) Reader[tuple.Tuple4[string, string, string, string]] {
	return Combine4(
		String(key1, opts...),
		String(key2, opts...),
		String(key3, opts...),
		String(key4, opts...))
}

// String5 returns a Reader containing a tuple of 5 environment variables.
// All-or-nothing: if any variable is missing, the entire Reader is missing.
// For more flexibility, use individual Readers or VarMap.
func String5(
	key1 string,
	key2 string,
	key3 string,
	key4 string,
	key5 string,
	opts ...Option[string],
) Reader[tuple.Tuple5[string, string, string, string, string]] {
	return Combine5(
		String(key1, opts...),
		String(key2, opts...),
		String(key3, opts...),
		String(key4, opts...),
		String(key5, opts...))
}

// String6 returns a Reader containing a tuple of 6 environment variables.
// All-or-nothing: if any variable is missing, the entire Reader is missing.
// For more flexibility, use individual Readers or VarMap.
func String6(
	key1 string,
	key2 string,
	key3 string,
	key4 string,
	key5 string,
	key6 string,
	opts ...Option[string],
) Reader[tuple.Tuple6[string, string, string, string, string, string]] {
	return Combine6(
		String(key1, opts...),
		String(key2, opts...),
		String(key3, opts...),
		String(key4, opts...),
		String(key5, opts...),
		String(key6, opts...))
}
