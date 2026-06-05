package retry

import (
	"context"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/optional"
)

func readOptionalDuration(
	ctx context.Context,
	key string,
) optional.Value[time.Duration] {
	d := envutil.Duration(ctx, key)

	if !d.HasValue() {
		return optional.None[time.Duration]()
	}

	val, err := d.Value()
	if err != nil {
		logger.Warn(ctx, "failed to parse time.Duration value for env var %q: %w", key, err)

		return optional.None[time.Duration]()
	}

	return optional.Some[time.Duration](val)
}

func readOptionalFloat(
	ctx context.Context,
	key string,
) optional.Value[float64] {
	f := envutil.Float64(ctx, key)

	if !f.HasValue() {
		return optional.None[float64]()
	}

	val, err := f.Value()
	if err != nil {
		logger.Warn(ctx, "failed to parse float64 value for env var %q: %w", key, err)

		return optional.None[float64]()
	}

	return optional.Some[float64](val)
}

func readOptionalUint(
	ctx context.Context,
	key string,
) optional.Value[uint] {
	u := envutil.Uint[uint](ctx, key)

	if !u.HasValue() {
		return optional.None[uint]()
	}

	val, err := u.Value()
	if err != nil {
		logger.Warn(ctx, "failed to parse uint value for env var %q: %w", key, err)

		return optional.None[uint]()
	}

	return optional.Some[uint](val)
}

func readBudget(ctx context.Context, key string) optional.Value[*Budget] {
	rate := readOptionalFloat(ctx, key+"_RATE")
	ratio := readOptionalFloat(ctx, key+"_RATIO")

	rateVal, ok := rate.Get()
	if !ok {
		return optional.None[*Budget]()
	}

	ratioVal, ok := ratio.Get()
	if !ok {
		return optional.None[*Budget]()
	}

	budget := &Budget{
		Rate:  rateVal,
		Ratio: ratioVal,
	}

	return optional.Some[*Budget](budget)
}

func readBackoff(ctx context.Context, key string) optional.Value[Backoff] {
	baseEnv := readOptionalDuration(ctx, key+"_BASE")
	maxEnv := readOptionalDuration(ctx, key+"_MAX")
	factorEnv := readOptionalFloat(ctx, key+"_FACTOR")

	base, baseOk := baseEnv.Get()
	maxVal, maxOk := maxEnv.Get()
	factor, factorOk := factorEnv.Get()

	if !baseOk || !maxOk || !factorOk {
		return optional.None[Backoff]()
	}

	backoff := ExpBackoff{
		Base:   base,
		Max:    maxVal,
		Factor: factor,
	}

	return optional.Some[Backoff](backoff)
}

func OptionsFromEnv(ctx context.Context, baseKey string) []Option {
	var opts []Option

	budget, ok := readBudget(ctx, baseKey+"_BUDGET").Get()
	if ok {
		opts = append(opts, WithBudget(budget))
	}

	timeout, ok := readOptionalDuration(ctx, baseKey+"_TIMEOUT").Get()
	if ok {
		opts = append(opts, WithTimeout(Timeout(timeout)))
	}

	attempts, ok := readOptionalUint(ctx, baseKey+"_ATTEMPTS").Get()
	if ok {
		opts = append(opts, WithAttempts(Attempts(attempts)))
	}

	jitter, ok := readOptionalFloat(ctx, baseKey+"_JITTER").Get()
	if ok {
		opts = append(opts, WithJitter(Jitter(jitter)))
	}

	backoff, ok := readBackoff(ctx, baseKey+"_BACKOFF").Get()
	if ok {
		opts = append(opts, WithBackoff(backoff))
	}

	return opts
}
