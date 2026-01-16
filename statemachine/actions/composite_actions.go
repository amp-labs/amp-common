package actions

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/amp-labs/amp-common/statemachine"
)

// ValidationFunc is a generic validation function.
type ValidationFunc func(ctx context.Context, data any) (valid bool, feedback string, err error)

var (
	// ErrBothActionsFailed is returned when both primary and fallback actions fail.
	ErrBothActionsFailed = errors.New("both primary and fallback actions failed")
	// ErrValidationFailed is returned when a validation check fails.
	ErrValidationFailed = errors.New("validation failed")
	// ErrBranchFailed is returned when a conditional branch fails.
	ErrBranchFailed = errors.New("branch failed")
	// ErrActionFailedAfterRetries is returned when an action fails after all retry attempts.
	ErrActionFailedAfterRetries = errors.New("action failed after retries")
	// ErrSomeActionsFailed is returned when some parallel actions fail.
	ErrSomeActionsFailed = errors.New("some actions failed")
	// ErrConditionNotMet is returned when a required condition is not met.
	ErrConditionNotMet = errors.New("condition not met")
	// ErrStepFailed is returned when a step in a sequence fails.
	ErrStepFailed = errors.New("step failed")
	// ErrValidationDataNotFound is returned when validation data is not found in context.
	ErrValidationDataNotFound = errors.New("validation data not found in context")
	// ErrSamplingNotAvailable is returned when AI sampling is not available.
	ErrSamplingNotAvailable = errors.New("sampling not available")
	// ErrPrimaryFailed is returned when a primary action fails (test error).
	ErrPrimaryFailed = errors.New("primary failed")
	// ErrTemporaryError is returned for temporary errors (test error).
	ErrTemporaryError = errors.New("temporary error")
	// ErrUserDeclined is returned when user declines an action.
	ErrUserDeclined = errors.New("user declined")
)

// TryWithFallback tries a primary action and falls back to secondary on error.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - primaryAction: The primary action to try (required)
//   - fallbackAction: The fallback action to execute on error (required)
//   - catchErrors: If true, catch all errors; if false, only catch specific errors
//
// Context outputs:
//   - {name}_source: "primary" or "fallback" indicating which action succeeded
//   - {name}_error: Error from primary action (if it failed)
//
// Example:
//
//	action := &TryWithFallback{
//	    Name: "get_help",
//	    PrimaryAction: &SampleWithFallback{...}, // Try AI
//	    FallbackAction: &ElicitForm{...},        // Fall back to asking user
//	}
type TryWithFallback struct {
	name           string
	PrimaryAction  statemachine.Action
	FallbackAction statemachine.Action
	CatchErrors    bool
}

// NewTryWithFallback creates a new TryWithFallback action.
func NewTryWithFallback(
	name string,
	primaryAction, fallbackAction statemachine.Action,
	catchErrors bool,
) *TryWithFallback {
	return &TryWithFallback{
		name:           name,
		PrimaryAction:  primaryAction,
		FallbackAction: fallbackAction,
		CatchErrors:    catchErrors,
	}
}

func (a *TryWithFallback) Name() string {
	return a.name
}

func (a *TryWithFallback) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	// Try primary action
	err := a.PrimaryAction.Execute(ctx, smCtx)
	if err == nil {
		smCtx.Set(a.name+"_source", "primary")

		return nil
	}

	// Store error
	smCtx.Set(a.name+"_error", err.Error())

	// Execute fallback
	fallbackErr := a.FallbackAction.Execute(ctx, smCtx)
	if fallbackErr != nil {
		return fmt.Errorf("%w: primary=%w, fallback=%w", ErrBothActionsFailed, err, fallbackErr)
	}

	smCtx.Set(a.name+"_source", "fallback")

	return nil
}

// ValidatedSequence executes actions in sequence with validation between steps.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - actions: List of actions to execute (required)
//   - validators: Map of step index to validator function (optional)
//   - continueOnError: If true, continue even if validation fails (default: false)
//
// Context outputs:
//   - {name}_completed_steps: Number of successfully completed steps
//   - {name}_failed_at: Step index where validation failed (-1 if all passed)
//   - {name}_validations: Map of step index to validation result
//
// Example:
//
//	action := &ValidatedSequence{
//	    Name: "oauth_setup",
//	    Actions: []Action{
//	        &ElicitForm{...},        // Step 0: Get credentials
//	        &ValidateInput{...},     // Step 1: Validate
//	        &ElicitConfirmation{...}, // Step 2: Confirm
//	    },
//	    Validators: map[int]ValidationFunc{
//	        0: func(ctx context.Context, data any) (bool, string, error) {
//	            // Validate step 0 completed
//	            return true, "", nil
//	        },
//	    },
//	}
type ValidatedSequence struct {
	name            string
	Actions         []statemachine.Action
	Validators      map[int]ValidationFunc
	ContinueOnError bool
}

