package xform

import (
	"compress/gzip"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amp-labs/amp-common/envtypes"
	"github.com/google/uuid"
)

const portMax = 65535

// TrimString represents a transformer that trims the given string.
func TrimString(s string) (string, error) {
	return strings.TrimSpace(s), nil
}

// SplitString represents a transformer that splits the given string using the given separator.
func SplitString(sep string) func(string) ([]string, error) {
	return func(s string) ([]string, error) {
		return strings.Split(s, sep), nil
	}
}

// Keyify represents a transformer that converts the given slice of strings to a map
// where the keys are the strings in the slice. The values are empty structs. This
// is basically just a very cheap way to create a set.
func Keyify(strs []string) (map[string]struct{}, error) {
	m := make(map[string]struct{}, len(strs))

	for _, s := range strs {
		m[s] = struct{}{}
	}

	return m, nil
}

// String represents a transformer that converts the given byte slice to a string.
func String(value []byte) (string, error) {
	return string(value), nil
}

// Bytes represents a transformer that converts the given string to a byte slice.
func Bytes(value string) ([]byte, error) {
	return []byte(value), nil
}

// ToLower represents a transformer that converts the given string to lower case.
func ToLower(s string) (string, error) {
	return strings.ToLower(s), nil
}

// ToUpper represents a transformer that converts the given string to upper case.
func ToUpper(s string) (string, error) {
	return strings.ToUpper(s), nil
}

// ReplaceAll represents a transformer that replaces all occurrences of the old
// string with the new string.
func ReplaceAll(oldStr, newStr string) func(string) (string, error) {
	return func(s string) (string, error) {
		return strings.ReplaceAll(s, oldStr, newStr), nil
	}
}

// OneOf represents a transformer that checks if the given value is one of the given choices.
func OneOf[A comparable](choices ...A) func(A) (A, error) { //nolint:ireturn
	return func(value A) (A, error) {
		for _, c := range choices {
			if c == value {
				return value, nil
			}
		}

		return value, ErrInvalidChoice
	}
}

// Bool represents a transformer that parses the given string as a bool.
func Bool(value string) (bool, error) {
	return strconv.ParseBool(value)
}

// Int64 represents a transformer that parses the given string as an int64.
func Int64(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

// Uint64 represents a transformer that parses the given string as an uint64.
func Uint64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

// Float64 represents a transformer that parses the given string as a float64.
func Float64(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// Float32 represents a transformer that casts the given float64 to a float32.
func Float32(value float64) (float32, error) {
	return float32(value), nil
}

// Positive represents a transformer that checks if the given value is positive.
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

// HostAndPort represents a transformer that parses the given string as a host:port pair.
func HostAndPort(value string) (envtypes.HostPort, error) {
	parts := strings.SplitN(value, ":", 2) //nolint:gomnd,mnd
	if len(parts) != 2 {                   //nolint:gomnd,mnd
		return envtypes.HostPort{}, fmt.Errorf("%w: %s", ErrBadHostAndPort, value)
	}

	host := parts[0]

	port, err := Port(parts[1])
	if err != nil {
		return envtypes.HostPort{}, fmt.Errorf("%w: %w", ErrBadHostAndPort, err)
	}

	return envtypes.HostPort{Host: host, Port: port}, nil
}

// Port represents a transformer that parses the given string as a port number.
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

// URL represents a transformer that parses the given string as a URL.
func URL(value string) (*url.URL, error) {
	return url.Parse(value)
}

// UUID represents a transformer that parses the given string as a UUID.
func UUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

// Path represents a transformer which treats the input as a local path.
// The path is stat'ed, and the result is a LocalPath struct.
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

// PathExists represents a transformer that checks if the given path exists.
func PathExists(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	return value, nil
}

// PathNotExists represents a transformer that checks if the given path does not exist.
func PathNotExists(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info != nil {
		return value, os.ErrExist
	}

	return value, nil
}

// PathIsFile represents a transformer that checks if the given path is a file.
func PathIsFile(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	if !value.Info.IsDir() {
		return value, nil
	}

	return value, fmt.Errorf("%w: %s", ErrNotAFile, value.Path)
}

// PathIsNonEmptyFile represents a transformer that checks if the given path is a
// non-empty file (i.e. size > 0).
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

// PathIsDir represents a transformer that checks if the given path is a directory.
func PathIsDir(value envtypes.LocalPath) (envtypes.LocalPath, error) {
	if value.Info == nil {
		return value, os.ErrNotExist
	}

	if value.Info.IsDir() {
		return value, nil
	}

	return value, fmt.Errorf("%w: %s", ErrNotADir, value.Path)
}

// OpenFile represents a transformer that opens the given file for reading.
// The file won't be closed, so the caller is responsible for closing it.
func OpenFile(value envtypes.LocalPath) (*os.File, error) {
	return os.Open(value.Path)
}

// ReadFile represents a transformer that reads the contents of the given file.
func ReadFile(value envtypes.LocalPath) ([]byte, error) {
	return os.ReadFile(value.Path)
}

// Duration represents a transformer that parses the given string as a time.Duration.
func Duration(value string) (time.Duration, error) {
	return time.ParseDuration(value)
}

// Time represents a transformer that parses the given string as a time.Time.
func Time(layout string) func(string) (time.Time, error) {
	return func(value string) (time.Time, error) {
		return time.Parse(layout, value)
	}
}

// CastNumeric represents a transformer that casts the given value to a
// different numeric type. Useful to go from (as an example) int64 to int32.
func CastNumeric[A Numeric, B Numeric](value A) (B, error) { //nolint:ireturn
	return B(value), nil
}

var ErrInvalidLogLevel = errors.New("invalid log level")

// SlogLevel represents a transformer that parses the given string as a slog.Level.
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
