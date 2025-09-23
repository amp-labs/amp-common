package xform

import (
	"errors"
	"time"
)

var (
	ErrInvalidChoice  = errors.New("invalid choice")
	ErrNonPositive    = errors.New("value must be positive")
	ErrBadPort        = errors.New("invalid port number")
	ErrBadHostAndPort = errors.New("invalid host:port pair")
	ErrNotAFile       = errors.New("not a file")
	ErrNotADir        = errors.New("not a directory")
	ErrEmptyFile      = errors.New("empty file")
)

type Intish interface {
	int | int8 | int16 | int32 | int64 | time.Duration
}

type Uintish interface {
	uint | uint8 | uint16 | uint32 | uint64
}

type Numeric interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 | float32 | float64 | int | uint | time.Duration
}