// NewValidatedSequence creates a new ValidatedSequence action.
func NewValidatedSequence(
	name string,
	actions []statemachine.Action,
	validators map[int]ValidationFunc,
	continueOnError bool,
) *ValidatedSequence {
	return &ValidatedSequence{
		name:            name,
		Actions:         actions,
		Validators:      validators,
		ContinueOnError: continueOnError,
	}
}

func (a *ValidatedSequence) Name() string {
	return a.name
}

func (a *ValidatedSequence) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	validations := make(map[int]bool)
	completedSteps := 0
	failedAt := -1

	for idx, action := range a.Actions {
		// Execute action
		err := action.Execute(ctx, smCtx)
		if err != nil && !a.ContinueOnError {
			failedAt = idx

			break
		}

		// Run validator if exists
		if validator, exists := a.Validators[idx]; exists {
			valid, feedback, err := validator(ctx, smCtx)
			if err != nil {
				return fmt.Errorf("%w at step %d: %w", ErrValidationFailed, idx, err)
			}

			validations[idx] = valid

			if !valid && !a.ContinueOnError {
				smCtx.Set(a.name+"_validation_feedback", feedback)

				failedAt = idx

				break
			}
		}

		completedSteps++
	}

	// Store results
	smCtx.Set(a.name+"_completed_steps", completedSteps)
	smCtx.Set(a.name+"_failed_at", failedAt)
	smCtx.Set(a.name+"_validations", validations)

	return nil
}

// ConditionalBranch executes the first matching branch based on conditions.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - branches: List of condition-action pairs (required)
//   - defaultAction: Action to execute if no conditions match (optional)
//
// Context outputs:
//   - {name}_branch_taken: Index of the branch that was executed (-1 for default)
//   - {name}_condition_results: Array of condition evaluation results
//
// Example:
//
//	action := &ConditionalBranch{
//	    Name: "provider_specific",
//	    Branches: []Branch{
//	        {
//	            Condition: func(c *statemachine.Context) bool {
//	                provider, _ := c.GetString("provider")
//	                return provider == "salesforce"
//	            },
//	            Action: &SampleWithFallback{...}, // Salesforce-specific
//	        },
//	        {
//	            Condition: func(c *statemachine.Context) bool {
//	                provider, _ := c.GetString("provider")
//	                return provider == "hubspot"
//	            },
//	            Action: &ElicitForm{...}, // HubSpot-specific
//	        },
//	    },
//	    DefaultAction: &SampleWithFallback{...}, // Generic
//	}
type ConditionalBranch struct {
	name          string
	Branches      []Branch
	DefaultAction statemachine.Action
}

// NewConditionalBranch creates a new ConditionalBranch action.
func NewConditionalBranch(name string, branches []Branch, defaultAction statemachine.Action) *ConditionalBranch {
	return &ConditionalBranch{
		name:          name,
		Branches:      branches,
		DefaultAction: defaultAction,
	}
}

func (a *ConditionalBranch) Name() string {
	return a.name
}

// Branch represents a condition-action pair.
type Branch struct {
	Condition func(c *statemachine.Context) bool
	Action    statemachine.Action
}

func (a *ConditionalBranch) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	conditionResults := make([]bool, len(a.Branches))
	branchTaken := -1

	// Evaluate conditions in order
	for i, branch := range a.Branches {
		result := branch.Condition(smCtx)
		conditionResults[i] = result

		if result && branchTaken == -1 {
			// Execute first matching branch
			err := branch.Action.Execute(ctx, smCtx)
			if err != nil {
				return fmt.Errorf("%w %d: %w", ErrBranchFailed, i, err)
			}

			branchTaken = i

			break
		}
	}

	// Execute default if no match
	if branchTaken == -1 && a.DefaultAction != nil {
		err := a.DefaultAction.Execute(ctx, smCtx)
		if err != nil {
			return fmt.Errorf("default %w: %w", ErrBranchFailed, err)
		}
	}

	// Store results
	smCtx.Set(a.name+"_branch_taken", branchTaken)
	smCtx.Set(a.name+"_condition_results", conditionResults)

	return nil
}

