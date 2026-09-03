package feature

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/amp-labs/amp-common/region"
	"github.com/amp-labs/amp-common/stage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinePanics(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("dup", Off())

	assert.Panics(t, func() { reg.Define("dup", Off()) })
	assert.Panics(t, func() { reg.Define("", Off()) })
	assert.Panics(t, func() { reg.Define("bad", Off().Percent(dimUser, 200)) })
}

func TestFlagAccessors(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("accessors", Off().Percent(dimUser, 5), Description("desc"), Owner("team"))

	assert.Equal(t, Name("accessors"), flag.Name())
	assert.Equal(t, "desc", flag.Description())
	assert.Equal(t, "team", flag.Owner())
	assert.True(t, flag.Default().Equal(Off().Percent(dimUser, 5)))
	assert.True(t, flag.Current().Equal(Off().Percent(dimUser, 5)))
	assert.False(t, flag.Overridden())

	info := flag.Info()
	assert.Equal(t, Name("accessors"), info.Name)
	assert.Equal(t, "desc", info.Description)
	assert.Equal(t, "team", info.Owner)
	assert.False(t, info.Overridden)

	current := flag.Current()
	current.Percentages[0].Percent = 99

	assert.InDelta(t, 5, flag.Current().Percentages[0].Percent, 0, "Current returns a copy")
}

func TestSetAndReset(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("set-reset", Off())
	ctx := t.Context()

	assert.False(t, flag.Enabled(ctx))

	require.NoError(t, flag.Set(On()))
	assert.True(t, flag.Enabled(ctx))
	assert.True(t, flag.Overridden())

	flag.Reset()
	assert.False(t, flag.Enabled(ctx))
	assert.False(t, flag.Overridden())

	require.ErrorIs(t, flag.Set(Off().Percent(dimUser, -5)), ErrInvalidRollout)
	assert.False(t, flag.Enabled(ctx), "invalid rollouts are not applied")

	require.NoError(t, reg.Set("set-reset", On()))
	assert.True(t, flag.Enabled(ctx))

	require.NoError(t, reg.Reset("set-reset"))
	assert.False(t, flag.Enabled(ctx))

	require.ErrorIs(t, reg.Set("nope", On()), ErrUndefined)
	require.ErrorIs(t, reg.Reset("nope"), ErrUndefined)

	require.NoError(t, flag.Set(Off()))
	assert.False(t, flag.Overridden(), "setting the default is not an override")
}

func TestLoad(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	first := reg.Define("a", Off())
	second := reg.Define("b", Off())
	third := reg.Define("c", On())
	ctx := t.Context()

	require.NoError(t, second.Set(On()))

	err := reg.Load(map[Name]Rollout{
		"a":     On(),
		"c":     Off(),
		"later": On(),
		"bad":   Off().Percent(dimUser, 500),
	})
	require.ErrorIs(t, err, ErrInvalidRollout)

	assert.True(t, first.Enabled(ctx))
	assert.False(t, second.Enabled(ctx), "manual override reverted because it was absent from the load")
	assert.False(t, third.Enabled(ctx))

	later := reg.Define("later", Off())
	assert.True(t, later.Enabled(ctx), "pending override applies when the flag is defined")
	assert.True(t, later.Overridden())

	require.NoError(t, reg.Load(nil))
	assert.False(t, first.Enabled(ctx))
	assert.False(t, later.Enabled(ctx))
	assert.True(t, third.Enabled(ctx))
}

func TestLoadInvalidLeavesFlagUntouched(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("keep", Off())
	ctx := t.Context()

	require.NoError(t, flag.Set(On()))
	require.ErrorIs(t, reg.Load(map[Name]Rollout{"keep": Off().Percent("", 10)}), ErrInvalidRollout)
	assert.True(t, flag.Enabled(ctx))
}

func TestSnapshotAndListing(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("b", Off())
	reg.Define("a", On())

	flags := reg.Flags()
	require.Len(t, flags, 2)
	assert.Equal(t, Name("a"), flags[0].Name())
	assert.Equal(t, Name("b"), flags[1].Name())

	infos := reg.Infos()
	require.Len(t, infos, 2)
	assert.Equal(t, Name("a"), infos[0].Name)

	snapshot := reg.Snapshot()
	assert.Len(t, snapshot, 2)
	assert.True(t, snapshot["a"].Equal(On()))
	assert.True(t, snapshot["b"].Equal(Off()))

	_, ok := reg.Lookup("a")
	assert.True(t, ok)

	_, ok = reg.Lookup("zzz")
	assert.False(t, ok)
}

