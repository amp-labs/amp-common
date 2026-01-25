package statemachine

import (
	"context"
	"fmt"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// stateMachineContextKey is the key used to store state machine context in Go context.
const stateMachineContextKey contextKey = "statemachine_context"

// Metric outcome constants.
const (
	outcomeSuccess = "success"
	outcomeError   = "error"
)

// ActionExecutionHook is called before and after action execution.
type ActionExecutionHook func(ctx context.Context, actionName string, stateName string, phase string, err error)

// Engine orchestrates state machine execution.
type Engine struct {
	states             map[string]State
	transitions        []Transition
	initialState       string
	finalStates        []string
	actionTimeout      time.Duration
	executionHooks     []ActionExecutionHook
	enableCancellation bool
	logger             Logger
}

// NewEngine creates a new state machine engine from a configuration.
// If factory is nil, a new default factory is created.
func NewEngine(config *Config, factory *ActionFactory) (*Engine, error) {
	err := config.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	engine := &Engine{
		states:             make(map[string]State),
		transitions:        []Transition{},
		initialState:       config.InitialState,
		finalStates:        config.FinalStates,
		actionTimeout:      0, // No timeout by default
		executionHooks:     []ActionExecutionHook{},
		enableCancellation: true, // Enable cancellation by default
	}

	// Create action factory if not provided
	if factory == nil {
		factory = NewActionFactory()
	}

	// Build states from config
	for _, stateConfig := range config.States {
		state, err := buildStateFromConfig(stateConfig, factory)
		if err != nil {
			return nil, fmt.Errorf("failed to build state %s: %w", stateConfig.Name, err)
		}

		engine.RegisterState(state)
	}

	// Build transitions from config
	for _, transConfig := range config.Transitions {
		transition := buildTransitionFromConfig(transConfig)
		engine.RegisterTransition(transition)
	}

	return engine, nil
}

// Execute runs the state machine to completion.
func (e *Engine) Execute(ctx context.Context, smCtx *Context) (err error) {
	// Create root execution span
	ctx, span := startExecutionSpan(ctx, smCtx)

	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "completed")
		}

		span.End()
	}()

	// Record execution start time for duration metric
	executionStart := time.Now()

	defer func() {
		// Record execution duration on exit (success or error)
		outcome := outcomeSuccess
		if err != nil {
			outcome = outcomeError
		}

		executionDuration.WithLabelValues(
			sanitizeTool(smCtx.ToolName),
			outcome,
			sanitizeProjectID(smCtx.ProjectID),
			sanitizeChunkID(smCtx.ContextChunkID),
		).Observe(time.Since(executionStart).Seconds())

		// Record path length
		pathLength.WithLabelValues(
			sanitizeTool(smCtx.ToolName),
			outcome,
			sanitizeProjectID(smCtx.ProjectID),
			sanitizeChunkID(smCtx.ContextChunkID),
		).Observe(float64(len(smCtx.PathHistory)))

		// Log state machine execution summary with complete path
		if e.logger != nil {
			e.logExecutionSummary(ctx, time.Since(executionStart), err)
		}
	}()

	// Inject state machine context into Go context for logger access
	ctx = context.WithValue(ctx, stateMachineContextKey, smCtx)

	// Set initial state if not already set
	if smCtx.CurrentState == "" {
		smCtx.CurrentState = e.initialState
	}

	for {
		// Check for context cancellation
		if e.enableCancellation {
			select {
			case <-ctx.Done():
				// Record cancellation metric
				executionsCancelledTotal.WithLabelValues(
					sanitizeTool(smCtx.ToolName),
					smCtx.CurrentState,
					sanitizeProjectID(smCtx.ProjectID),
					sanitizeChunkID(smCtx.ContextChunkID),
				).Inc()

				return WrapStateError(smCtx.CurrentState, ctx.Err())
			default:
				// Continue execution
			}
		}

		// Get current state
		state, exists := e.states[smCtx.CurrentState]
		if !exists {
			return WrapStateError(smCtx.CurrentState, ErrStateNotFound)
		}

		// Append current state to path history before execution
		smCtx.AppendToPath(smCtx.CurrentState)

		// Create state span
		stateCtx, stateSpan := startStateSpan(ctx, smCtx.CurrentState, smCtx)

		// Log state entry
		if e.logger != nil {
			e.logger.StateEntered(stateCtx, smCtx.CurrentState, smCtx.Data)
		}

		// Execute state with timeout and hooks
		stateStartTime := time.Now()
		result, err := e.executeStateWithHooks(stateCtx, state, smCtx)
		stateElapsed := time.Since(stateStartTime)

		// Update span on state exit
		stateSpan.SetAttributes(attribute.Int64("duration_ms", stateElapsed.Milliseconds()))

		if err != nil {
			stateSpan.RecordError(err)
			stateSpan.SetStatus(codes.Error, err.Error())
			stateSpan.SetAttributes(attribute.String("error", err.Error()))
		} else {
			stateSpan.SetStatus(codes.Ok, "completed")
		}

		// End state span explicitly before proceeding
		stateSpan.End()

		// Log state exit (regardless of error)
		if e.logger != nil {
			e.logger.StateExited(stateCtx, smCtx.CurrentState, stateElapsed, err)
		}

		// Record state exit outcome
		outcome := outcomeSuccess
		if err != nil {
			outcome = outcomeError
		}

		stateVisitsTotal.WithLabelValues(
			sanitizeTool(smCtx.ToolName),
			smCtx.CurrentState,
			sanitizeProvider(smCtx.Provider),
			outcome,
			sanitizeProjectID(smCtx.ProjectID),
			sanitizeChunkID(smCtx.ContextChunkID),
		).Inc()

		// Record state duration
		stateDuration.WithLabelValues(
			sanitizeTool(smCtx.ToolName),
			smCtx.CurrentState,
			sanitizeProvider(smCtx.Provider),
			outcome,
			sanitizeProjectID(smCtx.ProjectID),
			sanitizeChunkID(smCtx.ContextChunkID),
		).Observe(stateElapsed.Seconds())

		if err != nil {
			return WrapStateError(smCtx.CurrentState, err)
		}

		// Check if complete
		if result.Complete || slices.Contains(e.finalStates, result.NextState) {
			smCtx.CurrentState = result.NextState

			return nil
		}

		// Find and apply transition
		nextState, err := e.findTransition(ctx, smCtx, result.NextState)
		if err != nil {
			return err
		}

		// Log transition execution
		if e.logger != nil {
			e.logger.TransitionExecuted(ctx, smCtx.CurrentState, nextState)
		}

		// Record transition metric
		transitionTotal.WithLabelValues(
			sanitizeTool(smCtx.ToolName),
			smCtx.CurrentState,
			nextState,
			sanitizeProvider(smCtx.Provider),
			sanitizeProjectID(smCtx.ProjectID),
			sanitizeChunkID(smCtx.ContextChunkID),
		).Inc()

		// Record transition and update state
		smCtx.AddTransition(smCtx.CurrentState, nextState, result.Data)
		smCtx.CurrentState = nextState
		smCtx.Merge(result.Data)
	}
}

