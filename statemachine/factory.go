package statemachine

import (
	"context"
	"fmt"
	"time"
)

// ActionFactory creates actions from configuration.
// Applications can register custom action builders to extend the framework.
type ActionFactory struct {
	builders map[string]ActionBuilder
}

// ActionBuilder is a function that creates an action from configuration.
// The factory parameter allows reusing custom builders for nested actions.
type ActionBuilder func(factory *ActionFactory, name string, params map[string]any) (Action, error)

// ValidatorFunc is a function that validates context data.
type ValidatorFunc func(ctx context.Context, smCtx *Context) (bool, string, error)

// NewActionFactory creates a new action factory with default builders.
func NewActionFactory() *ActionFactory {
	factory := &ActionFactory{
		builders: make(map[string]ActionBuilder),
	}

	// Register built-in action builders
	factory.Register("noop", noopActionBuilder)
	factory.Register("sequence", sequenceActionBuilder)
	factory.Register("validation", validationActionBuilder)
	factory.Register("conditional", conditionalActionBuilder)
	factory.Register("retry", retryActionBuilder)

	return factory
}

// Register registers a custom action builder.
func (f *ActionFactory) Register(actionType string, builder ActionBuilder) {
	f.builders[actionType] = builder
}

// Create creates an action from configuration.
func (f *ActionFactory) Create(config ActionConfig) (Action, error) {
	builder, ok := f.builders[config.Type]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownActionType, config.Type)
	}

	return builder(f, config.Name, config.Parameters)
}

// noopActionBuilder creates a NoopAction from parameters.
func noopActionBuilder(_ *ActionFactory, name string, params map[string]any) (Action, error) {
	return &NoopAction{BaseAction: BaseAction{name: name}}, nil
}

// sequenceActionBuilder creates a SequenceAction from parameters.
func sequenceActionBuilder(factory *ActionFactory, name string, params map[string]any) (Action, error) {
	// Extract nested actions
	actionsParam, ok := params["actions"].([]any)
	if !ok {
		return nil, ErrSequenceActionsRequired
	}

	actions := make([]Action, 0, len(actionsParam))
	for actionIdx, actionParam := range actionsParam {
		actionMap, ok := actionParam.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("action %d: %w", actionIdx, ErrInvalidActionFormat)
		}

		// Convert to ActionConfig
		actionType, _ := actionMap["type"].(string)
		actionName, _ := actionMap["name"].(string)
		actionParams, _ := actionMap["parameters"].(map[string]any)

		config := ActionConfig{
			Type:       actionType,
			Name:       actionName,
			Parameters: actionParams,
		}

		action, err := factory.Create(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create action %d: %w", actionIdx, err)
		}

		actions = append(actions, action)
	}

	return NewSequenceAction(name, actions...), nil
}

// validationActionBuilder creates a ValidationAction from parameters.
// Validation actions require custom validator functions that cannot be specified in YAML.
func validationActionBuilder(_ *ActionFactory, name string, params map[string]any) (Action, error) {
	// Validation actions must be created programmatically
	return nil, ErrValidationMustBeProgrammatic
}

// conditionalActionBuilder creates a ConditionalAction from parameters.
// Conditional actions require condition functions that cannot be specified in YAML.
func conditionalActionBuilder(_ *ActionFactory, name string, params map[string]any) (Action, error) {
	// Conditional actions must be created programmatically
	return nil, ErrConditionalMustBeProgrammatic
}

// retryActionBuilder creates a RetryAction from parameters.
func retryActionBuilder(factory *ActionFactory, name string, params map[string]any) (Action, error) {
	// Extract action to retry
	actionParam, ok := params["action"].(map[string]any)
	if !ok {
		return nil, ErrRetryActionRequired
	}

	// Extract retry parameters
	maxRetries, ok := params["maxRetries"].(int)
	if !ok {
		maxRetries = 3 // default
	}

	backoffMs, ok := params["backoffMs"].(int)
	if !ok {
		backoffMs = 1000 // default 1 second
	}

	// Convert to ActionConfig
	config := ActionConfig{
		Type:       fmt.Sprintf("%v", actionParam["type"]),
		Name:       fmt.Sprintf("%v", actionParam["name"]),
		Parameters: make(map[string]any),
	}

	if params, ok := actionParam["parameters"].(map[string]any); ok {
		config.Parameters = params
	}

	// Reuse the factory to preserve custom action builders
	action, err := factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create action to retry: %w", err)
	}

	return NewRetryAction(name, action, maxRetries, time.Duration(backoffMs)*time.Millisecond), nil
}
