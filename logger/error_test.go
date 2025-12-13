//nolint:err113 // Test file uses errors.New() for creating test errors
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnnotateError_NilError tests that AnnotateError returns nil when given a nil error.
func TestAnnotateError_NilError(t *testing.T) {
	t.Parallel()

	result := AnnotateError(nil, "key", "value")
	assert.NoError(t, result)
}

// TestAnnotateError_BasicAnnotation tests basic error annotation with attributes.
func TestAnnotateError_BasicAnnotation(t *testing.T) { //nolint:err113 // test errors
	t.Parallel()

	baseErr := errors.New("base error")
	annotated := AnnotateError(baseErr, "user_id", "123", "operation", "update")

	require.Error(t, annotated)
	assert.Equal(t, "base error", annotated.Error())

	// Verify it's a slogError
	var se *slogError
	require.ErrorAs(t, annotated, &se)
	assert.Equal(t, baseErr, se.err)
	assert.Len(t, se.attrs, 2)

	// Verify attribute keys
	keys := make(map[string]bool)
	for _, attr := range se.attrs {
		keys[attr.Key] = true
	}

	assert.True(t, keys["user_id"])
	assert.True(t, keys["operation"])
}

// TestAnnotateError_SingleAttribute tests annotation with a single attribute.
func TestAnnotateError_SingleAttribute(t *testing.T) { //nolint:err113 // test errors
	t.Parallel()

	baseErr := errors.New("test error")
	annotated := AnnotateError(baseErr, "key", "value")

	var se *slogError
	require.ErrorAs(t, annotated, &se)
	require.Len(t, se.attrs, 1)
	assert.Equal(t, "key", se.attrs[0].Key)
	assert.Equal(t, "value", se.attrs[0].Value.Any())
}

// TestAnnotateError_NoAttributes tests annotation with no attributes.
func TestAnnotateError_NoAttributes(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("test error")
	annotated := AnnotateError(baseErr)

	require.Error(t, annotated)

	var se *slogError
	require.ErrorAs(t, annotated, &se)
	assert.Empty(t, se.attrs)
}

// TestAnnotateError_VariousTypes tests annotation with various value types.
func TestAnnotateError_VariousTypes(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("test error")
	annotated := AnnotateError(
		baseErr,
		"string", "value",
		"int", 42,
		"bool", true,
		"float", 3.14,
	)

	var se *slogError
	require.ErrorAs(t, annotated, &se)
	assert.Len(t, se.attrs, 4)

	attrMap := make(map[string]any)
	for _, attr := range se.attrs {
		attrMap[attr.Key] = attr.Value.Any()
	}

	assert.Equal(t, "value", attrMap["string"])
	assert.Equal(t, int64(42), attrMap["int"]) // slog converts int to int64
	assert.Equal(t, true, attrMap["bool"])
	assert.InDelta(t, 3.14, attrMap["float"], 0.001)
}

// TestSlogError_ErrorMessage tests that Error() returns the underlying error message.
func TestSlogError_ErrorMessage(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("original error message")
	annotated := AnnotateError(baseErr, "key", "value")

	assert.Equal(t, "original error message", annotated.Error())
}

// TestSlogError_Unwrap tests that Unwrap() returns the underlying error.
func TestSlogError_Unwrap(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	annotated := AnnotateError(baseErr, "key", "value")

	unwrapped := errors.Unwrap(annotated)
	assert.Equal(t, baseErr, unwrapped)
}

// TestSlogError_ErrorsIs tests compatibility with errors.Is.
func TestSlogError_ErrorsIs(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	annotated := AnnotateError(baseErr, "key", "value")

	require.ErrorIs(t, annotated, baseErr)
	assert.NotErrorIs(t, annotated, errors.New("different error")) //nolint:err113 // test error
}

// TestSlogError_ErrorsAs tests compatibility with errors.As.
func TestSlogError_ErrorsAs(t *testing.T) {
	t.Parallel()

	baseErr := &customError{msg: "custom error"}
	annotated := AnnotateError(baseErr, "key", "value")

	var ce *customError
	require.ErrorAs(t, annotated, &ce)
	assert.Equal(t, "custom error", ce.msg)
}