// RetryWithBackoff retries an action with exponential backoff.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - action: The action to retry (required)
//   - maxAttempts: Maximum retry attempts (default: 3)
//   - initialDelay: Initial delay duration (default: 1s)
//   - maxDelay: Maximum delay duration (default: 30s)
//   - backoffMultiplier: Backoff multiplier (default: 2.0)
//   - retryCondition: Function to determine if error should be retried (optional)
//
// Context outputs:
//   - {name}_attempts: Number of attempts made
//   - {name}_delays: Array of delay durations used
//   - {name}_succeeded: Boolean indicating if action eventually succeeded
//
// Example:
//
//	action := &RetryWithBackoff{
//	    Name: "fetch_data",
//	    Action: &SampleWithFallback{...},
//	    MaxAttempts: 5,
//	    InitialDelay: 2 * time.Second,
//	    RetryCondition: func(err error) bool {
//	        // Only retry on transient errors
//	        return isTransientError(err)
//	    },
//	}
type RetryWithBackoff struct {
	name              string
	Action            statemachine.Action
	MaxAttempts       int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
	RetryCondition    func(error) bool
}

// NewRetryWithBackoff creates a new RetryWithBackoff action.
func NewRetryWithBackoff(
	name string,
	action statemachine.Action,
	maxAttempts int,
	initialDelay, maxDelay time.Duration,
	backoffMultiplier float64,
	retryCondition func(error) bool,
) *RetryWithBackoff {
	return &RetryWithBackoff{
		name:              name,
		Action:            action,
		MaxAttempts:       maxAttempts,
		InitialDelay:      initialDelay,
		MaxDelay:          maxDelay,
		BackoffMultiplier: backoffMultiplier,
		RetryCondition:    retryCondition,
	}
}

func (a *RetryWithBackoff) Name() string {
	return a.name
}

func (a *RetryWithBackoff) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	maxAttempts := a.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}

	initialDelay := a.InitialDelay
	if initialDelay == 0 {
		initialDelay = 1 * time.Second
	}

	maxDelay := a.MaxDelay
	if maxDelay == 0 {
		maxDelay = 30 * time.Second //nolint:mnd // Reasonable default max delay for exponential backoff
	}

	multiplier := a.BackoffMultiplier
	if multiplier == 0 {
		multiplier = 2.0
	}

	var delays []time.Duration

	currentDelay := initialDelay
	attempts := 0

	for attempts < maxAttempts {
		attempts++

		err := a.Action.Execute(ctx, smCtx)
		if err == nil {
			smCtx.Set(a.name+"_attempts", attempts)
			smCtx.Set(a.name+"_delays", delays)
			smCtx.Set(a.name+"_succeeded", true)

			return nil
		}

		// Check if we should retry
		if a.RetryCondition != nil && !a.RetryCondition(err) {
			// Error is not retryable
			smCtx.Set(a.name+"_attempts", attempts)
			smCtx.Set(a.name+"_succeeded", false)

			return err
		}

		// Don't delay after last attempt
		if attempts < maxAttempts {
			delays = append(delays, currentDelay)

			// Wait with context cancellation support
			timer := time.NewTimer(currentDelay)
			select {
			case <-ctx.Done():
				timer.Stop()

				return ctx.Err()
			case <-timer.C:
			}

			// Calculate next delay
			currentDelay = min(time.Duration(float64(currentDelay)*multiplier), maxDelay)
		}
	}

	smCtx.Set(a.name+"_attempts", attempts)
	smCtx.Set(a.name+"_delays", delays)
	smCtx.Set(a.name+"_succeeded", false)

	return fmt.Errorf("%w: %d attempts", ErrActionFailedAfterRetries, maxAttempts)
}

// ParallelWithMerge executes actions in parallel and merges results.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - actions: List of actions to execute in parallel (required)
//   - mergeStrategy: Strategy for merging results (first, all, majority)
//   - continueOnError: If true, continue even if some actions fail
//
// Context outputs:
//   - {name}_results: Array of results from all actions
//   - {name}_failures: Array of errors from failed actions
//   - {name}_success_count: Number of successful actions
//
// Example:
//
//	action := &ParallelWithMerge{
//	    Name: "multi_provider_check",
//	    Actions: []Action{
//	        &ValidateInput{...}, // Check Salesforce
//	        &ValidateInput{...}, // Check HubSpot
//	        &ValidateInput{...}, // Check Notion
//	    },
//	    MergeStrategy: "majority",
//	    ContinueOnError: true,
//	}
type ParallelWithMerge struct {
	name            string
	Actions         []statemachine.Action
	MergeStrategy   string // "first", "all", "majority"
	ContinueOnError bool
}

// NewParallelWithMerge creates a new ParallelWithMerge action.
func NewParallelWithMerge(
	name string,
	actions []statemachine.Action,
	mergeStrategy string,
	continueOnError bool,
) *ParallelWithMerge {
	return &ParallelWithMerge{
		name:            name,
		Actions:         actions,
		MergeStrategy:   mergeStrategy,
		ContinueOnError: continueOnError,
	}
}

func (a *ParallelWithMerge) Name() string {
	return a.name
}

