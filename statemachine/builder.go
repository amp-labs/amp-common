package statemachine

import "context"

// Builder provides a fluent API for constructing state machines.
type Builder struct {
	config             *Config
	factory            *ActionFactory
	programmaticStates map[string]State // stores states created programmatically
}

// NewBuilder creates a new state machine builder.
func NewBuilder(name string) *Builder {
	return &Builder{
		config: &Config{
			Name:        name,
			States:      []StateConfig{},
			Transitions: []TransitionConfig{},
		},
		factory:            NewActionFactory(),
		programmaticStates: make(map[string]State),
	}
}

// WithInitialState sets the initial state.
func (b *Builder) WithInitialState(state string) *Builder {
	b.config.InitialState = state

	return b
}

// WithFinalStates sets the final states.
func (b *Builder) WithFinalStates(states ...string) *Builder {
	b.config.FinalStates = states

	return b
}

// AddState adds a state configuration.
func (b *Builder) AddState(config StateConfig) *Builder {
	b.config.States = append(b.config.States, config)

	return b
}

// AddTransition adds a transition configuration.
func (b *Builder) AddTransition(config TransitionConfig) *Builder {
	b.config.Transitions = append(b.config.Transitions, config)

	return b
}

// AddActionState adds a simple action state.
func (b *Builder) AddActionState(name string, action Action, next string) *Builder {
	// Store the programmatic action state
	actionState := NewActionState(name, action, next)
	b.programmaticStates[name] = actionState

	// Create a placeholder StateConfig with a dummy action to pass validation
	// This will be overridden by the programmatic state in Build()
	stateConfig := StateConfig{
		Name: name,
		Type: "action",
		Actions: []ActionConfig{
			{
				Type:       "sampling", // placeholder type that can be built
				Name:       name + "_placeholder",
				Parameters: map[string]any{"user": "placeholder"},
			},
		},
	}

	b.config.States = append(b.config.States, stateConfig)

	// Add automatic transition if next state is specified
	if next != "" {
		b.AddTransition(TransitionConfig{
			From:      name,
			To:        next,
			Condition: "always",
		})
	}

	return b
}

// AddConditionalState adds a conditional state.
func (b *Builder) AddConditionalState(name string, cond func(*Context) (string, error)) *Builder {
	// Store the programmatic conditional state with context adapter
	conditionalState := NewConditionalState(name, func(_ context.Context, smCtx *Context) (string, error) {
		return cond(smCtx)
	})
	b.programmaticStates[name] = conditionalState

	// Create a placeholder StateConfig with a dummy action to pass validation
	// Conditional type would fail validation, so use action type with placeholder
	stateConfig := StateConfig{
		Name: name,
		Type: "action",
		Actions: []ActionConfig{
			{
				Type:       "sampling", // placeholder type that can be built
				Name:       name + "_placeholder",
				Parameters: map[string]any{"user": "placeholder"},
			},
		},
	}

	b.config.States = append(b.config.States, stateConfig)

	return b
}

// Build constructs the state machine engine.
func (b *Builder) Build() (*Engine, error) {
	// Pass the builder's factory to preserve custom action builders
	engine, err := NewEngine(b.config, b.factory)
	if err != nil {
		return nil, err
	}

	// Override placeholder states with programmatic states
	for _, state := range b.programmaticStates {
		engine.RegisterState(state)
	}

	return engine, nil
}

// RegisterActionBuilder registers a custom action builder.
func (b *Builder) RegisterActionBuilder(actionType string, builder ActionBuilder) *Builder {
	b.factory.Register(actionType, builder)

	return b
}
