// Package xform provides type-safe transformation functions for converting
// and validating data. These transformers are designed to be composable and
// work seamlessly with the envutil package's fluent API.
//
// Each transformer function follows the pattern: func(Input) (Output, error)
// This allows them to be chained together for complex transformations:
//
//	envutil.String("PORT",
//	    envutil.Transform(xform.Int64),
//	    envutil.Transform(xform.CastNumeric[int64, int]),
//	    envutil.Transform(xform.Positive[int]),
//	).Value()
package xform

import (
	"errors"
	"time"
)

var (
	// ErrInvalidChoice is returned when a value is not one of the allowed choices.
	ErrInvalidChoice = errors.New("invalid choice")

	// ErrNonPositive is returned when a numeric value is not positive (i.e., <= 0).
	ErrNonPositive = errors.New("value must be positive")

	// ErrBadPort is returned when a port number is invalid (< 0 or > 65535).
	ErrBadPort = errors.New("invalid port number")

	// ErrBadHostAndPort is returned when a host:port pair cannot be parsed.
	ErrBadHostAndPort = errors.New("invalid host:port pair")

	// ErrNotAFile is returned when a path exists but is not a file.
	ErrNotAFile = errors.New("not a file")

	// ErrNotADir is returned when a path exists but is not a directory.
	ErrNotADir = errors.New("not a directory")

	// ErrEmptyFile is returned when a file exists but has zero size.
	ErrEmptyFile = errors.New("empty file")
)

// Intish is a constraint for signed integer types and time.Duration.
type Intish interface {
	int | int8 | int16 | int32 | int64 | time.Duration
}

// Uintish is a constraint for unsigned integer types.
type Uintish interface {
	uint | uint8 | uint16 | uint32 | uint64
}

// Numeric is a constraint for all numeric types including integers, floats, and time.Duration.
type Numeric interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | float32 | float64 | int | uint | time.Duration
}
