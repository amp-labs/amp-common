package statemachine

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Logger provides logging hooks for state machine execution.
type Logger interface {
	StateEntered(ctx context.Context, state string, data map[string]any)
	StateExited(ctx context.Context, state string, duration time.Duration, err error)
	TransitionExecuted(ctx context.Context, from, to string)
	ActionStarted(ctx context.Context, action string)
	ActionCompleted(ctx context.Context, action string, duration time.Duration, err error)
}

// ObservabilityLabels contains contextual labels for observability.
type ObservabilityLabels struct {
	SessionID      string
	ProjectID      string
	Provider       string
	ContextChunkID string
	ToolName       string
	CurrentState   string
	PathHistory    []string
}

// GetObservabilityLabels extracts observability labels from the context.
// Returns an empty ObservabilityLabels struct if no Context is found.
func GetObservabilityLabels(ctx context.Context) ObservabilityLabels {
	smCtx, hasCtx := ctx.Value(stateMachineContextKey).(*Context)
	if !hasCtx || smCtx == nil {
		return ObservabilityLabels{}
	}

	return ObservabilityLabels{
		SessionID:      smCtx.SessionID,
		ProjectID:      smCtx.ProjectID,
		Provider:       smCtx.Provider,
		ContextChunkID: smCtx.ContextChunkID,
		ToolName:       smCtx.ToolName,
		CurrentState:   smCtx.CurrentState,
		PathHistory:    smCtx.PathHistory,
	}
}

// DefaultLogger implements Logger using slog.
type DefaultLogger struct {
	logger *slog.Logger
}

// NewDefaultLogger creates a new default logger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: slog.Default(),
	}
}

func (l *DefaultLogger) StateEntered(ctx context.Context, state string, data map[string]any) {
	// Extract context if available for additional fields
	smCtx, hasCtx := ctx.Value(stateMachineContextKey).(*Context)

	fields := []any{
		"state", state,
		"data_keys", len(data),
	}

	if hasCtx {
		fields = append(fields,
			"session_id", smCtx.SessionID,
			"project_id", smCtx.ProjectID,
			"provider", smCtx.Provider,
			"chunk_id", smCtx.ContextChunkID,
			"tool", smCtx.ToolName,
			"path_history", smCtx.PathHistory,
		)

		// Add data keys as a list for debugging
		dataKeys := make([]string, 0, len(data))
		for k := range data {
			dataKeys = append(dataKeys, k)
		}

		fields = append(fields, "data_key_names", dataKeys)
	}

	l.logger.InfoContext(ctx, "State entered", fields...)
}

func (l *DefaultLogger) StateExited(ctx context.Context, state string, duration time.Duration, err error) {
	smCtx, hasCtx := ctx.Value(stateMachineContextKey).(*Context)

	fields := []any{
		"state", state,
		"duration_ms", duration.Milliseconds(),
	}

	if hasCtx {
		fields = append(fields,
			"session_id", smCtx.SessionID,
			"project_id", smCtx.ProjectID,
			"provider", smCtx.Provider,
			"chunk_id", smCtx.ContextChunkID,
			"tool", smCtx.ToolName,
			"path_history", smCtx.PathHistory,
			"outcome", func() string {
				if err != nil {
					return "error"
				}

				return "success"
			}(),
		)
	}

	if err != nil {
		l.logger.ErrorContext(ctx, "State exited with error", append(fields, "error", err)...)
	} else {
		l.logger.InfoContext(ctx, "State exited", fields...)
	}
}

func (l *DefaultLogger) TransitionExecuted(ctx context.Context, from, to string) {
	smCtx, hasCtx := ctx.Value(stateMachineContextKey).(*Context)

	fields := []any{
		"from", from,
		"to", to,
	}

	if hasCtx {
		fields = append(fields,
			"session_id", smCtx.SessionID,
			"project_id", smCtx.ProjectID,
			"provider", smCtx.Provider,
			"chunk_id", smCtx.ContextChunkID,
			"tool", smCtx.ToolName,
			"path_history", smCtx.PathHistory,
		)
	}

	l.logger.InfoContext(ctx, "Transition executed", fields...)
}

func (l *DefaultLogger) ActionStarted(ctx context.Context, action string) {
	l.logger.InfoContext(ctx, "Action started",
		"action", action,
	)
}

func (l *DefaultLogger) ActionCompleted(ctx context.Context, action string, duration time.Duration, err error) {
	if err != nil {
		l.logger.ErrorContext(ctx, "Action completed with error",
			"action", action,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
	} else {
		l.logger.InfoContext(ctx, "Action completed",
			"action", action,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// LoggingEngine wraps an engine with logging.
type LoggingEngine struct {
	engine *Engine
	logger Logger
}

// WithLogging wraps an engine with logging capabilities.
func WithLogging(engine *Engine, logger Logger) *LoggingEngine {
	if logger == nil {
		logger = NewDefaultLogger()
	}

	return &LoggingEngine{
		engine: engine,
		logger: logger,
	}
}

// Execute runs the state machine with logging.
func (e *LoggingEngine) Execute(ctx context.Context, smCtx *Context) error {
	// Set logger on the engine so it logs state transitions
	e.engine.SetLogger(e.logger)

	// Execute the state machine (engine will handle logging)
	return e.engine.Execute(ctx, smCtx)
}

// LoggingAction wraps an action with logging.
type LoggingAction struct {
	action Action
	logger Logger
}

// NewLoggingAction wraps an action with logging.
func NewLoggingAction(action Action, logger Logger) *LoggingAction {
	if logger == nil {
		logger = NewDefaultLogger()
	}

	return &LoggingAction{
		action: action,
		logger: logger,
	}
}

func (a *LoggingAction) Name() string {
	return a.action.Name()
}

func (a *LoggingAction) Execute(ctx context.Context, smCtx *Context) error {
	a.logger.ActionStarted(ctx, a.action.Name())

	// Create action span
	actionCtx, span := startActionSpan(ctx, a.action.Name(), smCtx.CurrentState, smCtx)
	defer span.End()

	start := time.Now()
	err := a.action.Execute(actionCtx, smCtx)
	duration := time.Since(start)

	// Update span with duration and status
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error", err.Error()))
	} else {
		span.SetStatus(codes.Ok, "completed")
	}

	a.logger.ActionCompleted(ctx, a.action.Name(), duration, err)

	// Record action duration metric
	actionDuration.WithLabelValues(
		sanitizeTool(smCtx.ToolName),
		a.action.Name(),
		smCtx.CurrentState,
		sanitizeProjectID(smCtx.ProjectID),
		sanitizeChunkID(smCtx.ContextChunkID),
	).Observe(duration.Seconds())

	return err
}