// RegisterState registers a state with the engine.
func (e *Engine) RegisterState(state State) {
	e.states[state.Name()] = state
}

// RegisterTransition registers a transition with the engine.
func (e *Engine) RegisterTransition(transition Transition) {
	e.transitions = append(e.transitions, transition)
}

// SetActionTimeout sets the maximum duration for action execution.
// A timeout of 0 means no timeout.
func (e *Engine) SetActionTimeout(timeout time.Duration) {
	e.actionTimeout = timeout
}

// AddExecutionHook adds a hook to be called before and after each action execution.
// Hooks are called with phase "start" before execution and "end" after execution.
func (e *Engine) AddExecutionHook(hook ActionExecutionHook) {
	e.executionHooks = append(e.executionHooks, hook)
}

// SetCancellationEnabled enables or disables context cancellation handling.
func (e *Engine) SetCancellationEnabled(enabled bool) {
	e.enableCancellation = enabled
}

// SetLogger sets the logger for state machine execution.
func (e *Engine) SetLogger(logger Logger) {
	e.logger = logger
}

// logExecutionSummary logs a summary of the state machine execution including the complete path.
func (e *Engine) logExecutionSummary(ctx context.Context, duration time.Duration, err error) {
	if err != nil {
		e.logger.StateExited(ctx, "state_machine_execution", duration, err)
	} else {
		e.logger.StateExited(ctx, "state_machine_execution", duration, nil)
	}
}

