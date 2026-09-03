package feature

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// evaluationsTotal counts flag evaluations.
	//
	// Labels:
	//   - flag: the flag name. Flags are defined in code, so the label set is bounded.
	//   - enabled: "true" or "false", the outcome.
	//   - reason: which part of the rollout decided the outcome (see Reason).
	//
	// Useful queries:
	//   - sum(rate(feature_flag_evaluations_total[5m])) by (flag, enabled) - rollout progress per flag
	//   - feature_flag_evaluations_total{reason="denied"} - how often the denylist fires
	evaluationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_flag_evaluations_total",
		Help: "The total number of feature flag evaluations",
	}, []string{"flag", "enabled", "reason"})

	// undefinedTotal counts lookups of flags that were never defined. It has no
	// flag label on purpose: undefined names may come from configuration and
	// would be unbounded. The name is logged once instead.
	undefinedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "feature_flag_undefined_lookups_total",
		Help: "The total number of lookups of feature flags that were never defined",
	})

	// changesTotal counts rollout changes.
	//
	// Labels:
	//   - flag: the flag name.
	//   - source: "set", "reset" or "load", how the change was made.
	changesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_flag_changes_total",
		Help: "The total number of feature flag rollout changes",
	}, []string{"flag", "source"})
)

func recordEvaluation(result Result) {
	evaluationsTotal.WithLabelValues(
		string(result.Flag),
		strconv.FormatBool(result.Enabled),
		string(result.Reason),
	).Inc()
}

func recordChange(name Name, source string) {
	changesTotal.WithLabelValues(string(name), source).Inc()
}
