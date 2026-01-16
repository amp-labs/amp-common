package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/amp-labs/amp-common/statemachine"
)

// ActionTracer tracks action execution for debugging.
type ActionTracer struct {
	traces []ActionTrace
}

// ActionTrace represents a single action execution trace.
type ActionTrace struct {
	ActionType string
	StartState map[string]any
	EndState   map[string]any
	Error      error
}

// NewActionTracer creates a new action tracer.
func NewActionTracer() *ActionTracer {
	return &ActionTracer{
		traces: make([]ActionTrace, 0),
	}
}

// TraceAction captures the execution of an action.
func (t *ActionTracer) TraceAction(actionType string, ctx *statemachine.Context, fn func() error) error {
	// Capture start state
	startState := make(map[string]any)
	maps.Copy(startState, ctx.Data)

	// Execute action
	err := fn()

	// Capture end state
	endState := make(map[string]any)
	maps.Copy(endState, ctx.Data)

	// Record trace
	t.traces = append(t.traces, ActionTrace{
		ActionType: actionType,
		StartState: startState,
		EndState:   endState,
		Error:      err,
	})

	return err
}

// GetTraces returns all execution traces.
func (t *ActionTracer) GetTraces() []ActionTrace {
	return t.traces
}

// PrintTraces prints all traces to stdout.
func (t *ActionTracer) PrintTraces() {
	fmt.Println("=== Action Execution Traces ===") //nolint:forbidigo // Debug tracer requires console output

	for i, trace := range t.traces {
		fmt.Printf("\n[%d] %s\n", i, trace.ActionType) //nolint:forbidigo // Debug tracer requires console output

		if trace.Error != nil {
			fmt.Printf("  Error: %v\n", trace.Error) //nolint:forbidigo // Debug tracer requires console output
		}

		// Show state changes
		fmt.Println("  State Changes:") //nolint:forbidigo // Debug tracer requires console output

		for key := range trace.EndState {
			startVal, hadBefore := trace.StartState[key]
			endVal := trace.EndState[key]

			if !hadBefore {
				fmt.Printf("    + %s: %v\n", key, endVal) //nolint:forbidigo // Debug tracer requires console output
			} else if startVal != endVal {
				fmt.Printf("    ~ %s: %v -> %v\n", key, startVal, endVal) //nolint:forbidigo // Debug tracer requires console output
			}
		}

		// Show removed keys
		for key := range trace.StartState {
			if _, stillExists := trace.EndState[key]; !stillExists {
				fmt.Printf("    - %s\n", key) //nolint:forbidigo // Debug tracer requires console output
			}
		}
	}

	fmt.Println("\n===============================") //nolint:forbidigo // Debug tracer requires console output
}

// DumpContext dumps context state for debugging.
func DumpContext(ctx *statemachine.Context) string {
	var builder strings.Builder

	builder.WriteString("=== Context State ===\n")
	builder.WriteString(fmt.Sprintf("Session ID: %s\n", ctx.SessionID))
	builder.WriteString(fmt.Sprintf("Project ID: %s\n", ctx.ProjectID))
	builder.WriteString(fmt.Sprintf("Current State: %s\n", ctx.CurrentState))
	builder.WriteString("\nData:\n")

	// Pretty print data
	for key, val := range ctx.Data {
		builder.WriteString(fmt.Sprintf("  %s: ", key))

		// Try to format as JSON for complex types
		if _, isSimple := val.(string); !isSimple { //nolint:nestif // Checking multiple simple types sequentially
			if _, isSimple := val.(int); !isSimple {
				if _, isSimple := val.(bool); !isSimple {
					jsonBytes, err := json.MarshalIndent(val, "    ", "  ")
					if err == nil {
						builder.WriteString("\n    ")
						builder.Write(jsonBytes)
						builder.WriteString("\n")

						continue
					}
				}
			}
		}

		builder.WriteString(fmt.Sprintf("%v\n", val))
	}

	builder.WriteString("\nHistory:\n")

	for historyIdx, transition := range ctx.History {
		builder.WriteString(fmt.Sprintf("  [%d] %s -> %s (%s)\n",
			historyIdx, transition.From, transition.To, transition.Timestamp.Format("15:04:05")))
	}

	builder.WriteString("====================\n")

	return builder.String()
}

