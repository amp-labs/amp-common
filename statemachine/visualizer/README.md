# State Machine Visualizer

The visualizer package generates Mermaid diagrams from state machine configurations, making complex workflows easy to understand and document.

## Features

- **Automatic diagram generation** from YAML configs
- **Customizable output** with multiple display options
- **State highlighting** to show execution paths
- **Action and condition display** for detailed views
- **Multiple themes** for different contexts
- **Markdown-friendly** output for documentation

## Installation

```go
import "github.com/amp-labs/server/builder-mcp/statemachine/visualizer"
```

## Quick Start

### Basic Usage

```go
// Load and visualize a config file
diagram, err := visualizer.GenerateMermaidFromFile("workflow.yaml")
if err != nil {
    log.Fatal(err)
}

fmt.Println(diagram)
```

### From Config Object

```go
config := &statemachine.Config{
    InitialState: "start",
    States: []statemachine.State{
        {
            Name: "start",
            Actions: []statemachine.Action{
                {Type: "initialize"},
            },
            Transitions: []statemachine.Transition{
                {NextState: "process"},
            },
        },
        {
            Name:    "process",
            IsFinal: true,
        },
    },
}

diagram, err := visualizer.GenerateMermaid(config)
```

## Customization Options

### Direction

Control the flow direction of the diagram:

```go
opts := visualizer.DefaultOptions().WithDirection("LR") // Left-to-right
diagram, err := visualizer.GenerateMermaidWithOptions(config, opts)

// Options:
// "TD" - Top-down (default)
// "LR" - Left-to-right
```

### Show/Hide Details

Control what information is displayed:

```go
opts := visualizer.DefaultOptions().
    WithShowActions(true).      // Show action names in states
    WithShowConditions(true)    // Show transition conditions

diagram, err := visualizer.GenerateMermaidWithOptions(config, opts)
```

### Highlight Paths

Highlight specific states to show execution paths:

```go
opts := visualizer.DefaultOptions().
    WithHighlightPath([]string{"start", "validate", "complete"})

diagram, err := visualizer.GenerateMermaidWithOptions(config, opts)
```

### Themes

Choose a color scheme:

```go
opts := visualizer.DefaultOptions().WithTheme("dark")

// Available themes:
// "default" - Standard colors
// "dark"    - Dark mode
// "forest"  - Green theme
```

## Output Format

The visualizer generates Mermaid state diagrams in markdown format:

```markdown
```mermaid
stateDiagram-TD
    [*] --> start
    start: start\n[initialize]
    start --> process
    process --> [*]

    class start actionState
    class process finalState

    classDef actionState fill:#e1f5ff,stroke:#01579b,stroke-width:2px
    classDef finalState fill:#c8e6c9,stroke:#2e7d32,stroke-width:2px
```

```

This can be:
- Rendered directly in GitHub/GitLab markdown
- Viewed in VS Code with Mermaid extensions
- Converted to PNG/SVG with mermaid-cli
- Embedded in documentation

## State Styling

States are automatically styled based on their type:

- **Action States** (blue): States that execute actions
- **Final States** (green): Terminal states
- **Highlighted States** (yellow): States in the highlight path

## Examples

### Simple Linear Workflow

```go
config := &statemachine.Config{
    InitialState: "start",
    States: []statemachine.State{
        {
            Name: "start",
            Transitions: []statemachine.Transition{
                {NextState: "middle"},
            },
        },
        {
            Name: "middle",
            Transitions: []statemachine.Transition{
                {NextState: "end"},
            },
        },
        {
            Name:    "end",
            IsFinal: true,
        },
    },
}

diagram, _ := visualizer.GenerateMermaid(config)
// Generates: start -> middle -> end
```

### Branching Workflow

```go
config := &statemachine.Config{
    InitialState: "check",
    States: []statemachine.State{
        {
            Name: "check",
            Transitions: []statemachine.Transition{
                {
                    NextState: "success_path",
                    Condition: "result.success == true",
                },
                {
                    NextState: "error_path",
                    Condition: "result.success == false",
                },
            },
        },
        {
            Name:    "success_path",
            IsFinal: true,
        },
        {
            Name:    "error_path",
            IsFinal: true,
        },
    },
}

opts := visualizer.DefaultOptions().WithShowConditions(true)
diagram, _ := visualizer.GenerateMermaidWithOptions(config, opts)
// Shows conditions on transition arrows
```

### Complex Workflow with Actions

