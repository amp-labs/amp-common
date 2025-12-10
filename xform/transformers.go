package xform

import (
	"compress/gzip"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/amp-labs/amp-common/envtypes"
	"github.com/google/uuid"
)

const portMax = 65535

// TrimString removes leading and trailing whitespace from a string.
func TrimString(s string) (string, error) {
	return strings.TrimSpace(s), nil
}

// SplitString returns a transformer that splits a string by the given separator.
// The separator can be any string, including multi-character separators.
func SplitString(sep string) func(string) ([]string, error) {
	return func(s string) ([]string, error) {
		return strings.Split(s, sep), nil
	}
}

// Keyify converts a slice of strings to a map where the keys are the strings
// from the slice and values are empty structs. This provides an efficient
// way to create a set for membership testing.
func Keyify(strs []string) (map[string]struct{}, error) {
	m := make(map[string]struct{}, len(strs))

	for _, s := range strs {
		m[s] = struct{}{}
	}

	return m, nil
}

// String converts a byte slice to a string.
func String(value []byte) (string, error) {
	return string(value), nil
}

// Bytes converts a string to a byte slice.
func Bytes(value string) ([]byte, error) {
	return []byte(value), nil
}

// ToLower converts a string to lowercase.
func ToLower(s string) (string, error) {
	return strings.ToLower(s), nil
}

// ToUpper converts a string to uppercase.
func ToUpper(s string) (string, error) {
	return strings.ToUpper(s), nil
}

// ReplaceAll returns a transformer that replaces all occurrences of oldStr
// with newStr in the input string.
func ReplaceAll(oldStr, newStr string) func(string) (string, error) {
	return func(s string) (string, error) {
		return strings.ReplaceAll(s, oldStr, newStr), nil
	}
}

// OneOf returns a transformer that validates a value is one of the allowed choices.
// Returns ErrInvalidChoice if the value doesn't match any of the choices.
func OneOf[A comparable](choices ...A) func(A) (A, error) { //nolint:ireturn
	return func(value A) (A, error) {
		if slices.Contains(choices, value) {
			return value, nil
		}

		return value, ErrInvalidChoice
	}
}

// Bool parses a string as a boolean value.
// Accepts: "1", "t", "T", "true", "TRUE", "True", "0", "f", "F", "false", "FALSE", "False".
func Bool(value string) (bool, error) {
	return strconv.ParseBool(value)
}

// Int64 parses a string as a base-10 int64.
func Int64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

// Uint64 parses a string as a base-10 uint64.
func Uint64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

// Float64 parses a string as a float64.
func Float64(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// Float32 converts a float64 to a float32.
// Note: This may lose precision for values outside the float32 range.
func Float32(value float64) (float32, error) {
	return float32(value), nil
}

// Positive validates that a numeric value is greater than zero.
// Returns ErrNonPositive if the value is less than or equal to zero.
func Positive[A Numeric](value A) (A, error) { // nolint:ireturn
	if value <= 0 {
		return value, ErrNonPositive
	}

	return value, nil
}

// GzipLevel represents a transformer that parses the given string as a gzip compression level.
// The string can be one of the following:
//   - "default"
//   - "best-speed" or "best_speed"
//   - "best-compression" or "best_compression"
//   - "no-compression" or "no_compression" or "none"
//   - "huffman-only" or "huffman_only"
//   - the numbers 0, 1, 9, -1, -2 (constants from the underlying library which
//     correspond to the above options)
func GzipLevel(value string) (int, error) {
	value = strings.TrimSpace(strings.ToLower(value))

	switch value {
	case "default":
		return gzip.DefaultCompression, nil
	case "best-speed", "best_speed":
		return gzip.BestSpeed, nil
	case "best-compression", "best_compression":
		return gzip.BestCompression, nil
	case "no-compression", "no_compression", "none":
		return gzip.NoCompression, nil
	case "huffman-only", "huffman_only":
		return gzip.HuffmanOnly, nil
	}

	level, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidGzipLevel, value)
	}

	switch int(level) {
	case gzip.DefaultCompression, gzip.BestSpeed,
		gzip.BestCompression, gzip.NoCompression, gzip.HuffmanOnly:
		return int(level), nil
	default:
		return 0, fmt.Errorf("%w: %d", ErrInvalidGzipLevel, level)
	}
}

// HostAndPort parses a string as a host:port pair.
// The input must be in the format "host:port" where port is a valid port number (0-65535).
// Returns ErrBadHostAndPort if the format is invalid or the port is out of range.
func HostAndPort(value string) (envtypes.HostPort, error) {
	parts := strings.SplitN(value, ":", 2) //nolint:mnd
	if len(parts) != 2 {                   //nolint:mnd
		return envtypes.HostPort{}, fmt.Errorf("%w: %s", ErrBadHostAndPort, value)
	}

	host := parts[0]

	port, err := Port(parts[1])
	if err != nil {
		return envtypes.HostPort{}, fmt.Errorf("%w: %w", ErrBadHostAndPort, err)
	}

	return envtypes.HostPort{Host: host, Port: port}, nil
}

