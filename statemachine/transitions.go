package statemachine

import (
	"context"
	"fmt"
	"strings"
)

const expectedExpressionParts = 2

// SimpleTransition always transitions from A to B.
type SimpleTransition struct {
	from string
	to   string
}

// NewSimpleTransition creates a new simple transition.
func NewSimpleTransition(from, to string) *SimpleTransition {
	return &SimpleTransition{
		from: from,
		to:   to,
	}
}

func (t *SimpleTransition) From() string {
	return t.from
}

func (t *SimpleTransition) To() string {
	return t.to
}

func (t *SimpleTransition) Condition(ctx context.Context, smCtx *Context) (bool, error) {
	return true, nil
}

// ConditionalTransition checks context before transitioning.
type ConditionalTransition struct {
	from      string
	to        string
	condition func(ctx context.Context, smCtx *Context) (bool, error)
}

// NewConditionalTransition creates a new conditional transition.
func NewConditionalTransition(
	from, to string,
	cond func(ctx context.Context, smCtx *Context) (bool, error),
) *ConditionalTransition {
	return &ConditionalTransition{
		from:      from,
		to:        to,
		condition: cond,
	}
}

func (t *ConditionalTransition) From() string {
	return t.from
}

func (t *ConditionalTransition) To() string {
	return t.to
}

func (t *ConditionalTransition) Condition(ctx context.Context, smCtx *Context) (bool, error) {
	return t.condition(ctx, smCtx)
}

// ExpressionTransition evaluates expression against context.
type ExpressionTransition struct {
	from       string
	to         string
	expression string // e.g., "data.provider == 'salesforce'"
}

// NewExpressionTransition creates a new expression-based transition.
func NewExpressionTransition(from, to, expr string) *ExpressionTransition {
	return &ExpressionTransition{
		from:       from,
		to:         to,
		expression: expr,
	}
}

func (t *ExpressionTransition) From() string {
	return t.from
}

func (t *ExpressionTransition) To() string {
	return t.to
}

func (t *ExpressionTransition) Condition(ctx context.Context, smCtx *Context) (bool, error) {
	// Simple expression evaluator for common patterns
	// In production, could use github.com/antonmedv/expr or similar
	return evaluateExpression(t.expression, smCtx)
}

// evaluateExpression evaluates simple expressions against context
// This is a basic implementation - production should use a proper expression library.
func evaluateExpression(expr string, smCtx *Context) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle simple equality checks: "data.key == 'value'"
	if strings.Contains(expr, "==") {
		parts := strings.Split(expr, "==")
		if len(parts) != expectedExpressionParts {
			return false, fmt.Errorf("%w: %s", ErrInvalidExpression, expr)
		}

		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		// Remove quotes from right side
		right = strings.Trim(right, "'\"")

		// Extract key from "data.key" format
		key := strings.TrimPrefix(left, "data.")

		// Get value from context
		value, exists := smCtx.Get(key)
		if !exists {
			return false, nil
		}

		// Compare as strings
		return fmt.Sprintf("%v", value) == right, nil
	}

	// Handle simple inequality checks: "data.key != 'value'"
	if strings.Contains(expr, "!=") {
		parts := strings.Split(expr, "!=")
		if len(parts) != expectedExpressionParts {
			return false, fmt.Errorf("%w: %s", ErrInvalidExpression, expr)
		}

		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		right = strings.Trim(right, "'\"")

		key := strings.TrimPrefix(left, "data.")

		value, exists := smCtx.Get(key)
		if !exists {
			return true, nil // If key doesn't exist, it's not equal
		}

		return fmt.Sprintf("%v", value) != right, nil
	}

	// Handle boolean checks: "data.key"
	if after, ok := strings.CutPrefix(expr, "data."); ok {
		key := after

		value, exists := smCtx.GetBool(key)
		if !exists {
			return false, nil
		}

		return value, nil
	}

	// Handle negation: "!data.key"
	if after, ok := strings.CutPrefix(expr, "!data."); ok {
		key := after

		value, exists := smCtx.GetBool(key)
		if !exists {
			return true, nil
		}

		return !value, nil
	}

	return false, fmt.Errorf("%w: %s", ErrUnsupportedExpression, expr)
}
