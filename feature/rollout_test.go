package feature

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRolloutBuilders(t *testing.T) {
	t.Parallel()

	base := Off()
	built := base.
		Allow(dimInstallation, "a", "b").
		Deny(dimUser, "bad").
		Percent(dimUser, 25).
		PercentWithSalt(Host, 10, "v2").
		WithDefault(true)

	assert.Equal(t, Rollout{}, base, "builders do not modify the receiver")
	assert.Equal(t, Rollout{
		Denied:  []Target{{Dimension: dimUser, Values: []string{"bad"}}},
		Allowed: []Target{{Dimension: dimInstallation, Values: []string{"a", "b"}}},
		Percentages: []Percentage{
			{Dimension: dimUser, Percent: 25},
			{Dimension: Host, Percent: 10, Salt: "v2"},
		},
		Default: true,
	}, built)
	assert.True(t, On().Default)
	assert.False(t, Off().Default)
}

func TestRolloutBuildersDoNotAlias(t *testing.T) {
	t.Parallel()

	values := []string{"a"}
	first := Off().Allow(dimUser, values...)
	second := first.Allow(dimInstallation, "x")

	values[0] = "changed"
	second.Allowed[0].Values[0] = "mutated"

	assert.Equal(t, "a", first.Allowed[0].Values[0])
}

func TestRolloutValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rollout Rollout
		wantErr bool
	}{
		{name: "zero value", rollout: Rollout{}},
		{name: "on", rollout: On()},
		{name: "percent", rollout: Off().Percent(dimUser, 25)},
		{name: "percent edges", rollout: Off().Percent(dimUser, 0).Percent(dimUser, 100)},
		{name: "allow without values", rollout: Off().Allow(dimUser)},
		{name: "negative percent", rollout: Off().Percent(dimUser, -1), wantErr: true},
		{name: "percent over 100", rollout: Off().Percent(dimUser, 100.01), wantErr: true},
		{name: "nan percent", rollout: Off().Percent(dimUser, math.NaN()), wantErr: true},
		{name: "percent without dimension", rollout: Off().Percent("", 10), wantErr: true},
		{name: "allow without dimension", rollout: Off().Allow("", "a"), wantErr: true},
		{name: "deny without dimension", rollout: Off().Deny("", "a"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.rollout.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidRollout)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRolloutEqual(t *testing.T) {
	t.Parallel()

	rollout := Off().Allow(dimUser, "a", "b").Percent(dimUser, 25)

	assert.True(t, rollout.Equal(Off().Allow(dimUser, "a", "b").Percent(dimUser, 25)))
	assert.False(t, rollout.Equal(Off().Allow(dimUser, "b", "a").Percent(dimUser, 25)), "order matters")
	assert.False(t, rollout.Equal(rollout.WithDefault(true)))
	assert.False(t, rollout.Equal(Off().Allow(dimUser, "a", "b").PercentWithSalt(dimUser, 25, "x")))
	assert.True(t, Rollout{}.Equal(Off()))
	assert.True(t, Rollout{Allowed: []Target{}}.Equal(Off()), "nil and empty are the same")
}

func TestRolloutJSONRoundTrip(t *testing.T) {
	t.Parallel()

	original := Off().Deny(dimUser, "bad").Allow(dimInstallation, "a").PercentWithSalt(dimUser, 12.5, "s")

	data, err := json.Marshal(original)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"denied":[{"dimension":"user","values":["bad"]}],
		"allowed":[{"dimension":"installation","values":["a"]}],
		"percentages":[{"dimension":"user","percent":12.5,"salt":"s"}],
		"default":false
	}`, string(data))

	var decoded Rollout

	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.True(t, original.Equal(decoded))
	assert.Equal(t, string(data), original.String())
}

func TestRolloutUnmarshalShorthand(t *testing.T) {
	t.Parallel()

	var flags map[Name]Rollout

	require.NoError(t, json.Unmarshal([]byte(`{
		"a": "on",
		"b": "off",
		"c": "25% user",
		"d": {"default": true}
	}`), &flags))

	assert.True(t, flags["a"].Equal(On()))
	assert.True(t, flags["b"].Equal(Off()))
	assert.True(t, flags["c"].Equal(Off().Percent(dimUser, 25)))
	assert.True(t, flags["d"].Equal(On()))

	var bad Rollout

	require.ErrorIs(t, json.Unmarshal([]byte(`"nonsense"`), &bad), ErrInvalidRollout)
}

func TestParseRollout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    Rollout
		wantErr bool
	}{
		{input: "on", want: On()},
		{input: " ON ", want: On()},
		{input: "true", want: On()},
		{input: "enabled", want: On()},
		{input: "1", want: On()},
		{input: "off", want: Off()},
		{input: "false", want: Off()},
		{input: "disabled", want: Off()},
		{input: "0", want: Off()},
		{input: "25% user", want: Off().Percent(dimUser, 25)},
		{input: "25%user", want: Off().Percent(dimUser, 25)},
		{input: "12.5% of installation", want: Off().Percent(dimInstallation, 12.5)},
		{input: "100% host", want: Off().Percent(Host, 100)},
		{input: "150% user", want: Off().Percent(dimUser, 150), wantErr: false},
		{input: `{"default":true}`, want: On()},
		{input: "", wantErr: true},
		{input: "maybe", wantErr: true},
		{input: "25%", wantErr: true},
		{input: "25% user extra", wantErr: true},
		{input: "x% user", wantErr: true},
		{input: "{bad json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := ParseRollout(tt.input)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidRollout)

				return
			}

			require.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "got %s", got)
		})
	}
}
