package feature

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	waitFor = 2 * time.Second
	tick    = 5 * time.Millisecond
)

var errBoom = errors.New("boom")

// watch runs reg.Watch in the background and returns a function that stops it
// and returns its error.
func watch(t *testing.T, reg *Registry, source Source) func() error {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		done <- reg.Watch(ctx, source)
	}()

	return func() error {
		cancel()

		return <-done
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}

func TestStaticSource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("static", Off())
	ctx := t.Context()

	stop := watch(t, reg, Static(map[Name]Rollout{"static": On()}))

	require.Eventually(t, func() bool { return flag.Enabled(ctx) }, waitFor, tick)
	require.NoError(t, stop())
}

func TestStaticSourceInvalid(t *testing.T) {
	t.Parallel()

	err := NewRegistry().Watch(t.Context(), Static(map[Name]Rollout{"x": Off().Percent(dimUser, 200)}))
	require.ErrorIs(t, err, ErrInvalidRollout)
}

func TestFileSource(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "flags.json")
	reg := NewRegistry()
	first := reg.Define("a", Off())
	second := reg.Define("b", Off())
	third := reg.Define("c", On())
	ctx := t.Context()

	stop := watch(t, reg, FileSource(path, PollInterval(tick)))

	writeFile(t, path, `{
		"a": "on",
		"b": {"percentages": [{"dimension": "user", "percent": 100}]},
		"c": {"percentages": [{"dimension": "user", "percent": 500}]}
	}`)
	require.Eventually(t, func() bool {
		return first.Enabled(ctx) && second.EnabledFor(ctx, Subject{dimUser: "u"})
	}, waitFor, tick)
	assert.True(t, third.Enabled(ctx), "invalid entry leaves the flag untouched")

	writeFile(t, path, `{not json`)
	assert.Never(t, func() bool { return !first.Enabled(ctx) }, 10*tick, tick)

	require.NoError(t, os.Remove(path))
	require.Eventually(t, func() bool { return !first.Enabled(ctx) && !second.EnabledFor(ctx, Subject{dimUser: "u"}) },
		waitFor, tick)

	writeFile(t, path, `{"a": "on"}`)
	require.Eventually(t, func() bool { return first.Enabled(ctx) }, waitFor, tick)

	require.NoError(t, stop())
}

func TestFileSourceDefaultInterval(t *testing.T) {
	t.Parallel()

	src, ok := FileSource("x").(*fileSource)
	require.True(t, ok)
	assert.Equal(t, DefaultPollInterval, src.interval)

	src, ok = FileSource("x", PollInterval(0)).(*fileSource)
	require.True(t, ok)
	assert.Equal(t, DefaultPollInterval, src.interval, "non-positive intervals are ignored")
}

func TestParseEnviron(t *testing.T) {
	t.Parallel()

	got, err := ParseEnviron("", []string{
		"FEATURE_NEW_BILLING_ENGINE=on",
		"FEATURE_FAST_SYNC=25% installation",
		`FEATURE_JSON={"default":true}`,
		"FEATURE_EMPTY=",
		"FEATURE_=on",
		"OTHER=on",
		"FEATURE_BAD=maybe",
		"NOEQUALS",
	})
	require.ErrorIs(t, err, ErrInvalidRollout)
	assert.Len(t, got, 3)
	assert.True(t, got["new-billing-engine"].Equal(On()))
	assert.True(t, got["fast-sync"].Equal(Off().Percent(dimInstallation, 25)))
	assert.True(t, got["json"].Equal(On()))

	got, err = ParseEnviron("FF_", []string{"FF_X=off"})
	require.NoError(t, err)
	assert.True(t, got["x"].Equal(Off()))

	assert.Equal(t, "FEATURE_NEW_BILLING_ENGINE", EnvKey("", "new-billing-engine"))
	assert.Equal(t, "FF_X_Y", EnvKey("FF_", "x-y"))
}

func TestEnvSource(t *testing.T) { //nolint:paralleltest // t.Setenv is incompatible with t.Parallel
	t.Setenv("FEATURE_ENV_SOURCE_TEST", "on")
	t.Setenv("FEATURE_ENV_SOURCE_BROKEN", "maybe")

	reg := NewRegistry()
	flag := reg.Define("env-source-test", Off())
	ctx := t.Context()

	stop := watch(t, reg, EnvSource(""))

	require.Eventually(t, func() bool { return flag.Enabled(ctx) }, waitFor, tick)
	require.NoError(t, stop())
}

func TestMultiSource(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	first := reg.Define("a", Off())
	second := reg.Define("b", Off())
	third := reg.Define("c", Off())
	ctx := t.Context()

	updates := make(chan map[Name]Rollout)
	dynamic := SourceFunc(func(ctx context.Context, apply ApplyFunc) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case set := <-updates:
				_ = apply(set)
			}
		}
	})

	stop := watch(t, reg, Multi(Static(map[Name]Rollout{"a": On(), "b": On()}), dynamic))

	state := func() [3]bool {
		return [3]bool{first.Enabled(ctx), second.Enabled(ctx), third.Enabled(ctx)}
	}

	require.Eventually(t, func() bool { return state() == [3]bool{true, true, false} }, waitFor, tick)

	updates <- map[Name]Rollout{"b": Off(), "c": On()}

	require.Eventually(t, func() bool { return state() == [3]bool{true, false, true} }, waitFor, tick,
		"the later source wins on conflicts")

	updates <- map[Name]Rollout{}

	require.Eventually(t, func() bool { return state() == [3]bool{true, true, false} }, waitFor, tick,
		"clearing the later source restores the earlier one")

	require.NoError(t, stop())
}

func TestMultiSourceFailureStopsOthers(t *testing.T) {
	t.Parallel()

	failing := SourceFunc(func(context.Context, ApplyFunc) error {
		return errBoom
	})
	blocking := SourceFunc(func(ctx context.Context, _ ApplyFunc) error {
		<-ctx.Done()

		return nil
	})

	err := NewRegistry().Watch(t.Context(), Multi(blocking, failing))
	require.ErrorIs(t, err, errBoom)
}

func TestMultiSourceEmpty(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, NewRegistry().Watch(ctx, Multi()))
}
