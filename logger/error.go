package logger

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// AnnotateError wraps an error with structured logging attributes (slog key-value pairs).
// When the returned error is logged using a logger configured with slogErrorLogger,
// the attributes are automatically extracted and included in the log output.
//
// This allows attaching rich context to errors at the point of creation, which will
// be preserved through error wrapping and unwrapping, and displayed when the error
// is ultimately logged.
//
// Args should be key-value pairs compatible with slog (string keys followed by values).
//
// Example:
//
//	err := doSomething()
//	if err != nil {
//	    return AnnotateError(err, "user_id", userID, "operation", "update_profile")
//	}
//
// Returns nil if err is nil.
func AnnotateError(err error, args ...any) error {
	if err == nil {
		return nil
	}

	r := slog.NewRecord(time.Now(), slog.LevelDebug, "", 0)
	r.Add(args...)

	var errAttrs []slog.Attr

	r.Attrs(func(attr slog.Attr) bool {
		errAttrs = append(errAttrs, attr)

		return true
	})

	return &slogError{
		err:   err,
		attrs: errAttrs,
	}
}

// slogError wraps an error with structured logging attributes.
// It implements the error interface and supports error unwrapping,
// making it compatible with errors.Is and errors.As.
type slogError struct {
	err   error       // The underlying error
	attrs []slog.Attr // Structured logging attributes attached to this error
}

// Error returns the error message from the underlying error.
func (s *slogError) Error() string {
	return s.err.Error()
}

// Unwrap returns the underlying error, supporting error chain traversal.
func (s *slogError) Unwrap() error {
	return s.err
}

// Compile-time check that slogError implements error interface.
var _ error = (*slogError)(nil)

// slogErrorLogger is a slog.Handler decorator that extracts structured attributes
// from annotated errors (created via AnnotateError) and includes them in log output.
//
// When a log record contains an error attribute that was created with AnnotateError,
// this handler extracts the embedded attributes and adds them to the log record,
// providing richer context in the logs.
//
// This handler wraps another slog.Handler and delegates all actual logging to it.
type slogErrorLogger struct {
	inner slog.Handler // The wrapped handler that performs actual logging
}

// Compile-time check that slogErrorLogger implements slog.Handler interface.
var _ slog.Handler = (*slogErrorLogger)(nil)

// Enabled reports whether the handler handles records at the given level.
// Delegates to the inner handler.
func (s *slogErrorLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return s.inner.Enabled(ctx, level)
}

// Handle processes a log record, extracting attributes from any annotated errors
// and including them in the final log output.
//
// The handler iterates through all attributes in the record. When it finds an error
// attribute that is an annotated error (slogError), it:
// 1. Replaces the error attribute with the unwrapped error
// 2. Extracts the embedded attributes and adds them to the record
//
// This ensures that context added via AnnotateError is visible in logs.
func (s *slogErrorLogger) Handle(ctx context.Context, record slog.Record) error {
	var (
		baseAttrs []slog.Attr
		errAttrs  []slog.Attr
	)

	record.Attrs(func(attr slog.Attr) bool {
		val := attr.Value.Any()

		switch v := val.(type) {
		case error:
			var se *slogError

			if errors.As(v, &se) {
				errAttr := slog.Attr{
					Key:   attr.Key,
					Value: slog.AnyValue(se.err),
				}

				baseAttrs = append(baseAttrs, errAttr)

				errAttrs = append(errAttrs, se.attrs...)
			}
		default:
			baseAttrs = append(baseAttrs, attr)
		}

		return true
	})

	if len(errAttrs) > 0 {
		r := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
		r.AddAttrs(baseAttrs...)
		r.AddAttrs(errAttrs...)

		return s.inner.Handle(ctx, r)
	}

	return s.inner.Handle(ctx, record)
}

// WithAttrs returns a new handler with the given attributes added.
// The new handler wraps the result of calling WithAttrs on the inner handler,
// maintaining the error annotation extraction behavior.
func (s *slogErrorLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	handler := s.inner.WithAttrs(attrs)

	return &slogErrorLogger{
		inner: handler,
	}
}

// WithGroup returns a new handler with the given group name.
// The new handler wraps the result of calling WithGroup on the inner handler,
// maintaining the error annotation extraction behavior.
func (s *slogErrorLogger) WithGroup(name string) slog.Handler {
	handler := s.inner.WithGroup(name)

	return &slogErrorLogger{
		inner: handler,
	}
}
