package feature

import (
	"math"

	"github.com/zeebo/xxh3"
)

// Reason explains why an evaluation produced its result.
type Reason string

const (
	// ReasonDefault means no rule matched and the rollout's Default was used.
	ReasonDefault Reason = "default"

	// ReasonAllowed means the subject matched an Allowed target.
	ReasonAllowed Reason = "allowed"

	// ReasonDenied means the subject matched a Denied target.
	ReasonDenied Reason = "denied"

	// ReasonPercentage means the subject fell inside a Percentage.
	ReasonPercentage Reason = "percentage"

	// ReasonUndefined means the flag has not been defined in the registry.
	ReasonUndefined Reason = "undefined"
)

// Result is the outcome of evaluating a flag for a subject, with enough detail
// to answer "why did this subject (not) get the flag".
type Result struct {
	Flag    Name   `json:"flag"`
	Enabled bool   `json:"enabled"`
	Reason  Reason `json:"reason"`

	// Dimension is the dimension of the rule that decided the result, if any.
	Dimension Dimension `json:"dimension,omitempty"`

	// Bucket is the subject's bucket, from 0 to Buckets-1, for the percentage
	// rule that decided the result. When no percentage rule matched it is the
	// bucket for the last percentage rule that applied to the subject, so that
	// "why am I not in the 25%?" can be answered. It is -1 when no percentage
	// rule applied.
	Bucket int `json:"bucket"`
}

// Hasher maps a string to a 64-bit hash. The default is xxh3. Replace it with
// WithHasher only if you need to match bucketing done elsewhere.
type Hasher func(string) uint64

// Bucket returns the bucket, from 0 to Buckets-1, that a value falls into for
// the given flag, salt and dimension. It is exported so that the bucketing can
// be reproduced elsewhere, for example in SQL or another language. A nil
// hasher uses the default.
func Bucket(hasher Hasher, name Name, salt string, dim Dimension, value string) int {
	if hasher == nil {
		hasher = defaultHasher
	}

	return int(hasher(hashPrefix(name, salt, dim)+value) % Buckets) //nolint:gosec // bounded by Buckets
}

// resolver returns a subject's value for a dimension, and whether it has one.
type resolver func(Dimension) (string, bool)

// compiled is a Rollout pre-processed for fast evaluation: target values are
// turned into sets and the hash key prefix for each percentage is built once.
type compiled struct {
	rollout     Rollout
	denied      []compiledTarget
	allowed     []compiledTarget
	percentages []compiledPercentage
	overridden  bool
}

type compiledTarget struct {
	dimension Dimension
	values    map[string]struct{}
}

type compiledPercentage struct {
	dimension Dimension
	threshold uint64
	prefix    string
}

func compile(name Name, rollout Rollout, overridden bool) *compiled {
	out := &compiled{
		rollout:     rollout.clone(),
		denied:      compileTargets(rollout.Denied),
		allowed:     compileTargets(rollout.Allowed),
		percentages: make([]compiledPercentage, len(rollout.Percentages)),
		overridden:  overridden,
	}

	for i, pct := range rollout.Percentages {
		out.percentages[i] = compiledPercentage{
			dimension: pct.Dimension,
			threshold: threshold(pct.Percent),
			prefix:    hashPrefix(name, pct.Salt, pct.Dimension),
		}
	}

	return out
}

func compileTargets(targets []Target) []compiledTarget {
	out := make([]compiledTarget, len(targets))

	for i, target := range targets {
		values := make(map[string]struct{}, len(target.Values))
		for _, value := range target.Values {
			values[value] = struct{}{}
		}

		out[i] = compiledTarget{dimension: target.Dimension, values: values}
	}

	return out
}

// evaluate applies the rollout to a subject. See Rollout for the ordering.
func (c *compiled) evaluate(name Name, hasher Hasher, resolve resolver) Result {
	result := Result{Flag: name, Bucket: -1}

	for _, target := range c.denied {
		if target.matches(resolve) {
			result.Reason = ReasonDenied
			result.Dimension = target.dimension

			return result
		}
	}

	for _, target := range c.allowed {
		if target.matches(resolve) {
			result.Enabled = true
			result.Reason = ReasonAllowed
			result.Dimension = target.dimension

			return result
		}
	}

	for _, pct := range c.percentages {
		value, ok := resolve(pct.dimension)
		if !ok {
			continue
		}

		bucket := hasher(pct.prefix+value) % Buckets
		result.Bucket = int(bucket) //nolint:gosec // bounded by Buckets

		if bucket < pct.threshold {
			result.Enabled = true
			result.Reason = ReasonPercentage
			result.Dimension = pct.dimension

			return result
		}
	}

	result.Enabled = c.rollout.Default
	result.Reason = ReasonDefault

	return result
}

func (t compiledTarget) matches(resolve resolver) bool {
	value, ok := resolve(t.dimension)
	if !ok {
		return false
	}

	_, found := t.values[value]

	return found
}

func defaultHasher(text string) uint64 {
	return xxh3.HashString(text)
}

// threshold converts a percentage to the number of buckets it covers.
func threshold(pct float64) uint64 {
	return uint64(math.Round(pct * percentScale))
}

// hashPrefix builds the stable part of the hash input. NUL separators keep
// "a"+"bc" and "ab"+"c" from colliding.
func hashPrefix(name Name, salt string, dim Dimension) string {
	return string(name) + "\x00" + salt + "\x00" + string(dim) + "\x00"
}