// TestSlogError_ChainedAnnotation tests annotating an already annotated error.
func TestSlogError_ChainedAnnotation(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")
	annotated1 := AnnotateError(baseErr, "key1", "value1")
	annotated2 := AnnotateError(annotated1, "key2", "value2")

	// Both annotations should be present
	var se *slogError
	require.ErrorAs(t, annotated2, &se)

	// The outer annotation should have key2
	require.Len(t, se.attrs, 1)
	assert.Equal(t, "key2", se.attrs[0].Key)

	// The inner annotation should still be accessible via unwrap
	unwrapped := errors.Unwrap(annotated2)
	require.ErrorAs(t, unwrapped, &se)
	require.Len(t, se.attrs, 1)
	assert.Equal(t, "key1", se.attrs[0].Key)
}

// TestSlogErrorLogger_Enabled tests that Enabled delegates to the inner handler.
func TestSlogErrorLogger_Enabled(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	logger := &slogErrorLogger{inner: innerHandler}

	assert.True(t, logger.Enabled(context.Background(), slog.LevelError))
	assert.True(t, logger.Enabled(context.Background(), slog.LevelWarn))
	assert.False(t, logger.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, logger.Enabled(context.Background(), slog.LevelDebug))
}

// TestSlogErrorLogger_Handle_NoAnnotatedError tests normal error logging without annotation.
func TestSlogErrorLogger_Handle_NoAnnotatedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	record := slog.NewRecord(time.Now(), slog.LevelError, "test message", 0)
	record.AddAttrs(slog.Any("error", errors.New("plain error")))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "plain error")
}

// TestSlogErrorLogger_Handle_WithAnnotatedError tests extraction of annotated error attributes.
func TestSlogErrorLogger_Handle_WithAnnotatedError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	baseErr := errors.New("base error")
	annotated := AnnotateError(baseErr, "user_id", "123", "operation", "update")

	record := slog.NewRecord(time.Now(), slog.LevelError, "operation failed", 0)
	record.AddAttrs(slog.Any("error", annotated))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "operation failed")
	assert.Contains(t, output, "base error")
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
	assert.Contains(t, output, "operation")
	assert.Contains(t, output, "update")
}

// TestSlogErrorLogger_Handle_MultipleAttributes tests multiple annotated attributes.
func TestSlogErrorLogger_Handle_MultipleAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	baseErr := errors.New("test error")
	annotated := AnnotateError(
		baseErr,
		"attr1", "value1",
		"attr2", 42,
		"attr3", true,
	)

	record := slog.NewRecord(time.Now(), slog.LevelError, "test", 0)
	record.AddAttrs(slog.Any("error", annotated))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	var logData map[string]any

	err = json.Unmarshal(buf.Bytes(), &logData)
	require.NoError(t, err)

	assert.Equal(t, "value1", logData["attr1"])
	assert.InDelta(t, 42, logData["attr2"], 0.001) // JSON numbers are float64
	assert.Equal(t, true, logData["attr3"])
}

// TestSlogErrorLogger_Handle_MixedAttributes tests annotated error with other attributes.
func TestSlogErrorLogger_Handle_MixedAttributes(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	baseErr := errors.New("error message")
	annotated := AnnotateError(baseErr, "from_error", "error_value")

	record := slog.NewRecord(time.Now(), slog.LevelError, "mixed test", 0)
	record.AddAttrs(
		slog.String("regular_attr", "regular_value"),
		slog.Any("error", annotated),
		slog.Int("another_attr", 100),
	)

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "regular_attr")
	assert.Contains(t, output, "regular_value")
	assert.Contains(t, output, "from_error")
	assert.Contains(t, output, "error_value")
	assert.Contains(t, output, "another_attr")
	assert.Contains(t, output, "100")
}

