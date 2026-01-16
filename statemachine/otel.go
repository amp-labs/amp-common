package statemachine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// startExecutionSpan creates root span for state machine execution.
// Uses the global tracer initialized by github.com/amp-labs/amp-common/telemetry.
// The caller is responsible for calling span.End().
//
//nolint:spancheck // Span lifecycle managed by caller (factory pattern)
func startExecutionSpan(ctx context.Context, smCtx *Context) (context.Context, trace.Span) {
	tracer := otel.Tracer("statemachine")
	ctx, span := tracer.Start(ctx, "statemachine.execute")
	addContextAttributes(span, smCtx)
	logSpanDebug(ctx, "started", "statemachine.execute", span)

	return ctx, span
}

// startStateSpan creates child span for state execution.
// The caller is responsible for calling span.End().
//
//nolint:spancheck // Span lifecycle managed by caller (factory pattern)
func startStateSpan(ctx context.Context, stateName string, smCtx *Context) (context.Context, trace.Span) {
	tracer := otel.Tracer("statemachine")
	spanName := "state." + stateName
	ctx, span := tracer.Start(ctx, spanName)
	addContextAttributes(span, smCtx)
	span.SetAttributes(
		attribute.String("state", stateName),
		attribute.StringSlice("path_history", smCtx.PathHistory),
	)
	logSpanDebug(ctx, "started", spanName, span)

	return ctx, span
}

// startActionSpan creates child span for action execution.
// The caller is responsible for calling span.End().
//
//nolint:spancheck // Span lifecycle managed by caller (factory pattern)
func startActionSpan(
	ctx context.Context,
	actionName string,
	stateName string,
	smCtx *Context,
) (context.Context, trace.Span) {
	tracer := otel.Tracer("statemachine")
	spanName := "action." + actionName
	ctx, span := tracer.Start(ctx, spanName)
	addContextAttributes(span, smCtx)
	span.SetAttributes(
		attribute.String("action", actionName),
		attribute.String("state", stateName),
		attribute.StringSlice("path_history", smCtx.PathHistory),
	)
	logSpanDebug(ctx, "started", spanName, span)

	return ctx, span
}

// addContextAttributes adds Context metadata to span.
func addContextAttributes(span trace.Span, smCtx *Context) {
	span.SetAttributes(
		attribute.String("tool", smCtx.ToolName),
		attribute.String("session_id", smCtx.SessionID),
		attribute.String("project_id_hash", hashID(smCtx.ProjectID)),
		attribute.String("provider", smCtx.Provider),
		attribute.String("chunk_id", smCtx.ContextChunkID),
		attribute.String("chunk_id_hash", hashID(smCtx.ContextChunkID)),
	)
}

// hashID creates a short hash of an ID for span attributes (privacy).
func hashID(id string) string {
	if id == "" {
		return ""
	}

	h := sha256.Sum256([]byte(id))

	return hex.EncodeToString(h[:4]) // First 8 chars
}

// logSpanDebug logs span creation/completion when MCP_DEBUG=1.
func logSpanDebug(ctx context.Context, phase string, spanName string, span trace.Span) {
	if os.Getenv("MCP_DEBUG") != "1" {
		return
	}

	spanCtx := span.SpanContext()
	slog.InfoContext(ctx, "OTEL Span "+phase,
		"span_name", spanName,
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)
}

// extractTraceContext extracts trace ID and span ID from context for logging.
func extractTraceContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()

		return spanCtx.TraceID().String(), spanCtx.SpanID().String()
	}

	return "", ""
}

// isDebugMode checks if MCP_DEBUG mode is enabled.
func isDebugMode() bool {
	return strings.EqualFold(os.Getenv("MCP_DEBUG"), "1") ||
		strings.EqualFold(os.Getenv("MCP_DEBUG"), "true")
}
