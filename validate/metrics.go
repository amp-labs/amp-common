package validate

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// validationsTotal is a Prometheus counter that tracks the total number of validation attempts.
	//
	// This metric helps monitor validation activity across the application and identify types that
	// may not implement the Validator interface when they should (or vice versa). It also tracks
	// the success/failure rate of validations.
	//
	// Labels:
	//   - can_validate_type: "true" if the type implements either HasValidate or HasValidateWithContext
	//     interface and validation was executed, "false" if the type does not implement either interface
	//     and validation was skipped.
	//   - has_error: "true" if the validation returned an error, "false" if validation succeeded or
	//     the type doesn't implement a validation interface. This allows tracking validation failure rates.
	//
	// The counter increments each time Validate() is called, regardless of outcome. This allows tracking:
	//   - Total validation volume across the application
	//   - Percentage of types that implement validation vs those that don't
	//   - Validation success/failure rates by type capability
	//   - Validation hotspots and usage patterns
	//
	// Usage example in dashboards:
	//   - rate(validation_calls_total[5m]) - Validations per second
	//   - validation_calls_total{can_validate_type="false"} - Count of non-validatable types
	//   - validation_calls_total{has_error="true"} - Count of failed validations
	//   - sum(rate(validation_calls_total[5m])) by (can_validate_type) - Breakdown by type capability
	//   - sum(rate(validation_calls_total{has_error="true"}[5m])) / sum(rate(validation_calls_total[5m])) - Error rate
	//
	// The nolint:gochecknoglobals directive is used because Prometheus metrics are intentionally
	// global by design - they need to be registered once and accessed throughout the application
	// lifecycle for consistent metric collection.
	validationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{ //nolint:gochecknoglobals
		Name: "validation_calls_total",
		Help: "The total number of calls to Validate",
	}, []string{"can_validate_type", "has_error"})

	// validationTime is a Prometheus histogram that tracks the duration of validation operations in milliseconds.
	//
	// This metric provides detailed performance insights into validation execution times, enabling
	// identification of slow validation logic, performance regressions, and optimization opportunities.
	// It records timing only for types that implement validation interfaces (HasValidate or HasValidateWithContext).
	//
	// Labels:
	//   - type: The Go type name being validated (e.g., "CreateUserRequest", "UpdateOrderRequest").
	//     This allows tracking performance characteristics of different validation implementations.
	//   - has_error: "true" if the validation returned an error, "false" if validation succeeded.
	//     This enables comparing performance between successful validations and those that fail,
	//     which is useful for identifying if error paths take significantly different amounts of time.
	//
	// Buckets: The histogram uses carefully chosen buckets covering a wide range of validation durations:
	//   - Sub-10ms: 1, 5, 10ms - Fast validations (simple field checks)
	//   - 10-100ms: 25, 50, 100ms - Medium validations (multiple checks, basic logic)
	//   - 100ms-1s: 250, 500, 1000ms - Slow validations (database lookups, external calls)
	//   - 1s+: 2500, 5000, 10000ms - Very slow validations (may indicate performance issues)
	//
	// The wide bucket range accommodates both lightweight validations (field presence checks)
	// and complex validations (uniqueness checks requiring database queries).
	//
	// Usage example in dashboards:
	//   - histogram_quantile(0.95, rate(validation_time_millis_bucket[5m])) - 95th percentile latency
	//   - histogram_quantile(0.50, rate(validation_time_millis_bucket[5m])) by (type) - Median by type
	//   - histogram_quantile(0.95, rate(validation_time_millis_bucket{has_error="true"}[5m])) - p95 for failed validations
	//   - rate(validation_time_millis_sum[5m]) / rate(validation_time_millis_count[5m]) - Average duration
	//   - validation_time_millis_bucket{le="100"} - Count of validations completing under 100ms
	//
	// Alerting examples:
	//   - Alert if p95 validation time exceeds 500ms for more than 5 minutes
	//   - Alert if any type consistently takes >1s to validate
	//   - Alert if error path validations are significantly slower than success path
	//
	// Note: Histograms are more expensive than counters. The metric records every validation duration,
	// so extremely high-frequency validations (>10k/sec) should be monitored for memory impact.
	validationTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "validation_time_millis",
		Help: "The time it takes to validate, in milliseconds",
		Buckets: []float64{
			1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000,
		},
	}, []string{"type", "has_error"})
)

// init pre-initializes the validationsTotal metric with zero values for all known label combinations.
//
// This initialization is important for several reasons:
//
// 1. Prevents missing data gaps in time series:
//   - Prometheus queries on metrics that don't exist yet return no data
//   - Pre-initialization ensures the metric exists from application start
//   - Dashboards and alerts work correctly even if no validations have occurred yet
//
// 2. Enables accurate rate calculations:
//   - rate() and increase() functions need consistent time series
//   - Without initialization, the first data point appears "late" causing rate spikes
//
// 3. Improves query performance:
//   - Prometheus can optimize queries when all label combinations are known upfront
//
// 4. Makes monitoring more reliable:
//   - Alerting rules can detect when validation rates drop to zero
//   - Without initialization, zero vs non-existent metrics are indistinguishable
//
// The function initializes all four combinations of the two label dimensions:
//   - can_validate_type × has_error: (true, true), (false, true), (true, false), (false, false)
//
// This establishes the complete set of possible validation outcomes:
//   - (true, true): Type implements validation and returned an error
//   - (true, false): Type implements validation and succeeded
//   - (false, true): Should never occur in practice (non-validatable types don't return errors)
//   - (false, false): Type doesn't implement validation interface
func init() {
	validationsTotal.WithLabelValues("true", "true").Add(0)
	validationsTotal.WithLabelValues("false", "true").Add(0)
	validationsTotal.WithLabelValues("true", "false").Add(0)
	validationsTotal.WithLabelValues("false", "false").Add(0)
}