// TestSlogErrorLogger_WithAttrs tests that WithAttrs maintains error extraction behavior.
func TestSlogErrorLogger_WithAttrs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	// Create a handler with additional attributes
	withAttrs := logger.WithAttrs([]slog.Attr{
		slog.String("handler_attr", "handler_value"),
	})

	// Verify it's still an slogErrorLogger
	errLogger, ok := withAttrs.(*slogErrorLogger)
	require.True(t, ok)

	baseErr := errors.New("test error")
	annotated := AnnotateError(baseErr, "error_attr", "error_value")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "with attrs test", 0)
	record.AddAttrs(slog.Any("error", annotated))

	err := errLogger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "handler_attr")
	assert.Contains(t, output, "handler_value")
	assert.Contains(t, output, "error_attr")
	assert.Contains(t, output, "error_value")
}

// TestSlogErrorLogger_WithGroup tests that WithGroup maintains error extraction behavior.
func TestSlogErrorLogger_WithGroup(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	// Create a handler with a group
	withGroup := logger.WithGroup("mygroup")

	// Verify it's still an slogErrorLogger
	errLogger, ok := withGroup.(*slogErrorLogger)
	require.True(t, ok)

	baseErr := errors.New("test error")
	annotated := AnnotateError(baseErr, "error_attr", "error_value")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "with group test", 0)
	record.AddAttrs(slog.Any("error", annotated))

	err := errLogger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "mygroup")
	assert.Contains(t, output, "error_attr")
	assert.Contains(t, output, "error_value")
}

// TestSlogErrorLogger_Integration tests the complete flow with ConfigureLoggingWithOptions.
func TestSlogErrorLogger_Integration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "error-test",
		JSON:      true,
		Output:    &buf,
	})

	baseErr := errors.New("database connection failed")
	annotated := AnnotateError(
		baseErr,
		"host", "db.example.com",
		"port", 5432,
		"retry_count", 3,
	)

	ctx := context.Background()
	ctx = WithCustomerId(ctx, "customer-123")

	Get(ctx).Error("failed to connect to database", "error", annotated)

	output := buf.String()
	assert.Contains(t, output, "error-test")                 // subsystem
	assert.Contains(t, output, "customer-123")               // from context
	assert.Contains(t, output, "database connection failed") // error message
	assert.Contains(t, output, "host")
	assert.Contains(t, output, "db.example.com")
	assert.Contains(t, output, "port")
	assert.Contains(t, output, "5432")
	assert.Contains(t, output, "retry_count")
	assert.Contains(t, output, "3")
}

// TestSlogErrorLogger_ChainedAnnotations tests logging errors with multiple annotation layers.
func TestSlogErrorLogger_ChainedAnnotations(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "chain-test",
		JSON:      true,
		Output:    &buf,
	})

	baseErr := errors.New("original error")
	annotated1 := AnnotateError(baseErr, "layer1", "value1")
	annotated2 := AnnotateError(annotated1, "layer2", "value2")

	ctx := context.Background()
	Get(ctx).Error("chained error", "error", annotated2)

	output := buf.String()
	// Only the outermost annotation's attributes should be extracted
	assert.Contains(t, output, "layer2")
	assert.Contains(t, output, "value2")
	// The inner annotation is still part of the error chain but not automatically extracted
	assert.Contains(t, output, "original error")
}

// TestSlogErrorLogger_Handle_JoinedErrors tests extraction of attributes from errors.Join.
func TestSlogErrorLogger_Handle_JoinedErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	// Create multiple annotated errors
	err1 := AnnotateError(errors.New("first error"), "error1_attr", "value1")
	err2 := AnnotateError(errors.New("second error"), "error2_attr", "value2")

	// Join them
	joinedErr := errors.Join(err1, err2)

	record := slog.NewRecord(time.Now(), slog.LevelError, "multiple errors occurred", 0)
	record.AddAttrs(slog.Any("error", joinedErr))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "multiple errors occurred")
	// Verify both error messages are present
	assert.Contains(t, output, "first error")
	assert.Contains(t, output, "second error")
	// Verify attributes from both errors are extracted
	assert.Contains(t, output, "error1_attr")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "error2_attr")
	assert.Contains(t, output, "value2")
}

