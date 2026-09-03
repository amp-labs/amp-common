package feature

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Rollout says who gets a flag. It is pure data so that it can come from code,
// a file, an environment variable, or an HTTP request, and can be logged and
// diffed.
//
// The zero value is "off for everyone". Evaluation order is:
//
//  1. Denied: if the subject matches any denied target, the flag is off.
//  2. Allowed: if the subject matches any allowed target, the flag is on.
//  3. Percentages: if the subject falls inside any percentage, the flag is on.
//  4. Default: otherwise the result is Default.
//
// A subject that lacks the dimension a rule refers to never matches that rule.
//
// The builders use value receivers so that rollouts compose without aliasing;
// only UnmarshalJSON needs a pointer.
type Rollout struct { //nolint:recvcheck // see above
	// Denied lists subjects for which the flag is always off. Deny wins over everything else.
	Denied []Target `json:"denied,omitempty"`

	// Allowed lists subjects for which the flag is always on, unless denied.
	Allowed []Target `json:"allowed,omitempty"`

	// Percentages roll the flag out to a deterministic fraction of a dimension.
	// Multiple entries are OR'd: a subject inside any of them gets the flag.
	Percentages []Percentage `json:"percentages,omitempty"`

	// Default is what subjects matching none of the above get.
	Default bool `json:"default"`
}

// Target matches subjects whose value for Dimension is one of Values.
type Target struct {
	Dimension Dimension `json:"dimension"`
	Values    []string  `json:"values"`
}

// Percentage matches subjects whose bucket falls below Percent. The bucket is
// derived from a hash of the flag name, Salt, Dimension, and the subject's
// value for Dimension, so it is stable across processes and over time.
type Percentage struct {
	Dimension Dimension `json:"dimension"`

	// Percent is in the range 0 to 100 with 0.01 granularity.
	Percent float64 `json:"percent"`

	// Salt changes the bucketing without renaming the flag. Bump it to reshuffle
	// which subjects fall inside the percentage.
	Salt string `json:"salt,omitempty"`
}

// Buckets is the number of buckets a dimension's values are hashed into.
// A percentage of P covers the first P*100 buckets.
const Buckets = 10_000

// percentScale converts a percentage to a number of buckets.
const percentScale = Buckets / 100

var (
	// ErrInvalidRollout is returned (wrapped) when a rollout fails to parse or validate.
	ErrInvalidRollout = errors.New("invalid rollout")

	// ErrUndefined is returned when a flag name has not been defined in the registry.
	ErrUndefined = errors.New("undefined feature flag")
)

// Off returns a rollout that is off for everyone.
func Off() Rollout {
	return Rollout{}
}

// On returns a rollout that is on for everyone.
func On() Rollout {
	return Rollout{Default: true}
}

// Allow returns a copy of the rollout that is always on for subjects whose
// value for dim is one of values.
func (r Rollout) Allow(dim Dimension, values ...string) Rollout {
	out := r.clone()
	out.Allowed = append(out.Allowed, Target{Dimension: dim, Values: slices.Clone(values)})

	return out
}

// Deny returns a copy of the rollout that is always off for subjects whose
// value for dim is one of values.
func (r Rollout) Deny(dim Dimension, values ...string) Rollout {
	out := r.clone()
	out.Denied = append(out.Denied, Target{Dimension: dim, Values: slices.Clone(values)})

	return out
}

// Percent returns a copy of the rollout that is on for pct percent of the
// values of dim.
func (r Rollout) Percent(dim Dimension, pct float64) Rollout {
	return r.PercentWithSalt(dim, pct, "")
}

// PercentWithSalt is Percent with an explicit salt, which changes the
// bucketing without renaming the flag.
func (r Rollout) PercentWithSalt(dim Dimension, pct float64, salt string) Rollout {
	out := r.clone()
	out.Percentages = append(out.Percentages, Percentage{Dimension: dim, Percent: pct, Salt: salt})

	return out
}

// WithDefault returns a copy of the rollout with Default set to dfl.
func (r Rollout) WithDefault(dfl bool) Rollout {
	out := r.clone()
	out.Default = dfl

	return out
}

