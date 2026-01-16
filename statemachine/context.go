package statemachine

import (
	"maps"
	"sync"
	"time"
)

// Context is a thread-safe context object that carries data between states.
type Context struct {
	mu             sync.RWMutex
	SessionID      string
	ProjectID      string
	CurrentState   string
	Data           map[string]any
	History        []StateTransition
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Provider       string   // Provider being configured (e.g., "salesforce", "hubspot")
	ContextChunkID string   // Unique ID for this conversation chunk/interaction
	ToolName       string   // Tool name (e.g., "guided_setup", "integration_doctor")
	PathHistory    []string // Ordered list of states visited (append CurrentState on entry)
}

// StateTransition records a transition in the state machine history.
type StateTransition struct {
	From      string
	To        string
	Timestamp time.Time
	Data      map[string]any
}

// NewContext creates a new state machine context.
func NewContext(sessionID, projectID string) *Context {
	now := time.Now()

	return &Context{
		SessionID:    sessionID,
		ProjectID:    projectID,
		CurrentState: "",
		Data:         make(map[string]any),
		History:      []StateTransition{},
		Metadata:     make(map[string]any),
		CreatedAt:    now,
		UpdatedAt:    now,
		PathHistory:  []string{},
	}
}

// Get retrieves a value from the context data.
func (c *Context) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.Data[key]

	return val, ok
}

// Set stores a value in the context data.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Data[key] = value
	c.UpdatedAt = time.Now()
}

// GetString retrieves a string value from the context data.
func (c *Context) GetString(key string) (string, bool) {
	val, ok := c.Get(key)
	if !ok {
		return "", false
	}

	str, ok := val.(string)

	return str, ok
}

// GetBool retrieves a boolean value from the context data.
func (c *Context) GetBool(key string) (bool, bool) {
	val, ok := c.Get(key)
	if !ok {
		return false, false
	}

	b, ok := val.(bool)

	return b, ok
}

// GetInt retrieves an integer value from the context data.
func (c *Context) GetInt(key string) (int, bool) {
	val, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	i, ok := val.(int)

	return i, ok
}

// Merge merges a map of data into the context.
func (c *Context) Merge(data map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	maps.Copy(c.Data, data)

	c.UpdatedAt = time.Now()
}

// Clone creates a deep copy of the context.
func (c *Context) Clone() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	clone := &Context{
		SessionID:      c.SessionID,
		ProjectID:      c.ProjectID,
		CurrentState:   c.CurrentState,
		Data:           make(map[string]any),
		History:        make([]StateTransition, len(c.History)),
		Metadata:       make(map[string]any),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		Provider:       c.Provider,
		ContextChunkID: c.ContextChunkID,
		ToolName:       c.ToolName,
		PathHistory:    make([]string, len(c.PathHistory)),
	}

	// Deep copy data
	maps.Copy(clone.Data, c.Data)

	// Deep copy history
	copy(clone.History, c.History)

	// Deep copy metadata
	maps.Copy(clone.Metadata, c.Metadata)

	// Deep copy path history
	copy(clone.PathHistory, c.PathHistory)

	return clone
}

// AddTransition records a state transition in the history.
func (c *Context) AddTransition(from, to string, data map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	transition := StateTransition{
		From:      from,
		To:        to,
		Timestamp: time.Now(),
		Data:      make(map[string]any),
	}

	// Copy transition data
	maps.Copy(transition.Data, data)

	c.History = append(c.History, transition)
	c.UpdatedAt = time.Now()
}

// AppendToPath adds the current state to the path history.
func (c *Context) AppendToPath(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.PathHistory = append(c.PathHistory, state)
	c.UpdatedAt = time.Now()
}