// TestSlogErrorLogger_Handle_JoinedErrors_WithIndexing tests indexed error attributes.
func TestSlogErrorLogger_Handle_JoinedErrors_WithIndexing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	err1 := AnnotateError(errors.New("error one"), "source", "service1")
	err2 := AnnotateError(errors.New("error two"), "source", "service2")
	err3 := AnnotateError(errors.New("error three"), "source", "service3")

	joinedErr := errors.Join(err1, err2, err3)

	record := slog.NewRecord(time.Now(), slog.LevelError, "batch operation failed", 0)
	record.AddAttrs(slog.Any("error", joinedErr))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	var logData map[string]any

	err = json.Unmarshal(buf.Bytes(), &logData)
	require.NoError(t, err)

	// Verify indexed error attributes exist
	assert.Contains(t, logData, "error[0]")
	assert.Contains(t, logData, "error[1]")
	assert.Contains(t, logData, "error[2]")

	// Verify the error messages are present
	output := buf.String()
	assert.Contains(t, output, "error one")
	assert.Contains(t, output, "error two")
	assert.Contains(t, output, "error three")
}

// TestSlogErrorLogger_Handle_JoinedErrors_MixedAnnotated tests joined errors with mixed annotation.
func TestSlogErrorLogger_Handle_JoinedErrors_MixedAnnotated(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	// Mix of annotated and plain errors
	annotatedErr := AnnotateError(errors.New("annotated error"), "user_id", "123")
	plainErr := errors.New("plain error")

	joinedErr := errors.Join(annotatedErr, plainErr)

	record := slog.NewRecord(time.Now(), slog.LevelError, "mixed errors", 0)
	record.AddAttrs(slog.Any("error", joinedErr))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	// Both error messages should be present
	assert.Contains(t, output, "annotated error")
	assert.Contains(t, output, "plain error")
	// Attribute from annotated error should be extracted
	assert.Contains(t, output, "user_id")
	assert.Contains(t, output, "123")
}

// TestSlogErrorLogger_Handle_NestedJoinedErrors tests deeply nested joined errors.
func TestSlogErrorLogger_Handle_NestedJoinedErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	innerHandler := slog.NewJSONHandler(&buf, nil)
	logger := &slogErrorLogger{inner: innerHandler}

	// Create nested joined errors
	err1 := AnnotateError(errors.New("error 1"), "level", "1")
	err2 := AnnotateError(errors.New("error 2"), "level", "2")
	joined1 := errors.Join(err1, err2)

	err3 := AnnotateError(errors.New("error 3"), "level", "3")
	joined2 := errors.Join(joined1, err3)

	record := slog.NewRecord(time.Now(), slog.LevelError, "nested join", 0)
	record.AddAttrs(slog.Any("error", joined2))

	err := logger.Handle(context.Background(), record)
	require.NoError(t, err)

	output := buf.String()
	// All error messages should be present
	assert.Contains(t, output, "error 1")
	assert.Contains(t, output, "error 2")
	assert.Contains(t, output, "error 3")
}

// TestSlogErrorLogger_Integration_JoinedErrors tests the complete flow with joined errors.
func TestSlogErrorLogger_Integration_JoinedErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	ConfigureLoggingWithOptions(Options{
		Subsystem: "join-test",
		JSON:      true,
		Output:    &buf,
	})

	// Create multiple errors from different operations
	dbErr := AnnotateError(
		errors.New("connection failed"),
		"host", "db.example.com",
		"port", 5432,
	)
	cacheErr := AnnotateError(
		errors.New("cache miss"),
		"key", "user:123",
		"ttl", 300,
	)
	apiErr := AnnotateError(
		errors.New("timeout"),
		"endpoint", "/api/users",
		"timeout_ms", 5000,
	)

	combinedErr := errors.Join(dbErr, cacheErr, apiErr)

	ctx := context.Background()
	Get(ctx).Error("multiple subsystems failed", "error", combinedErr)

	output := buf.String()

	// Verify all error messages are present
	assert.Contains(t, output, "connection failed")
	assert.Contains(t, output, "cache miss")
	assert.Contains(t, output, "timeout")

	// Verify attributes from all errors are extracted
	assert.Contains(t, output, "host")
	assert.Contains(t, output, "db.example.com")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "user:123")
	assert.Contains(t, output, "endpoint")
	assert.Contains(t, output, "/api/users")
}

// customError is a helper type for testing errors.As compatibility.
type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