// executeStateWithHooks executes a state with timeout and hooks.
func (e *Engine) executeStateWithHooks(ctx context.Context, state State, smCtx *Context) (TransitionResult, error) {
	// Create context with timeout if configured
	execCtx := ctx

	if e.actionTimeout > 0 {
		var cancel context.CancelFunc

		execCtx, cancel = context.WithTimeout(ctx, e.actionTimeout)
		defer cancel()
	}

	// Call "start" hooks
	for _, hook := range e.executionHooks {
		hook(execCtx, state.Name(), smCtx.CurrentState, "start", nil)
	}

	// Execute state
	result, err := state.Execute(execCtx, smCtx)

	// Call "end" hooks
	for _, hook := range e.executionHooks {
		hook(execCtx, state.Name(), smCtx.CurrentState, "end", err)
	}

	// Return result and error (properly propagated)
	return result, err
}

// findTransition finds the next valid transition.
func (e *Engine) findTransition(ctx context.Context, smCtx *Context, preferred string) (string, error) {
	currentState := smCtx.CurrentState

	// If a preferred next state is provided and there's a direct transition, use it
	//nolint:nestif // Nested logic necessary for transition preference check
	if preferred != "" {
		for _, transition := range e.transitions {
			if transition.From() == currentState && transition.To() == preferred {
				// Check condition
				valid, err := transition.Condition(ctx, smCtx)
				if err != nil {
					return "", WrapTransitionError(currentState, preferred, err)
				}

				if valid {
					return preferred, nil
				}
			}
		}
	}

	// Otherwise, find first valid transition from current state
	for _, transition := range e.transitions {
		if transition.From() == currentState {
			valid, err := transition.Condition(ctx, smCtx)
			if err != nil {
				return "", WrapTransitionError(currentState, transition.To(), err)
			}

			if valid {
				return transition.To(), nil
			}
		}
	}

	return "", WrapTransitionError(currentState, "", ErrTransitionNotFound)
}

// buildStateFromConfig creates a State from configuration.
func buildStateFromConfig(config StateConfig, factory *ActionFactory) (State, error) {
	switch config.Type {
	case "action":
		// Build actions
		var actions []Action

		for _, actionConfig := range config.Actions {
			action, err := factory.Create(actionConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create action: %w", err)
			}

			actions = append(actions, action)
		}

		// Wrap in sequence if multiple actions
		var action Action
		if len(actions) == 1 {
			action = actions[0]
		} else {
			action = NewSequenceAction(config.Name+"_sequence", actions...)
		}

		// Determine next state (will be overridden by transitions)
		return NewActionState(config.Name, action, ""), nil

	case "conditional":
		// Conditional states need custom implementation
		return nil, ErrConditionalNotImplemented

	case "final":
		return NewFinalState(config.Name), nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownStateType, config.Type)
	}
}

// buildTransitionFromConfig creates a Transition from configuration.
func buildTransitionFromConfig(config TransitionConfig) Transition {
	if config.Condition == "" || config.Condition == "always" {
		return NewSimpleTransition(config.From, config.To)
	}

	// For now, use expression-based transitions
	return NewExpressionTransition(config.From, config.To, config.Condition)
}
