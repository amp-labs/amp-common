package visualizer

// Options configures the visualization output.
type Options struct {
	// ShowActions includes action details in state nodes
	ShowActions bool

	// ShowConditions shows transition conditions as labels
	ShowConditions bool

	// Direction controls diagram flow: "TD" (top-down) or "LR" (left-right)
	Direction string

	// HighlightPath highlights a specific state path through the diagram
	HighlightPath []string

	// Theme controls the color scheme: "default", "dark", "forest"
	Theme string
}

// DefaultOptions returns sensible defaults for visualization.
func DefaultOptions() Options {
	return Options{
		ShowActions:    true,
		ShowConditions: true,
		Direction:      "TD",
		Theme:          "default",
	}
}

// WithShowActions enables/disables action details.
func (o Options) WithShowActions(show bool) Options {
	o.ShowActions = show

	return o
}

// WithShowConditions enables/disables transition conditions.
func (o Options) WithShowConditions(show bool) Options {
	o.ShowConditions = show

	return o
}

// WithDirection sets the diagram direction.
func (o Options) WithDirection(direction string) Options {
	o.Direction = direction

	return o
}

// WithHighlightPath sets states to highlight.
func (o Options) WithHighlightPath(path []string) Options {
	o.HighlightPath = path

	return o
}

// WithTheme sets the color theme.
func (o Options) WithTheme(theme string) Options {
	o.Theme = theme

	return o
}
