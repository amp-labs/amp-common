# Package: statemachine/visualizer

Visualization tools for state machine workflows.

## Overview

See [README.md](./README.md) for visualization options and examples.

## Quick Reference

Generate visualizations:
- Graphviz DOT format
- Mermaid diagrams
- ASCII art state diagrams
- Interactive HTML views

## Example

```go
// Generate DOT diagram
dot, err := visualizer.ToDot(config)

// Generate Mermaid
mermaid, err := visualizer.ToMermaid(config)
```

## Related

- [README.md](./README.md) - Full visualization documentation
- `statemachine` - Core framework
