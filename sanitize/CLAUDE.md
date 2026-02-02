# Package: sanitize

Functions to sanitize strings for use as filenames.

## Usage

```go
// Sanitize user input for filename
filename := sanitize.FileName("My Document: v2.0 (final).txt")
// Returns: "My_Document_v2.0_final.txt"

// Handles special characters
filename := sanitize.FileName("Über München & Co")
// Returns: "Ueber_Muenchen_and_Co"
```

## Common Patterns

- Convert user input to safe filenames
- Removes/replaces problematic characters (/, \, :, *, ?, etc.)
- Handles Unicode (removes accents, converts special chars)
- Collapses multiple underscores

## Gotchas

- Removes characters unsafe in Windows, shells, or URLs
- Converts accented characters (ä→ae, é→e)
- Currency symbols converted to words (€→Euro)
