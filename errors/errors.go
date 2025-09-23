package errors

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrWrongType      = errors.New("wrong type")
)
