package statemachine

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("sequence action requires 'actions' parameter")
	}

	var actions []Action
	for i, actionParam := range actionsParam {
		actionMap, ok := actionParam.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("action %d: invalid action format", i)
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
			return nil, fmt.Errorf("failed to create action %d: %w", i, err)
		}

		actions = append(actions, action)
	}

	return NewSequenceAction(name, actions...), nil
}
