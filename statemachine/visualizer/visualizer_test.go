package visualizer

import (
	"strings"
	"testing"

	"github.com/amp-labs/amp-common/statemachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMermaid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      *statemachine.Config
		wantErr     bool
		wantContain []string
	}{
		{
			name: "simple linear workflow",
			config: &statemachine.Config{
				Name:         "simple",
				InitialState: "start",
				FinalStates:  []string{"complete"},
				States: []statemachine.StateConfig{
					{
						Name: "start",
						Type: "action",
						Actions: []statemachine.ActionConfig{
							{Type: "initialize", Name: "init"},
						},
					},
					{
						Name: "process",
						Type: "action",
						Actions: []statemachine.ActionConfig{
							{Type: "processData", Name: "proc"},
						},
					},
					{
						Name: "complete",
						Type: "final",
					},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "process", Condition: "always"},
					{From: "process", To: "complete", Condition: "always"},
				},
			},
			wantContain: []string{
				"stateDiagram-TD",
				"[*] --> start",
				"start --> process",
				"process --> complete",
				"complete --> [*]",
				"initialize",
				"processData",
			},
		},
		{
			name: "branching workflow with conditions",
			config: &statemachine.Config{
				Name:         "branching",
				InitialState: "start",
				FinalStates:  []string{"path_a", "path_b"},
				States: []statemachine.StateConfig{
					{
						Name: "start",
						Type: "action",
					},
					{
						Name: "path_a",
						Type: "final",
					},
					{
						Name: "path_b",
						Type: "final",
					},
				},
				Transitions: []statemachine.TransitionConfig{
					{
						From:      "start",
						To:        "path_a",
						Condition: "value > 10",
					},
					{
						From:      "start",
						To:        "path_b",
						Condition: "value <= 10",
					},
				},
			},
			wantContain: []string{
				"start --> path_a: value > 10",
				"start --> path_b: value <= 10",
				"path_a --> [*]",
				"path_b --> [*]",
			},
		},
		{
			name: "workflow with loop",
			config: &statemachine.Config{
				Name:         "loop",
				InitialState: "start",
				FinalStates:  []string{"complete"},
				States: []statemachine.StateConfig{
					{
						Name: "start",
						Type: "action",
					},
					{
						Name: "retry",
						Type: "action",
					},
					{
						Name: "complete",
						Type: "final",
					},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "retry", Condition: "always"},
					{
						From:      "retry",
						To:        "retry",
						Condition: "attempts < 3",
					},
					{
						From:      "retry",
						To:        "complete",
						Condition: "attempts >= 3",
					},
				},
			},
			wantContain: []string{
				"retry --> retry: attempts < 3",
				"retry --> complete: attempts >= 3",
			},
		},
		{
			name:    "nil config returns error",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config without initial state returns error",
			config: &statemachine.Config{
				Name:        "test",
				FinalStates: []string{"end"},
				States: []statemachine.StateConfig{
					{Name: "start", Type: "action"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GenerateMermaid(tt.config)

			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, "```mermaid")

			for _, want := range tt.wantContain {
				assert.Contains(t, result, want,
					"diagram should contain %q", want)
			}
		})
	}
}

func TestGenerateMermaidWithOptions(t *testing.T) {
	t.Parallel()

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "start",
		FinalStates:  []string{"complete"},
		States: []statemachine.StateConfig{
			{
				Name: "start",
				Type: "action",
				Actions: []statemachine.ActionConfig{
					{Type: "initialize", Name: "init"},
				},
			},
			{
				Name: "complete",
				Type: "final",
			},
		},
		Transitions: []statemachine.TransitionConfig{
			{
				From:      "start",
				To:        "complete",
				Condition: "ready",
			},
		},
	}

	tests := []struct {
		name           string
		opts           Options
		wantContain    []string
		wantNotContain []string
	}{
		{
			name: "left-right direction",
			opts: DefaultOptions().WithDirection("LR"),
			wantContain: []string{
				"stateDiagram-LR",
			},
		},
		{
			name: "hide actions",
			opts: DefaultOptions().WithShowActions(false),
			wantNotContain: []string{
				"initialize",
			},
		},
		{
			name: "hide conditions",
			opts: DefaultOptions().WithShowConditions(false),
			wantNotContain: []string{
				": ready",
			},
		},
		{
			name: "highlight path",
			opts: DefaultOptions().WithHighlightPath([]string{"start"}),
			wantContain: []string{
				"class start highlighted",
			},
		},
		{
			name: "always condition not shown as label",
			opts: DefaultOptions().WithShowConditions(true),
			wantNotContain: []string{
				": always",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GenerateMermaidWithOptions(config, tt.opts)
			require.NoError(t, err)

			for _, want := range tt.wantContain {
				assert.Contains(t, result, want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant)
			}
		})
	}
}

func TestOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultOptions()

	assert.True(t, opts.ShowActions)
	assert.True(t, opts.ShowConditions)
	assert.Equal(t, "TD", opts.Direction)
	assert.Equal(t, "default", opts.Theme)

	opts = opts.WithShowActions(false).
		WithShowConditions(false).
		WithDirection("LR").
		WithTheme("dark").
		WithHighlightPath([]string{"state1", "state2"})

	assert.False(t, opts.ShowActions)
	assert.False(t, opts.ShowConditions)
	assert.Equal(t, "LR", opts.Direction)
	assert.Equal(t, "dark", opts.Theme)
	assert.Equal(t, []string{"state1", "state2"}, opts.HighlightPath)
}

func TestGenerateMermaidStyling(t *testing.T) {
	t.Parallel()

	config := &statemachine.Config{
		Name:         "styling",
		InitialState: "start",
		FinalStates:  []string{"complete"},
		States: []statemachine.StateConfig{
			{
				Name: "start",
				Type: "action",
				Actions: []statemachine.ActionConfig{
					{Type: "action1", Name: "a1"},
				},
			},
			{
				Name: "complete",
				Type: "final",
			},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "start", To: "complete", Condition: "always"},
		},
	}

	result, err := GenerateMermaid(config)
	require.NoError(t, err)

	// Verify class definitions are present
	assert.Contains(t, result, "classDef actionState")
	assert.Contains(t, result, "classDef finalState")
	assert.Contains(t, result, "classDef highlighted")

	// Verify states are classified correctly
	assert.Contains(t, result, "class start actionState")
	assert.Contains(t, result, "class complete finalState")
}

func TestMermaidStructure(t *testing.T) {
	t.Parallel()

	config := &statemachine.Config{
		Name:         "structure",
		InitialState: "start",
		FinalStates:  []string{"start"},
		States: []statemachine.StateConfig{
			{
				Name: "start",
				Type: "final",
			},
		},
	}

	result, err := GenerateMermaid(config)
	require.NoError(t, err)

	// Verify proper markdown code block
	assert.True(t, strings.HasPrefix(result, "```mermaid\n"))
	assert.True(t, strings.HasSuffix(result, "```\n"))

	// Verify it contains core elements
	lines := strings.Split(result, "\n")
	assert.GreaterOrEqual(t, len(lines), 5, "should have multiple lines")
}