// Port parses a string as a TCP/UDP port number.
// Valid port numbers are in the range 0-65535.
// Returns ErrBadPort if the value is not a valid port number.
func Port(value string) (uint16, error) {
	port, err := Int64(value)
	if err != nil {
		return 0, err
	}

	if port < 0 {
		return 0, fmt.Errorf("%w: %d", ErrBadPort, port)
	}

	if port > portMax {
		return 0, fmt.Errorf("%w: %d", ErrBadPort, port)
	}

	return uint16(port), nil
}

// URL parses a string as a URL using Go's standard url.Parse.
// The URL may be relative or absolute.
func URL(value string) (*url.URL, error) {
	return url.Parse(value)
}

// UUID parses a string as a UUID in any of the formats accepted by github.com/google/uuid.
// Accepts formats like: "6ba7b810-9dad-11d1-80b4-00c04fd430c8" or "6ba7b8109dad11d180b400c04fd430c8".
func UUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

// Path treats the input as a local filesystem path and returns a LocalPath struct.
// The path is stat'ed to gather file information. If the path doesn't exist,
// the LocalPath will have a nil Info field but no error is returned.
// Other stat errors (permission denied, etc.) are returned as errors.
func Path(value string) (envtypes.LocalPath, error) {
	stat, err := os.Stat(value)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return envtypes.LocalPath{
				Path: value,
				Info: nil,
			}, nil
		} else {
			return envtypes.LocalPath{}, err
		}
	}

	return envtypes.LocalPath{
		Path: value,
		Info: stat,
	}, nil
}

// PathExists validates that a path exists on the filesystem.
// Returns os.ErrNotExist if the path does not exist.
func PathExists(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	return value, nil
}

// PathNotExists validates that a path does NOT exist on the filesystem.
// Returns os.ErrExist if the path already exists.
func PathNotExists(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info != nil {
		return value, os.ErrExist
	}

	return value, nil
}

// PathIsFile validates that a path exists and is a regular file (not a directory).
// Returns os.ErrNotExist if the path doesn't exist, or ErrNotAFile if it's a directory.
func PathIsFile(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	if !value.Info.IsDir() {
		return value, nil
	}

	return value, fmt.Errorf("%w: %s", ErrNotAFile, value.Path)
}

// PathIsNonEmptyFile validates that a path exists, is a regular file, and has size > 0.
// Returns os.ErrNotExist if the path doesn't exist, ErrNotAFile if it's a directory,
// or ErrEmptyFile if the file size is zero.
func PathIsNonEmptyFile(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	if value.Info.IsDir() {
		return value, fmt.Errorf("%w: %s", ErrNotAFile, value.Path)
	}

	if value.Info.Size() == 0 {
		return value, fmt.Errorf("%w: %s", ErrEmptyFile, value.Path)
	}

	return value, nil
}

// PathIsDir validates that a path exists and is a directory.
// Returns os.ErrNotExist if the path doesn't exist, or ErrNotADir if it's a file.
func PathIsDir(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	if value.Info.IsDir() {
		return value, nil
	}

	return value, fmt.Errorf("%w: %s", ErrNotADir, value.Path)
}

// OpenFile opens the file at the given path for reading.
// The caller is responsible for closing the returned file.
func OpenFile(value envtypes.LocalPath) (*os.File, error) {
	return os.Open(value.Path)
}

// ReadFile reads and returns the entire contents of the file at the given path.
func ReadFile(value envtypes.LocalPath) ([]byte, error) {
	return os.ReadFile(value.Path)
}

// Duration parses a string as a time.Duration.
// Accepts formats like "1h30m", "5s", "100ms", etc. as defined by time.ParseDuration.
func Duration(value string) (time.Duration, error) {
	return time.ParseDuration(value)
}

// Time returns a transformer that parses a string as a time.Time using the given layout.
// The layout uses Go's reference time format (Mon Jan 2 15:04:05 MST 2006).
func Time(layout string) func(string) (time.Time, error) {
	return func(value string) (time.Time, error) {
		return time.Parse(layout, value)
	}
}

// CastNumeric converts a numeric value from one type to another.
// Example: CastNumeric[int64, int32] converts int64 to int32.
// Note: This may truncate or lose precision depending on the types involved.
func CastNumeric[A Numeric, B Numeric](value A) (B, error) { //nolint:ireturn
	return B(value), nil
}

// ErrInvalidLogLevel is returned when a log level string is not recognized.
var ErrInvalidLogLevel = errors.New("invalid log level")

// SlogLevel parses a string as a slog.Level.
// Accepts: "debug", "info", "warn", "error" (case-sensitive).
// Returns ErrInvalidLogLevel for unrecognized values.
func SlogLevel(value string) (slog.Level, error) {
	switch value {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrInvalidLogLevel, value)
	}
}
