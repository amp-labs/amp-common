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

// NewReader returns a Reader for the given raw data. If you feel like
// you want to branch out from using the environment variables directly,
// this will provide the same functionality - except you need to provide
// the initial values yourself.
func NewReader[T any](key string, present bool, err error, value T) Reader[T] {
	return Reader[T]{
		key:     key,
		present: present,
		value:   value,
		err:     err,
	}
}

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

func Bytes(key string, opts ...Option[[]byte]) Reader[[]byte] {
	rdr := Map(get(key), xform.Bytes)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Bool(key string, opts ...Option[bool]) Reader[bool] {
	rdr := Map(get(key), xform.Bool)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Int[I xform.Intish](key string, opts ...Option[I]) Reader[I] {
	rdr := Map(Map(get(key), xform.Int64), xform.CastNumeric[int64, I])
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Uint[U xform.Uintish](key string, opts ...Option[U]) Reader[U] {
	rdr := Map(Map(get(key), xform.Uint64), xform.CastNumeric[uint64, U])
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Float64(key string, opts ...Option[float64]) Reader[float64] {
	rdr := Map(get(key), xform.Float64)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Float32(key string, opts ...Option[float32]) Reader[float32] {
	rdr := Map(Map(get(key), xform.Float64), xform.Float32)

	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Duration(key string, opts ...Option[time.Duration]) Reader[time.Duration] {
	rdr := Map(get(key), xform.Duration)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Time(key string, format string, opts ...Option[time.Time]) Reader[time.Time] {
	rdr := Map(get(key), xform.Time(format))
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func Port(key string, opts ...Option[uint16]) Reader[uint16] {
	rdr := Map(get(key), xform.Port)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// HostAndPort returns a Reader for the given environment variable key.
// The expected format in the env is "host:port".
func HostAndPort(key string, opts ...Option[envtypes.HostPort]) Reader[envtypes.HostPort] {
	rdr := Map(get(key), xform.HostAndPort)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// URL returns a Reader for the given environment variable key.
func URL(key string, opts ...Option[*url.URL]) Reader[*url.URL] {
	rdr := Map(get(key), xform.URL)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

func UUID(key string, opts ...Option[uuid.UUID]) Reader[uuid.UUID] {
	rdr := Map(get(key), xform.UUID)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// SlogLevel returns a Reader for the given environment variable key.
func SlogLevel(key string, opts ...Option[slog.Level]) Reader[slog.Level] {
	rdr := Map(Map(Map(get(key), xform.TrimString), xform.ToLower), xform.SlogLevel)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// FilePath returns a Reader for the given environment variable key.
func FilePath(key string, opts ...Option[envtypes.LocalPath]) Reader[envtypes.LocalPath] {
	rdr := Map(Map(get(key), xform.Path), xform.PathIsFile)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// FileContents returns a Reader for the given environment variable key.
// The value will be treated as a file path, and that file will be read in
// to memory. Note that the default you provide isn't a default file path,
// but rather actual file contents to use if key isn't provided.
func FileContents(key string, opts ...Option[[]byte]) Reader[[]byte] {
	rdr := Map(Map(Map(get(key), xform.Path), xform.PathExists), xform.ReadFile)
	for _, opt := range opts {
		rdr = opt(rdr)
	}

	return rdr
}

// GzipLevel returns a Reader for the given environment variable key.
// The value will be treated as a gzip compression level.
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

// Many returns a map of Readers for the given environment variable keys.
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

// VarMap returns a map of Readers for the given environment variable keys.
// All variables are treated as optional, and if they are missing, it means
// they were never set in the environment. Enforcement is done by the caller.
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

// String2 returns a Reader for the given environment variable keys.
// Note that this is all or nothing, so if one of the keys is missing,
// the entire Reader will be missing. If you need more flexibility,
// either use the individual Readers, or use the VarMap function.
func String2(
	key1 string,
	key2 string,
	opts ...Option[string],
) Reader[tuple.Tuple2[string, string]] {
	return Combine2(
		String(key1, opts...),
		String(key2, opts...))
}

// String3 returns a Reader for the given environment variable keys.
// Note that this is all or nothing, so if one of the keys is missing,
// the entire Reader will be missing. If you need more flexibility,
// either use the individual Readers, or use the VarMap function.
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

// String4 returns a Reader for the given environment variable keys.
// Note that this is all or nothing, so if one of the keys is missing,
// the entire Reader will be missing. If you need more flexibility,
// either use the individual Readers, or use the VarMap function.
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

// String5 returns a Reader for the given environment variable keys.
// Note that this is all or nothing, so if one of the keys is missing,
// the entire Reader will be missing. If you need more flexibility,
// either use the individual Readers, or use the VarMap function.
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

// String6 returns a Reader for the given environment variable keys.
// Note that this is all or nothing, so if one of the keys is missing,
// the entire Reader will be missing. If you need more flexibility,
// either use the individual Readers, or use the VarMap function.
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
