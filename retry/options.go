package retry

// Option is a function that configures a Runner or ValueRunner.
// Options follow the functional options pattern for flexible configuration.
type Option func(*options)

// options holds the internal configuration for retry behavior.
type options struct {
	attempts Attempts // Maximum number of retry attempts
	backoff  Backoff  // Backoff strategy for calculating delays
	budget   *Budget  // Retry budget to prevent cascading failures
	jitter   Jitter   // Jitter strategy for randomizing delays
	timeout  Timeout  // Timeout for each individual attempt
}

// WithBudget configures a retry budget to prevent cascading failures.
// The budget limits retries when the system is under heavy load.
//
// Example:
//
//	budget := &retry.Budget{
//	    Rate:  10.0,  // Enforce budget when > 10 req/sec
//	    Ratio: 0.1,   // Allow up to 10% retries
//	}
//	runner := retry.NewRunner(retry.WithBudget(budget))
func WithBudget(budget *Budget) Option {
	return func(o *options) {
		o.budget = budget
	}
}

// WithBackoff configures the backoff strategy for calculating retry delays.
//
// Example:
//
//	backoff := retry.ExpBackoff{
//	    Base:   100 * time.Millisecond,
//	    Max:    10 * time.Second,
//	    Factor: 2.0,
//	}
//	runner := retry.NewRunner(retry.WithBackoff(backoff))
func WithBackoff(b Backoff) Option {
	return func(o *options) {
		o.backoff = b
	}
}

// WithAttempts configures the maximum number of retry attempts.
// A value of 0 means unlimited retries (use with caution).
//
// Example:
//
//	runner := retry.NewRunner(retry.WithAttempts(5))
func WithAttempts(a Attempts) Option {
	return func(o *options) {
		o.attempts = a
	}
}

// WithTimeout configures a timeout for each individual retry attempt.
// If an attempt exceeds this duration, it will be canceled and retried.
//
// Example:
//
//	runner := retry.NewRunner(retry.WithTimeout(30 * time.Second))
func WithTimeout(t Timeout) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// WithJitter configures the jitter strategy for randomizing retry delays.
// Jitter helps prevent thundering herd problems.
//
// Example:
//
//	runner := retry.NewRunner(retry.WithJitter(retry.FullJitter))
func WithJitter(j Jitter) Option {
	return func(o *options) {
		o.jitter = j
	}
}