func TestSubjectPrecedence(t *testing.T) {
	t.Parallel()

	reg := NewRegistry(WithGlobalAttribute(dimUser, "global"))
	flag := reg.Define("precedence", Off().Allow(dimUser, "winner"))
	ctx := t.Context()

	assert.False(t, flag.Enabled(ctx))

	reg.SetGlobalAttribute(dimUser, "winner")
	assert.True(t, flag.Enabled(ctx), "global attribute is used when nothing else is set")
	assert.Equal(t, Subject{dimUser: "winner"}, reg.GlobalAttributes())

	inContext := WithAttribute(ctx, dimUser, "ctx")
	assert.False(t, flag.Enabled(inContext), "context beats global")
	assert.True(t, flag.EnabledFor(inContext, Subject{dimUser: "winner"}), "explicit beats context")

	reg.SetGlobalAttribute(dimUser, "")
	assert.False(t, flag.Enabled(ctx))
	assert.Empty(t, reg.GlobalAttributes())
}

func TestHostDefaultsToHostname(t *testing.T) {
	t.Parallel()

	host, err := os.Hostname()
	require.NoError(t, err)

	if host == "" {
		t.Skip("no hostname available")
	}

	reg := NewRegistry()
	flag := reg.Define("host", Off().Allow(Host, host))
	ctx := t.Context()

	assert.True(t, flag.Enabled(ctx))

	reg.SetGlobalAttribute(Host, "other-pod")
	assert.False(t, flag.Enabled(ctx))
}

func TestStageDimension(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("staged", Off().Allow(Stage, string(stage.Dev), string(stage.Staging)))
	ctx := t.Context()

	assert.True(t, flag.Enabled(stage.WithStage(ctx, stage.Dev)))
	assert.True(t, flag.Enabled(stage.WithStage(ctx, stage.Staging)))
	assert.False(t, flag.Enabled(stage.WithStage(ctx, stage.Prod)))
	assert.True(t, flag.EnabledFor(stage.WithStage(ctx, stage.Prod), Subject{Stage: "dev"}), "explicit beats automatic")
}

func TestRegionDimension(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("regional", Off().Allow(Region, string(region.Eu)))
	ctx := t.Context()

	assert.True(t, flag.Enabled(region.WithRegion(ctx, region.Eu)))
	assert.False(t, flag.Enabled(region.WithRegion(ctx, region.Us)))
	assert.False(t, flag.Enabled(region.WithRegion(ctx, region.Unknown)))
	assert.True(t, flag.EnabledFor(region.WithRegion(ctx, region.Us), Subject{Region: "eu"}), "explicit beats automatic")

	unknown := reg.Define("unknown-region", Off().Allow(Region, string(region.Unknown)))
	assert.True(t, unknown.Enabled(region.WithRegion(ctx, region.Unknown)), "unknown is a value, not an absence")
}

func TestEnabledByName(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.Define("named", On())

	ctx := t.Context()

	assert.True(t, reg.Enabled(ctx, "named"))
	assert.False(t, reg.Enabled(ctx, "missing"))

	result := reg.Evaluate(ctx, "missing", nil)
	assert.Equal(t, ReasonUndefined, result.Reason)
	assert.Equal(t, Name("missing"), result.Flag)
	assert.Equal(t, -1, result.Bucket)
}

func TestNilContext(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("nil-context", On())

	assert.True(t, flag.Enabled(nil)) //nolint:staticcheck // nil contexts are explicitly supported
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	const name Name = "feature-test-default-registry"

	flag := Define(name, Off())
	ctx := t.Context()

	found, ok := Lookup(name)
	require.True(t, ok)
	assert.Same(t, flag, found)
	assert.Same(t, Default(), flag.registry)

	assert.False(t, Enabled(ctx, name))

	require.NoError(t, Set(name, On()))
	assert.True(t, Enabled(ctx, name))

	require.NoError(t, Reset(name))
	assert.False(t, Enabled(ctx, name))

	require.NoError(t, Load(map[Name]Rollout{name: On()}))
	assert.True(t, Enabled(ctx, name))

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.NoError(t, Watch(canceled, Static(nil)))
	assert.False(t, Enabled(ctx, name))
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	flag := reg.Define("concurrent", Off().Percent(dimUser, 50))
	ctx := t.Context()

	var workers sync.WaitGroup

	for range 8 {
		workers.Go(func() {
			for i := range 2000 {
				flag.EnabledFor(ctx, Subject{dimUser: strconv.Itoa(i)})
				reg.Enabled(ctx, "concurrent")
			}
		})
	}

	workers.Go(func() {
		for i := range 200 {
			_ = flag.Set(Off().Percent(dimUser, float64(i%100)))
			reg.SetGlobalAttribute(Host, strconv.Itoa(i))
			_ = reg.Load(map[Name]Rollout{"concurrent": On()})
			_ = reg.Snapshot()
		}
	})

	workers.Wait()
}
