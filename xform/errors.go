package xform

import "errors"

// ErrInvalidGzipLevel is returned when a gzip compression level string
// cannot be parsed or is not a valid compression level constant.
var ErrInvalidGzipLevel = errors.New("invalid gzip level")