func (a *ParallelWithMerge) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	type result struct {
		index      int
		err        error
		clonedCtx  *statemachine.Context
		actionName string
	}

	results := make(chan result, len(a.Actions))

	// Execute actions in parallel
	for i, action := range a.Actions {
		go func(idx int, act statemachine.Action) {
			// Clone context for parallel execution to avoid race conditions
			clonedCtx := smCtx.Clone()
			err := act.Execute(ctx, clonedCtx)

			// Send result back with cloned context for sequential merging
			results <- result{
				index:      idx,
				err:        err,
				clonedCtx:  clonedCtx,
				actionName: act.Name(),
			}
		}(i, action)
	}

	// Collect all results sequentially
	collectedResults := make([]result, 0, len(a.Actions))

	var failures []string

	successCount := 0

	for range a.Actions {
		res := <-results
		collectedResults = append(collectedResults, res)

		if res.err != nil {
			failures = append(failures, fmt.Sprintf("Action %d (%s): %s", res.index, res.actionName, res.err.Error()))
		} else {
			successCount++
		}
	}

	// Apply merge strategy sequentially (no race conditions)
	var mergedData map[string]any

	switch a.MergeStrategy {
	case "first":
		// Take the first successful result
		for _, res := range collectedResults {
			if res.err == nil {
				mergedData = res.clonedCtx.Data

				break
			}
		}

	case "majority":
		// Only merge if majority succeeded
		if successCount > len(a.Actions)/2 {
			mergedData = make(map[string]any)

			for _, res := range collectedResults {
				if res.err == nil {
					maps.Copy(mergedData, res.clonedCtx.Data)
				}
			}
		}

	case "all":
		fallthrough
	default:
		// Merge all successful results
		mergedData = make(map[string]any)

		for _, res := range collectedResults {
			if res.err == nil {
				maps.Copy(mergedData, res.clonedCtx.Data)
			}
		}
	}

	// Apply merged data to main context (sequential, no race)
	if mergedData != nil {
		smCtx.Merge(mergedData)
	}

	// Store results consistently
	smCtx.Set(a.name+"_results", mergedData)
	smCtx.Set(a.name+"_failures", failures)
	smCtx.Set(a.name+"_success_count", successCount)

	// Check if we should fail based on strategy
	if !a.ContinueOnError && len(failures) > 0 {
		return fmt.Errorf("%w: %v", ErrSomeActionsFailed, failures)
	}

	return nil
}

// ProgressiveDisclosure performs multi-step elicitation with conditional steps.
//
// Parameters:
//   - name: The name for storing results in context (required)
//   - steps: List of progressive disclosure steps (required)
//
// Context outputs:
//   - {name}_steps_completed: Number of steps completed
//   - {name}_all_data: Combined data from all steps
//
// Example:
//
//	action := &ProgressiveDisclosure{
//	    Name: "oauth_setup",
//	    Steps: []DisclosureStep{
//	        {
//	            Action: &ElicitForm{...}, // Basic OAuth fields
//	            Required: true,
//	        },
//	        {
//	            Action: &ElicitForm{...}, // Advanced scopes
//	            Required: false,
//	            Condition: func(c *statemachine.Context) bool {
//	                advanced, _ := c.GetBool("want_advanced")
//	                return advanced
//	            },
//	        },
//	    },
//	}
type ProgressiveDisclosure struct {
	name  string
	Steps []DisclosureStep
}

// NewProgressiveDisclosure creates a new ProgressiveDisclosure action.
func NewProgressiveDisclosure(name string, steps []DisclosureStep) *ProgressiveDisclosure {
	return &ProgressiveDisclosure{
		name:  name,
		Steps: steps,
	}
}

func (a *ProgressiveDisclosure) Name() string {
	return a.name
}

// DisclosureStep represents a step in progressive disclosure.
type DisclosureStep struct {
	Action    statemachine.Action
	Required  bool
	Condition func(c *statemachine.Context) bool
}

func (a *ProgressiveDisclosure) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	allData := make(map[string]any)
	stepsCompleted := 0

	for i, step := range a.Steps {
		// Check if step should be executed
		if step.Condition != nil && !step.Condition(smCtx) {
			if step.Required {
				return fmt.Errorf("required step %d: %w", i, ErrConditionNotMet)
			}

			continue
		}

		// Execute step
		err := step.Action.Execute(ctx, smCtx)
		if err != nil {
			if step.Required {
				return fmt.Errorf("required step %d: %w: %w", i, ErrStepFailed, err)
			}
			// Optional step failed, continue
			continue
		}

		stepsCompleted++

		// Collect data from this step
		// This is a simplified version - in practice, you'd want to track
		// which data came from which step
		maps.Copy(allData, smCtx.Data)
	}

	// Store results
	smCtx.Set(a.name+"_steps_completed", stepsCompleted)
	smCtx.Set(a.name+"_all_data", allData)

	return nil
}