// Validate reports whether the rollout is well formed: every rule names a
// dimension and every percentage is within 0 to 100. The returned error wraps
// ErrInvalidRollout.
func (r Rollout) Validate() error {
	var problems []string

	for i, target := range r.Denied {
		if target.Dimension == "" {
			problems = append(problems, "denied["+strconv.Itoa(i)+"]: dimension is required")
		}
	}

	for i, target := range r.Allowed {
		if target.Dimension == "" {
			problems = append(problems, "allowed["+strconv.Itoa(i)+"]: dimension is required")
		}
	}

	for i, pct := range r.Percentages {
		if pct.Dimension == "" {
			problems = append(problems, "percentages["+strconv.Itoa(i)+"]: dimension is required")
		}

		if math.IsNaN(pct.Percent) || pct.Percent < 0 || pct.Percent > 100 {
			problems = append(problems, fmt.Sprintf("percentages[%d]: percent %v is not within 0 to 100", i, pct.Percent))
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrInvalidRollout, strings.Join(problems, "; "))
}

// Equal reports whether two rollouts are identical, including rule order.
func (r Rollout) Equal(other Rollout) bool {
	return r.Default == other.Default &&
		slices.EqualFunc(r.Denied, other.Denied, targetEqual) &&
		slices.EqualFunc(r.Allowed, other.Allowed, targetEqual) &&
		slices.Equal(r.Percentages, other.Percentages)
}

// String returns the rollout as compact JSON.
func (r Rollout) String() string {
	out, err := json.Marshal(rolloutJSON(r))
	if err != nil {
		return fmt.Sprintf("%+v", rolloutJSON(r))
	}

	return string(out)
}

// UnmarshalJSON accepts either the object form or a shorthand string such as
// "on", "off", or "25% user" (see ParseRollout).
func (r *Rollout) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var shorthand string

		err := json.Unmarshal(trimmed, &shorthand)
		if err != nil {
			return err
		}

		parsed, err := ParseRollout(shorthand)
		if err != nil {
			return err
		}

		*r = parsed

		return nil
	}

	var plain rolloutJSON

	err := json.Unmarshal(data, &plain)
	if err != nil {
		return err
	}

	*r = Rollout(plain)

	return nil
}

// clone returns a deep copy so that the fluent builders never alias slices.
func (r Rollout) clone() Rollout {
	return Rollout{
		Denied:      cloneTargets(r.Denied),
		Allowed:     cloneTargets(r.Allowed),
		Percentages: slices.Clone(r.Percentages),
		Default:     r.Default,
	}
}

// rolloutJSON is Rollout without methods, used to avoid recursion in UnmarshalJSON.
type rolloutJSON Rollout

// ParseRollout parses the shorthand rollout syntax used by environment
// variables and accepted anywhere a rollout is given as a JSON string:
//
//	on | off                 fully on or off (also true/false, enabled/disabled, 1/0)
//	25% user                 25 percent of the "user" dimension
//	25% of installation      same thing; "of" is optional
//	{"default":true,...}     the JSON object form
//
// Keywords are case-insensitive and surrounding whitespace is ignored. The
// result is parsed, not validated; call Validate to check ranges. Errors wrap
// ErrInvalidRollout.
func ParseRollout(text string) (Rollout, error) {
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "{") {
		var plain rolloutJSON

		err := json.Unmarshal([]byte(text), &plain)
		if err != nil {
			return Rollout{}, fmt.Errorf("%w: %w", ErrInvalidRollout, err)
		}

		return Rollout(plain), nil
	}

	switch strings.ToLower(text) {
	case "on", "true", "enabled", "1":
		return On(), nil
	case "off", "false", "disabled", "0":
		return Off(), nil
	}

	pctText, dimText, found := strings.Cut(text, "%")
	if !found {
		return Rollout{}, fmt.Errorf("%w: cannot parse %q", ErrInvalidRollout, text)
	}

	pct, err := strconv.ParseFloat(strings.TrimSpace(pctText), 64)
	if err != nil {
		return Rollout{}, fmt.Errorf("%w: bad percentage in %q: %w", ErrInvalidRollout, text, err)
	}

	fields := strings.Fields(dimText)
	if len(fields) > 1 && strings.EqualFold(fields[0], "of") {
		fields = fields[1:]
	}

	if len(fields) != 1 {
		return Rollout{}, fmt.Errorf("%w: %q needs exactly one dimension, for example \"25%% user\"",
			ErrInvalidRollout, text)
	}

	return Off().Percent(Dimension(fields[0]), pct), nil
}

func cloneTargets(targets []Target) []Target {
	if targets == nil {
		return nil
	}

	out := make([]Target, len(targets))
	for i, target := range targets {
		out[i] = Target{Dimension: target.Dimension, Values: slices.Clone(target.Values)}
	}

	return out
}

func targetEqual(a, b Target) bool {
	return a.Dimension == b.Dimension && slices.Equal(a.Values, b.Values)
}
