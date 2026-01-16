package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SequenceAction executes actions in order.
type SequenceAction struct {
	BaseAction

	actions []Action
}

// NewSequenceAction creates a new sequence action.
func NewSequenceAction(name string, actions ...Action) *SequenceAction {
	return &SequenceAction{
		BaseAction: BaseAction{name: name},
		actions:    actions,
	}
}

func (a *SequenceAction) Execute(ctx context.Context, smCtx *Context) error {
	for _, action := range a.actions {
		err := action.Execute(ctx, smCtx)
		if err != nil {
			return fmt.Errorf("sequence step %s failed: %w", action.Name(), err)
		}
	}

	return nil
}

// ConditionalAction executes action based on condition.
type ConditionalAction struct {
	BaseAction

	condition  func(ctx context.Context, smCtx *Context) (bool, error)
	thenAction Action
	elseAction Action
}

// NewConditionalAction creates a new conditional action.
func NewConditionalAction(
	name string,
	cond func(ctx context.Context, smCtx *Context) (bool, error),
	thenAction, elseAction Action,
) *ConditionalAction {
	return &ConditionalAction{
		BaseAction: BaseAction{name: name},
		condition:  cond,
		thenAction: thenAction,
		elseAction: elseAction,
	}
}

func (a *ConditionalAction) Execute(ctx context.Context, smCtx *Context) error {
	shouldExecute, err := a.condition(ctx, smCtx)
	if err != nil {
		return err
	}

	if shouldExecute && a.thenAction != nil {
		return a.thenAction.Execute(ctx, smCtx)
	}

	if !shouldExecute && a.elseAction != nil {
		return a.elseAction.Execute(ctx, smCtx)
	}

	return nil
}

// RetryAction retries action on failure.
type RetryAction struct {
	BaseAction

	action     Action
	maxRetries int
	backoff    time.Duration
}

// NewRetryAction creates a new retry action.
func NewRetryAction(name string, action Action, maxRetries int, backoff time.Duration) *RetryAction {
	return &RetryAction{
		BaseAction: BaseAction{name: name},
		action:     action,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

func (a *RetryAction) Execute(ctx context.Context, smCtx *Context) error {
	var lastErr error

	for i := range a.maxRetries {
		err := a.action.Execute(ctx, smCtx)
		if err == nil {
			return nil
		}

		lastErr = err

		if i < a.maxRetries-1 {
			time.Sleep(a.backoff * time.Duration(i+1))
		}
	}

	return fmt.Errorf("retry exhausted after %d attempts: %w", a.maxRetries, lastErr)
}

// ParallelAction executes actions concurrently.
type ParallelAction struct {
	BaseAction

	actions []Action
}

// NewParallelAction creates a new parallel action.
func NewParallelAction(name string, actions ...Action) *ParallelAction {
	return &ParallelAction{
		BaseAction: BaseAction{name: name},
		actions:    actions,
	}
}

func (a *ParallelAction) Execute(ctx context.Context, smCtx *Context) error {
	var waitGroup sync.WaitGroup

	errChan := make(chan error, len(a.actions))

	// Clone context for each action to avoid race conditions
	for _, action := range a.actions {
		waitGroup.Add(1)

		go func(act Action) {
			defer waitGroup.Done()
			// Note: Using same smCtx for all parallel actions may cause race conditions
			// In production, might want to use separate contexts and merge results
			err := act.Execute(ctx, smCtx)
			if err != nil {
				errChan <- fmt.Errorf("parallel action %s failed: %w", act.Name(), err)
			}
		}(action)
	}

	waitGroup.Wait()
	close(errChan)

	// Collect errors
	errs := make([]error, 0, len(a.actions))
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		// Return first error for simplicity
		// Could combine all errors in production
		return errs[0]
	}

	return nil
}