```go
config := &statemachine.Config{
    InitialState: "init",
    States: []statemachine.State{
        {
            Name: "init",
            Actions: []statemachine.Action{
                {Type: "loadData"},
                {Type: "validate"},
            },
            Transitions: []statemachine.Transition{
                {NextState: "process"},
            },
        },
        {
            Name: "process",
            Actions: []statemachine.Action{
                {Type: "transform"},
                {Type: "enrich"},
            },
            Transitions: []statemachine.Transition{
                {NextState: "complete"},
            },
        },
        {
            Name:    "complete",
            IsFinal: true,
        },
    },
}

opts := visualizer.DefaultOptions().
    WithShowActions(true).
    WithDirection("LR")

diagram, _ := visualizer.GenerateMermaidWithOptions(config, opts)
// Shows action names within state nodes
```

## Integration with Documentation

### In Markdown Files

```markdown
# My Workflow

Here's how the workflow operates:

<!-- Generate with: visualizer.GenerateMermaidFromFile("workflow.yaml") -->
```mermaid
stateDiagram-TD
    [*] --> start
    start --> complete
    complete --> [*]
```

```

### Automated Generation

```bash
# Using the CLI tool
statemachine-cli visualize workflow.yaml > workflow-diagram.md

# In Makefile
generate-docs:
    @for config in configs/*.yaml; do \
        statemachine-cli visualize $$config > docs/$$(basename $$config .yaml).md; \
    done
```

### In CI/CD

```yaml
# GitHub Actions example
- name: Generate Diagrams
  run: |
    make statemachine-cli
    make generate-diagrams

- name: Commit Updated Diagrams
  run: |
    git add docs/*.md
    git commit -m "Update workflow diagrams" || true
```

## Advanced Usage

### Custom Styling

Modify the generated diagram by post-processing:

```go
diagram, _ := visualizer.GenerateMermaid(config)

// Add custom class definitions
customDiagram := strings.Replace(diagram,
    "```\n",
    "    classDef custom fill:#ff0,stroke:#333\n```\n",
    1)
```

### Combining with Validation

```go
// Validate before visualizing
if err := validator.ValidateFile("workflow.yaml"); err != nil {
    log.Fatal("Invalid config:", err)
}

// Generate diagram for valid config
diagram, err := visualizer.GenerateMermaidFromFile("workflow.yaml")
```

### Exporting to Images

```bash
# Install mermaid-cli
npm install -g @mermaid-js/mermaid-cli

# Generate PNG
statemachine-cli visualize workflow.yaml | mmdc -i - -o workflow.png

# Generate SVG
statemachine-cli visualize workflow.yaml | mmdc -i - -o workflow.svg
```

## API Reference

### Functions

#### `GenerateMermaid(config *statemachine.Config) (string, error)`

Generates a Mermaid diagram from a config with default options.

#### `GenerateMermaidFromFile(path string) (string, error)`

Loads a config file and generates a Mermaid diagram.

#### `GenerateMermaidWithOptions(config *statemachine.Config, opts Options) (string, error)`

Generates a Mermaid diagram with custom options.

### Types

#### `Options`

```go
type Options struct {
    ShowActions    bool     // Include action details
    ShowConditions bool     // Show transition conditions
    Direction      string   // "TD" or "LR"
    HighlightPath  []string // States to highlight
    Theme          string   // Color theme
}
```

#### Option Builders

```go
DefaultOptions() Options
WithShowActions(bool) Options
WithShowConditions(bool) Options
WithDirection(string) Options
WithHighlightPath([]string) Options
WithTheme(string) Options
```

## Best Practices

1. **Version control diagrams**: Commit generated diagrams alongside configs
2. **Automate generation**: Use CI/CD to keep diagrams up-to-date
3. **Use highlights**: Show different execution paths for debugging
4. **Combine with validation**: Always validate before visualizing
5. **Document complex workflows**: Add diagrams to pull request descriptions

## Troubleshooting

### Diagram not rendering

- Ensure you're using a Mermaid-compatible viewer (GitHub, GitLab, VS Code)
- Check that the markdown code block is properly formatted
- Verify the diagram syntax is valid (run through validator)

### States not showing correctly

- Check that all referenced states exist in the config
- Verify state names don't contain special characters
- Ensure transitions reference valid states

### Styling not applied

- Confirm class definitions are included in output
- Check that theme option is valid
- Verify state types (action vs final) are set correctly

## See Also

- [State Machine README](../README.md)
- [Validator Package](../validator/README.md)
- [CLI Tool](../../../scripts/statemachine-cli/README.md)
- [Mermaid Documentation](https://mermaid.js.org/syntax/stateDiagram.html)