// VisualizeDependencies creates a text visualization of action dependencies.
func VisualizeDependencies(actions map[string][]string) string {
	var builder strings.Builder

	builder.WriteString("=== Action Dependencies ===\n\n")

	for action, deps := range actions {
		builder.WriteString(action + "\n")

		if len(deps) == 0 {
			builder.WriteString("  (no dependencies)\n")
		} else {
			for depIdx, dep := range deps {
				if depIdx == len(deps)-1 {
					builder.WriteString(fmt.Sprintf("  └─ %s\n", dep))
				} else {
					builder.WriteString(fmt.Sprintf("  ├─ %s\n", dep))
				}
			}
		}

		builder.WriteString("\n")
	}

	builder.WriteString("==========================\n")

	return builder.String()
}

// CompareContexts compares two contexts and shows differences.
func CompareContexts(before, after *statemachine.Context) string {
	var builder strings.Builder

	builder.WriteString("=== Context Diff ===\n\n")

	// Check state change
	if before.CurrentState != after.CurrentState {
		builder.WriteString(fmt.Sprintf("State: %s -> %s\n\n", before.CurrentState, after.CurrentState))
	}

	// Check data changes
	builder.WriteString("Data Changes:\n")

	// Added/Modified keys
	for key, afterVal := range after.Data {
		beforeVal, existed := before.Data[key]

		if !existed {
			builder.WriteString(fmt.Sprintf("  + %s: %v\n", key, afterVal))
		} else if fmt.Sprintf("%v", beforeVal) != fmt.Sprintf("%v", afterVal) {
			builder.WriteString(fmt.Sprintf("  ~ %s: %v -> %v\n", key, beforeVal, afterVal))
		}
	}

	// Removed keys
	for key := range before.Data {
		if _, exists := after.Data[key]; !exists {
			builder.WriteString(fmt.Sprintf("  - %s\n", key))
		}
	}

	// New transitions
	if len(after.History) > len(before.History) {
		builder.WriteString("\nNew Transitions:\n")

		for transIdx := len(before.History); transIdx < len(after.History); transIdx++ {
			t := after.History[transIdx]
			builder.WriteString(fmt.Sprintf("  %s -> %s (%s)\n",
				t.From, t.To, t.Timestamp.Format("15:04:05")))
		}
	}

	builder.WriteString("\n===================\n")

	return builder.String()
}

// LogAction logs an action execution for debugging.
func LogAction(actionType string, data map[string]any) {
	fmt.Printf("[DEBUG] Executing: %s\n", actionType) //nolint:forbidigo // Debug logging requires console output

	if len(data) > 0 {
		fmt.Println("  Parameters:") //nolint:forbidigo // Debug logging requires console output

		for k, v := range data {
			fmt.Printf("    %s: %v\n", k, v) //nolint:forbidigo // Debug logging requires console output
		}
	}
}

// BreakpointAction creates an action that pauses execution for debugging.
func BreakpointAction(name string) statemachine.Action {
	return &MockAction{
		name: name,
		ExecuteFn: func(ctx context.Context, c *statemachine.Context) error {
			fmt.Printf("\n=== BREAKPOINT: %s ===\n", name) //nolint:forbidigo // Debug action requires console output
			fmt.Println(DumpContext(c))                    //nolint:forbidigo // Debug action requires console output
			fmt.Print("Press Enter to continue...")        //nolint:forbidigo // Debug action requires user interaction

			var input string

			_, _ = fmt.Scanln(&input) //nolint:errcheck // Best-effort read for debug breakpoint

			return nil
		},
	}
}
